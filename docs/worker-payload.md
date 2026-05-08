# Worker Payload Contract

Each matching filesystem event is sent to the RoadRunner worker pool as a raw payload.

## Codec

```text
frame.CodecRaw
```

Workers should treat the request body as raw JSON bytes.

## JSON Shape

```json
{
  "directory": "./lmx/results",
  "file": "result.json",
  "op": "WRITE",
  "path": "lmx/results/result.json",
  "eventTime": "2026-05-08 12:34:56.789 +0200 CEST"
}
```

## Fields

| Field       | Type   | Description                                                                 |
|-------------|--------|-----------------------------------------------------------------------------|
| `directory` | string | Configured watch directory.                                                 |
| `file`      | string | Event file name from the watcher.                                           |
| `op`        | string | Watcher operation name, such as `CREATE`, `WRITE`, `RENAME`, or `MOVE`.     |
| `path`      | string | Event path from the watcher.                                                |
| `eventTime` | string | Event modification time formatted with Go's default `Time.String()` output. |

## Execution Timeout

Each worker execution receives a deadline of 10 seconds. If the worker does not complete in time, the execution is
counted as an error and logged.
