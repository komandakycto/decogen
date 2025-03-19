# Decogen - Decorator Pattern Generator for Go

Decogen is a code generation tool that automatically creates decorator implementations for Go interfaces. It helps eliminate boilerplate code when implementing patterns like retry, caching, metrics collection, and logging.

## Installation

```bash
go install github.com/clobucks/decogen/cmd/decogen@latest
```

## Usage

### Basic Usage with go:generate

Add a go:generate directive to your interface declaration:

```go
//go:generate decogen -interface=MyInterface -source=myfile.go -output=decorators/my_decorators.go -decorators=retry,logging

// MyInterface is an example interface
type MyInterface interface {
    DoSomething(ctx context.Context, param1 string) error
    GetData(ctx context.Context, id string) (*Data, error)
}
```

Then run:

```bash
go generate ./...
```

### Using a Configuration File

Create a config file (e.g., `decogen.json`):

```json
{
  "interface": {
    "name": "OperationsLogStorage",
    "source": "internal/service/postback/register.go"
  },
  "decorators": [
    {
      "name": "retry",
      "config": {
        "maxAttempts": 5,
        "backoff": {
          "minDelay": "100ms",
          "maxDelay": "5s",
          "factor": 2.0,
          "jitter": 0.1
        }
      }
    },
    {
      "name": "metrics",
      "config": {
        "namespace": "app",
        "subsystem": "operations"
      }
    }
  ],
  "output": "internal/storage/decorators/operations_log_decorators.go",
  "package": "decorators",
  "imports": [
    "github.com/myorg/myproject/internal/storage/models"
  ]
}
```

Then reference it with go:generate:

```go
//go:generate decogen -config=../../configs/decogen/operations_log.json
```

Or run the tool directly:

```bash
decogen -config=configs/decogen/operations_log.json
```

## Available Decorators

Decogen can generate the following decorator types:

### Retry Decorator

Automatically retries operations on failure with exponential backoff.

```
-decorators=retry
```

Dependencies:
- `github.com/sirupsen/logrus` for logging

### Metrics Decorator

Adds Prometheus metrics for method call count and duration.

```
-decorators=metrics
```

Dependencies:
- `github.com/prometheus/client_golang/prometheus` for metrics
- `github.com/sirupsen/logrus` for logging

### Cache Decorator

Adds in-memory caching for pure functions.

```
-decorators=cache
```

Dependencies:
- `github.com/sirupsen/logrus` for logging

### Logging Decorator

Adds structured logging for method calls.

```
-decorators=logging
```

Dependencies:
- `github.com/sirupsen/logrus` for logging

## Using Generated Decorators

The generated decorators follow a consistent pattern:

```go
// Create the base implementation
baseStorage := pg.NewLogStorage(db)

// Add retry capability
retryStorage := decorators.NewOperationsLogStorageWithRetry(
    baseStorage,
    backoff.DefaultBackOff(),
    logger,
    5, // maxAttempts
)

// Add metrics (composing decorators)
metricsStorage := decorators.NewOperationsLogStorageWithMetrics(
    retryStorage, // Use the retry decorator as the underlying implementation
    logger,
    "app",
    "operations",
)

// Use the fully decorated implementation
service := postback.NewOperationsLog(metricsStorage)
```

## Extending Decogen

To add custom decorator types, you need to:

1. Create a new template file in `internal/generator/templates/`
2. Update `internal/generator/generator.go` to include your template
3. Add your decorator type to `internal/config/config.go`

## License

MIT License - See LICENSE file for details