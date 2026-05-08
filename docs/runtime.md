# Runtime Behavior

## Startup

During `Init`, the plugin:

1. Checks whether the RoadRunner configuration contains the `file_watch` section.
2. Unmarshals that section into `Config`.
3. Applies defaults, including `dir: ./lmx/results` when neither `dir` nor `dirs` is provided.
4. Stores the RoadRunner server and logger dependencies.
5. Creates the Prometheus stats exporter.

During `Serve`, the plugin:

1. Validates all configured watch directories, skipping missing paths with warnings.
2. Validates the configured regular expression, when present.
3. Creates a RoadRunner static worker pool.
4. Starts the filesystem listener goroutine.

## Filesystem Watching

The listener uses `github.com/radovskyb/watcher` and polls every 100 milliseconds. Every configured directory is added
to the same watcher instance. Missing directories and non-directory paths are skipped; startup fails only when no
configured directory is usable.

The plugin filters for these filesystem operations:

- `Create`
- `Write`
- `Rename`
- `Move`

When `regexp` is configured, the watcher adds a regex filter hook. Only matching events are delivered to workers.

## Event Processing

For each watcher event, the plugin:

1. Builds an event details object.
2. Increments the `events` metric.
3. Coalesces repeated events for the same path until the configured debounce window is quiet.
4. Marshals the latest event details to JSON.
5. Wraps the JSON in a RoadRunner raw payload.
6. Executes the payload on the worker pool with a 10 second deadline.
7. Reads the worker response and increments either the successful job counter or failed job counter.

The worker execution path is protected by the plugin mutex so a pool reset cannot mutate the pool while an event is
being submitted.

Worker response handling treats `OK` as success and `ERROR`, response-level errors, empty responses, nil responses, and
unexpected response bodies as failed jobs.

The debounce behavior is intentionally delay-based rather than skip-based. If a file receives a create event followed by
write events, the plugin dispatches only the latest event after the path has been quiet for the configured duration.
This avoids triggering the import while the result file is still being written.

## Reset and Stop

`Reset` calls `workersPool.Reset(context.Background())`, replacing the current workers.

`Stop` closes the filesystem watcher and then closes the plugin stop channel. The method is idempotent, so repeated stop
calls are safe.
