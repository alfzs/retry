package retry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
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
	MaxAttempts int              // Максимальное количество попыток
	MinDelay    time.Duration    // Минимальная задержка
	MaxDelay    time.Duration    // Максимальная задержка
	Logger      *slog.Logger     // Логгер (nil = логирование отключено)
	ShouldRetry func(error) bool // Определяет, стоит ли повторять
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
	if config.ShouldRetry == nil {
		config.ShouldRetry = shouldRetryError
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

		// Проверка — повторять ли эту ошибку
		if config.ShouldRetry != nil && !config.ShouldRetry(lastErr) {
			if config.Logger != nil {
				config.Logger.Warn("Retry aborted due to non-retriable error",
					slog.String("operation", operationName),
					slog.Int("attempt", attempt),
					slog.Any("error", lastErr))
			}
			break
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

// shouldRetryError определяет, стоит ли повторять операцию при данной ошибке
func shouldRetryError(err error) bool {
	if err == nil {
		return false
	}

	// Контекст отменён или дедлайн — не повторяем
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Проверяем сетевые ошибки
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return true
		}
	}

	// Ошибки операционной сети
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}

	// URL ошибки
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return true
	}

	// HTTP ошибки 5xx и 429
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.Temporary()
	}

	return false
}
