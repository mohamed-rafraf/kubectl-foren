package tasks

import (
	"strings"
	"time"

	"github.com/mohamed-rafraf/kubectl-foren/pkg/state"
	"k8s.io/apimachinery/pkg/util/wait"
)

// defaultRetryBackoff is backoff with with duration of 5 seconds and factor of 2.0
func defaultRetryBackoff(retries int) wait.Backoff {
	return wait.Backoff{
		Steps:    retries,
		Duration: 10 * time.Second,
		Factor:   1.4,
	}
}

type Task struct {
	Fn            func(s *state.State) error
	Predicate     func(s *state.State) bool
	Description   string
	Operation     string
	Retries       int
	Timeout       time.Duration
	OutputHandler func(output string)
}

// Run runs a task
func (t *Task) Run(s *state.State) error {
	if t.Retries == 0 {
		t.Retries = 10
	}

	backoff := defaultRetryBackoff(t.Retries)

	var lastError error
	err := wait.ExponentialBackoff(backoff, func() (bool, error) {
		if lastError != nil {
			s.Logger.Warn("Retrying task...")
		}

		lastError = t.Fn(s)
		if lastError != nil {
			s.Logger.Warnf("Task failed, error was: %s", strings.ReplaceAll(lastError.Error(), "\\n", "\n"))

			return false, nil
		}

		return true, nil
	})

	if wait.Interrupted(err) {
		err = lastError
	}

	return err
}
