package retry

import (
	"fmt"
	"time"

	"github.com/user/autolink/internal/logging"
)

// WithExponentialBackoff executes an operation and retries it on failure with exponential backoff.
func WithExponentialBackoff(operationName string, maxRetries int, operation func() error) error {
	var err error
	delay := time.Second

	for i := 0; i < maxRetries; i++ {
		err = operation()
		if err == nil {
			return nil
		}

		if i < maxRetries-1 {
			logging.Logger.Warnf("%s failed: %v. Retrying in %v (Attempt %d/%d)...", operationName, err, delay, i+1, maxRetries)
			time.Sleep(delay)
			delay *= 2 // Exponential backoff
		}
	}

	return fmt.Errorf("%s failed after %d retries: %v", operationName, maxRetries, err)
}
