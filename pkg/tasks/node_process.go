package tasks

import (
	"context"
	"fmt"
	"os"
	"time"

	"os/signal"
	"syscall"

	"github.com/creack/pty"
	"github.com/mohamed-rafraf/kubectl-foren/pkg/state"
	terminal "golang.org/x/term"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewDeletePodTask(s *state.State, podName string) Task {
	return Task{
		Description: "Delete temporary pod",
		Fn: func(s *state.State) error {
			s.Logger.Debug("Deleting pod ", podName)
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      podName,
					Namespace: "default",
				},
			}
			err := s.K8sClient.Delete(s.Context, pod)
			if err != nil {
				s.Logger.Error(err, "Failed to delete pod ", podName)
				return fmt.Errorf("failed to delete pod %s: %w", podName, err)
			}

			s.Logger.Debug("Pod deleted successfully ", podName)
			return nil
		},
		Retries: 1,
		Timeout: 15 * time.Second,
	}
}

func NewExecuteTopTask(s *state.State, podName string) Task {
	return Task{
		Description: "Execute top inside pod",
		Fn: func(s *state.State) error {
			s.Logger.Debug("Executing top inside pod ", podName)

			clientset, err := kubernetes.NewForConfig(s.RESTConfig)
			if err != nil {
				return fmt.Errorf("failed to create clientset: %w", err)
			}

			// Set up terminal
			oldState, err := terminal.MakeRaw(int(os.Stdin.Fd()))
			if err != nil {
				return fmt.Errorf("failed to set raw terminal: %w", err)
			}
			defer terminal.Restore(int(os.Stdin.Fd()), oldState)

			// Create exec request
			req := clientset.CoreV1().RESTClient().Post().
				Resource("pods").
				Name(podName).
				Namespace("default").
				SubResource("exec").
				VersionedParams(&corev1.PodExecOptions{
					Container: "disk-access",
					Command:   []string{"top"},
					Stdin:     true,
					Stdout:    true,
					Stderr:    true,
					TTY:       true,
				}, scheme.ParameterCodec)

			executor, err := remotecommand.NewSPDYExecutor(s.RESTConfig, "POST", req.URL())
			if err != nil {
				return fmt.Errorf("failed to create executor: %w", err)
			}

			// Handle terminal resize
			resize := make(chan remotecommand.TerminalSize)
			go func() {
				for {
					winSize, err := pty.GetsizeFull(os.Stdin)
					if err != nil {
						s.Logger.Error(err, "Failed to get terminal size")
						return
					}
					resize <- remotecommand.TerminalSize{
						Width:  uint16(winSize.Cols),
						Height: uint16(winSize.Rows),
					}
				}
			}()

			// Set up signal handling
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigChan
				s.Logger.Debug("Received interrupt, closing session...")
				cancel()
				terminal.Restore(int(os.Stdin.Fd()), oldState)
				os.Exit(0)
			}()

			// Execute with context and terminal resize support
			err = executor.StreamWithContext(ctx, remotecommand.StreamOptions{
				Stdin:  os.Stdin,
				Stdout: os.Stdout,
				Stderr: os.Stderr,
				Tty:    true,
			})

			if err != nil {
				if ctx.Err() == context.Canceled {
					return fmt.Errorf("session interrupted")
				}
				return fmt.Errorf("stream error: %w", err)
			}

			return nil
		},
		Retries: 1,
		Timeout: 0,
	}
}

func NewDeployPodTask(s *state.State, podName string) Task {
	return Task{
		Description: "Deploy privileged pod on node",
		Fn: func(s *state.State) error {
			s.Logger.Debug("Deploying pod ", podName)

			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      podName,
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					HostNetwork: true,
					HostPID:     true,
					NodeName:    podName,
					Containers: []corev1.Container{
						{
							Name:    "disk-access",
							Image:   "ubuntu", // use the appropriate image
							Command: []string{"/bin/sh", "-c", "sleep 3600"},
							SecurityContext: &corev1.SecurityContext{
								Privileged: BoolPtr(true),
							},
							Env: []corev1.EnvVar{ // Set TERM for proper styling
								{
									Name:  "TERM",
									Value: "xterm-256color",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "dev-volume",
									MountPath: "/dev",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "dev-volume",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/dev",
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			}

			if err := s.K8sClient.Create(s.Context, pod); err != nil {
				s.Logger.Error(err, "Failed to deploy pod ", podName)
				return fmt.Errorf("failed to deploy pod on node %s: %w", podName, err)
			}

			s.Logger.Debug("Pod deployed successfully ", podName)
			return nil
		},
		Retries: 3,
		Timeout: 30 * time.Second,
	}
}

func BoolPtr(b bool) *bool {
	return &b
}

// NewWaitForPodRunningTask creates a task that waits for a pod to reach the Running state.
func NewWaitForPodRunningTask(s *state.State, podName string) Task {
	return Task{
		Description: "Wait for pod to be running",
		Fn: func(s *state.State) error {
			s.Logger.Debug("Waiting for pod to be running ", podName)

			// Define a timeout context to control how long we wait
			ctx, cancel := context.WithTimeout(s.Context, 60*time.Second)
			defer cancel()

			var pod corev1.Pod
			for {
				err := s.K8sClient.Get(ctx, client.ObjectKey{Namespace: "default", Name: podName}, &pod)
				if err != nil {
					s.Logger.Error(err, "Failed to get pod status", "podName", podName)
					return fmt.Errorf("failed to get pod %s status: %w", podName, err)
				}

				if pod.Status.Phase == corev1.PodRunning {
					s.Logger.Debug("Pod is now running ", podName)
					return nil
				}

				// Check if we've timed out
				select {
				case <-ctx.Done():
					s.Logger.Error(ctx.Err(), "Timed out waiting for pod to be running", "podName", podName)
					return fmt.Errorf("timed out waiting for pod %s to be running", podName)
				case <-time.After(2 * time.Second): // Poll every 2 seconds
					s.Logger.Debug("Pod not yet running, retrying... ", podName)
				}
			}
		},
		Retries: 5,
		Timeout: 60 * time.Second,
	}
}

// NewExecutePsAuxTask creates a task to execute "ps aux" in a specified pod.
func NewExecutePsAuxTask(s *state.State, podName string) Task {
	return Task{
		Description: "Execute ps aux inside pod",
		Fn: func(s *state.State) error {
			s.Logger.Debug("Executing ps aux inside pod ", podName)

			// Create a Kubernetes clientset from the RESTConfig
			clientset, err := kubernetes.NewForConfig(s.RESTConfig)
			if err != nil {
				return fmt.Errorf("failed to create Kubernetes clientset: %w", err)
			}

			req := clientset.CoreV1().RESTClient().Post().
				Resource("pods").
				Name(podName).
				Namespace("default").
				SubResource("exec").
				VersionedParams(&corev1.PodExecOptions{
					Container: "disk-access",
					Command:   []string{"tmux", "new", "-d", "htop"},
					Stdin:     false,
					Stdout:    true,
					Stderr:    true,
					TTY:       true,
				}, scheme.ParameterCodec)

			executor, err := remotecommand.NewSPDYExecutor(s.RESTConfig, "POST", req.URL())
			if err != nil {
				return fmt.Errorf("failed to create SPDY executor: %w", err)
			}

			err = executor.Stream(remotecommand.StreamOptions{
				Stdin:  os.Stdin,
				Stdout: os.Stdout,
				Stderr: os.Stderr,
				Tty:    true,
			})
			if err != nil {
				s.Logger.Error(err, "Failed to execute ps aux inside pod ", podName)
				return fmt.Errorf("failed to execute ps aux in pod %s: %w", podName, err)
			}

			s.Logger.Debug("ps aux executed successfully inside pod ", podName)
			return nil
		},
		Retries: 1,
		Timeout: 30 * time.Second, // Timeout for this specific task
	}
}
