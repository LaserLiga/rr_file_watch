# Metrics and Health

## Prometheus Namespace

All plugin metrics use the Prometheus namespace:

```text
rr_file_watch
```

## Plugin Metrics

| Metric                   | Type  | Description                                                           |
|--------------------------|-------|-----------------------------------------------------------------------|
| `rr_file_watch_events`   | gauge | Number of filesystem events registered by the plugin.                 |
| `rr_file_watch_jobs_ok`  | gauge | Number of notifications successfully processed by workers.            |
| `rr_file_watch_jobs_err` | gauge | Number of notifications that failed while being processed by workers. |

These values are stored as atomic counters in the plugin and exported as gauges.

## Worker Metrics

| Metric                               | Type  | Description                                       |
|--------------------------------------|-------|---------------------------------------------------|
| `rr_file_watch_total_workers`        | gauge | Total number of workers used by the plugin.       |
| `rr_file_watch_workers_memory_bytes` | gauge | Cumulative worker memory usage in bytes.          |
| `rr_file_watch_worker_state`         | gauge | Worker state metric labeled by `state` and `pid`. |
| `rr_file_watch_worker_memory_bytes`  | gauge | Memory usage for one worker, labeled by `pid`.    |
| `rr_file_watch_workers_ready`        | gauge | Number of workers currently in the ready state.   |
| `rr_file_watch_workers_working`      | gauge | Number of workers currently in the working state. |
| `rr_file_watch_workers_invalid`      | gauge | Number of workers in any other state.             |

## Status Check

`Status()` returns:

- `200 OK` when at least one worker process is active;
- `503 Service Unavailable` when no workers are active.

## Readiness Check

`Ready()` returns:

- `200 OK` when at least one worker is in the RoadRunner `ready` state;
- `503 Service Unavailable` when no workers are ready.
