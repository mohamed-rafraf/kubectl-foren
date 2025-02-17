package tasks

import (
	"context"
	"testing"

	"github.com/mohamed-rafraf/kubectl-foren/pkg/state"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// Initialize a logrus logger
func newTestLogger() logrus.FieldLogger {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel) // Set log level to Debug for testing
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	return logger.WithField("component", "test")
}

func TestDeployDeletePod(t *testing.T) {
	//Create a scheme and register required types
	schema := runtime.NewScheme()
	err := corev1.AddToScheme(schema)
	assert.NoError(t, err)

	//Create a fake controller-runtime client
	k8sClient := fake.NewClientBuilder().
		WithScheme(schema). // Use the updated scheme
		Build()

	//Initialize state
	ctx := context.TODO()
	s := &state.State{
		K8sClient:  k8sClient,
		RESTConfig: &rest.Config{}, // Dummy REST config for testing
		Context:    ctx,
		Logger:     newTestLogger(),
	}

	//Define pod name
	podName := "test-pod"

	//Deploy the pod
	deployTask := DeloyForenPod(s, podName)
	err = deployTask.Fn(s)
	assert.NoError(t, err)

	//Verify pod creation
	pod := &corev1.Pod{}
	err = k8sClient.Get(ctx, client.ObjectKey{Name: podName, Namespace: "default"}, pod)
	assert.NoError(t, err)
	assert.Equal(t, podName, pod.Name)

	//Delete the pod
	deleteTask := DeleteForenPod(s, podName)
	err = deleteTask.Fn(s)
	assert.NoError(t, err)

	//Verify pod deletion
	err = k8sClient.Get(ctx, client.ObjectKey{Name: podName, Namespace: "default"}, pod)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
