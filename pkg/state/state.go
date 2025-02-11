package state

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Option func(*State)

func New(ctx context.Context, opts ...Option) (*State, error) {
	s := &State{
		Context: ctx,
	}

	for _, opt := range opts {
		opt(s)
	}

	// Initialize RESTConfig if not provided via options
	var err error
	if s.RESTConfig == nil {
		s.RESTConfig, err = ctrl.GetConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get Kubernetes REST config: %w", err)
		}
	}
	if s.ClientSet == nil {
		// Create a kubernetes clientset from the RESTConfig
		s.ClientSet, err = kubernetes.NewForConfig(s.RESTConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create Kubernetes clientset: %w", err)
		}
	}

	// Initialize K8sClient if not provided via options
	if s.K8sClient == nil {
		s.K8sClient, err = client.New(s.RESTConfig, client.Options{})
		if err != nil {
			return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
		}
	}

	return s, nil
}

type State struct {
	Logger        logrus.FieldLogger
	Context       context.Context
	ClientSet     *kubernetes.Clientset
	RESTConfig    *rest.Config
	K8sClient     client.Client
	Verbose       bool
	PodName       string
	Namespace     string
	ContainerName string
}

// WithLogger sets a custom logger
func WithLogger(logger logrus.FieldLogger) Option {
	return func(s *State) {
		s.Logger = logger
	}
}

// WithClientSet sets a custom kubernetes clientset
func WithRESTConfig(config *kubernetes.Clientset) Option {
	return func(s *State) {
		s.ClientSet = config
	}
}

// WithK8sClient sets a custom Kubernetes client
func WithK8sClient(k8sClient client.Client) Option {
	return func(s *State) {
		s.K8sClient = k8sClient
	}
}
