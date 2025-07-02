# Retry

Пакет `retry` предоставляет удобный механизм для повторного выполнения операций с экспоненциальным backoff и логированием.

## Установка

```bash
go get github.com/alfzs/retry
```

## Использование

Основная функция пакета - `WithRetry`, которая принимает контекст, конфигурацию, имя операции и саму операцию (функцию), которую нужно выполнить с повторными попытками.

### Пример

```go
package main

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/yourusername/retry"
)

func main() {
	ctx := context.Background()
	logger := slog.Default()

	config := retry.RetryConfig{
		MaxAttempts: 5,
		MinDelay:    100 * time.Millisecond,
		MaxDelay:    10 * time.Second,
		Logger:      logger,
	}

	result, err := retry.WithRetry(ctx, config, "example-operation", func(ctx context.Context) (string, error) {
		// Здесь ваша операция, которая может завершиться ошибкой
		return "", errors.New("temporary error")
	})

	if err != nil {
		logger.Error("Operation failed", slog.Any("error", err))
		return
	}

	logger.Info("Operation succeeded", slog.String("result", result))
}
```

## Конфигурация

`RetryConfig` позволяет настроить параметры повторных попыток:

- `MaxAttempts` - максимальное количество попыток (по умолчанию 3)
- `MinDelay` - минимальная задержка между попытками (по умолчанию 100ms)
- `MaxDelay` - максимальная задержка между попытками (по умолчанию 5s)
- `Logger` - логгер для записи информации о попытках (nil отключает логирование)

## Ошибки

При исчерпании всех попыток возвращается ошибка типа `RetryError`, которая содержит:

- Название операции
- Количество выполненных попыток
- Последнюю ошибку

## Зависимости

Пакет использует [github.com/alfzs/backoff](https://github.com/alfzs/backoff) для расчета экспоненциального backoff.

## Лицензия

MIT
