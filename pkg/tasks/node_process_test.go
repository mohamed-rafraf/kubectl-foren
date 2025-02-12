package tasks

import (
	"context"
	"testing"

	"github.com/mohamed-rafraf/kubectl-foren/pkg/state"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestNewDeployPodTask(t *testing.T) {
	// Create a fake client
	schema := runtime.NewScheme()
	corev1.AddToScheme(schema)
	k8sClient := ctrlfake.NewClientBuilder().WithScheme(schema).Build()

	// Initialize state
	s := &state.State{
		K8sClient: k8sClient,
		Context:   context.TODO(),
		Logger:    newTestLogger(),
	}

	// Define task
	podName := "test-pod"
	task := NewDeployPodTask(s, podName)

	// Run task
	err := task.Fn(s)
	assert.NoError(t, err)

	//Verify pod creation using the controller-runtime fake client
	pod := &corev1.Pod{}
	err = k8sClient.Get(context.TODO(), client.ObjectKey{Name: podName, Namespace: "default"}, pod)
	assert.NoError(t, err)
	assert.Equal(t, podName, pod.Name)
}

// Initialize a logrus logger
func newTestLogger() logrus.FieldLogger {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel) // Set log level to Debug for testing
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	return logger.WithField("component", "test")
}
