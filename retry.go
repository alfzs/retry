package retry

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/alfzs/backoff"
)

// Default значения для повторных попыток
const (
	DefaultMaxAttempts = 3
	DefaultMinDelay    = 100 * time.Millisecond
	DefaultMaxDelay    = 5 * time.Second
)

// RetryConfig содержит параметры для повторных попыток
type RetryConfig struct {
	MaxAttempts int           // Максимальное количество попыток
	MinDelay    time.Duration // Минимальная задержка
	MaxDelay    time.Duration // Максимальная задержка
	Logger      *slog.Logger  // Логгер (nil = логирование отключено)
}

// RetryError представляет ошибку после всех неудачных попыток
type RetryError struct {
	Operation string
	Attempts  int
	LastError error
}

func (e *RetryError) Error() string {
	return fmt.Sprintf("operation '%s' failed after %d attempts: %v",
		e.Operation, e.Attempts, e.LastError)
}

func (e *RetryError) Unwrap() error {
	return e.LastError
}

// WithRetry выполняет операцию с экспоненциальным backoff и повторными попытками.
func WithRetry[T any](
	ctx context.Context,
	config RetryConfig,
	operationName string,
	operationFn func(context.Context) (T, error),
) (T, error) {
	// Устанавливаем значения по умолчанию
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = DefaultMaxAttempts
	}
	if config.MinDelay <= 0 {
		config.MinDelay = DefaultMinDelay
	}
	if config.MaxDelay <= 0 {
		config.MaxDelay = DefaultMaxDelay
	}

	var result T
	var lastErr error

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		result, lastErr = operationFn(ctx)
		if lastErr == nil {
			if attempt > 1 && config.Logger != nil {
				config.Logger.Info("Operation succeeded after retry",
					slog.String("operation", operationName),
					slog.Int("attempt", attempt))
			}
			return result, nil
		}

		if config.Logger != nil {
			config.Logger.Error("Operation failed, will retry",
				slog.String("operation", operationName),
				slog.Int("attempt", attempt),
				slog.Int("max_attempt", config.MaxAttempts),
				slog.Any("error", lastErr))
		}

		if attempt == config.MaxAttempts {
			break
		}

		delay := backoff.CalculateExponentialBackoff(attempt, config.MinDelay, config.MaxDelay)

		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(delay):
		}
	}

	return result, &RetryError{
		Operation: operationName,
		Attempts:  config.MaxAttempts,
		LastError: lastErr,
	}
}
