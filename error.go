package retry

import "fmt"

// HTTPError представляет HTTP ошибку для повторных попыток
type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

func (e *HTTPError) Timeout() bool {
	// 408 Request Timeout - настоящий таймаут
	return e.StatusCode == 408
}

func (e *HTTPError) Temporary() bool {
	// 5xx - ошибки сервера, 429 - слишком много запросов
	return e.StatusCode >= 500 || e.StatusCode == 429
}
