# rr_file_watch

`rr_file_watch` is a RoadRunner plugin that watches configured directories for file changes and sends matching
filesystem events to a RoadRunner worker pool.

The plugin is registered under the RoadRunner configuration key `file_watch`. When enabled, it:

- validates the configured watch directories and optional regular expression;
- starts a static RoadRunner worker pool with `RR_MODE=file_watch` in the worker environment;
- watches the configured directories for file create, write, rename, and move events;
- serializes each event as raw JSON;
- submits the JSON payload to the worker pool with a 10 second execution deadline;
- exports Prometheus metrics for events, worker jobs, worker states, and worker memory;
- participates in RoadRunner status and readiness checks.

## Repository Layout

| Path            | Purpose                                                                                            |
|-----------------|----------------------------------------------------------------------------------------------------|
| `plugin.go`     | Plugin lifecycle: initialization, validation, pool creation, reset, stop, and worker state access. |
| `listener.go`   | Filesystem watcher loop and worker payload dispatch.                                               |
| `config.go`     | `file_watch` configuration model and defaults.                                                     |
| `metrics.go`    | Prometheus collector implementation for file events, jobs, and worker state.                       |
| `status.go`     | RoadRunner health and readiness checks.                                                            |
| `interfaces.go` | Local interfaces for RoadRunner services used by the plugin.                                       |
| `go.mod`        | Go module definition and dependencies.                                                             |

## Main Dependencies

- `github.com/roadrunner-server/api/v4`: RoadRunner plugin API integration.
- `github.com/roadrunner-server/pool`: worker pool creation and execution.
- `github.com/roadrunner-server/goridge/v3`: raw payload codec.
- `github.com/radovskyb/watcher`: polling filesystem watcher.
- `github.com/prometheus/client_golang`: Prometheus metrics.
- `go.uber.org/zap`: structured logging.

## Documentation

- [Configuration](configuration.md)
- [Runtime Behavior](runtime.md)
- [Worker Payload Contract](worker-payload.md)
- [Metrics and Health](metrics-and-health.md)
- [Build and Verification](build-and-verification.md)
