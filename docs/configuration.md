# Configuration

The plugin is enabled by adding a `file_watch` section to RoadRunner configuration.

## Options

| Option     | Type            | Default                  | Description                                                                                                                                                                                                      |
|------------|-----------------|--------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `dir`      | string          | `./lmx/results`          | Legacy single directory to watch. The directory must exist and must be a directory, not a file.                                                                                                                  |
| `dirs`     | string array    | empty                    | Additional/multiple directories to watch. When set without `dir`, only these directories are watched. When set with `dir`, duplicate entries are ignored.                                                        |
| `regexp`   | string          | empty                    | Optional regular expression filter applied to watched events. Empty means no regex filter.                                                                                                                       |
| `debounce` | duration string | `1s`                     | Quiet period before dispatching the latest event for a path. Repeated events inside this window reset the timer so partially written files are less likely to be imported early. Use `0s` to disable coalescing. |
| `pool`     | object          | RoadRunner pool defaults | Worker pool configuration passed to RoadRunner's static pool implementation.                                                                                                                                     |

The plugin refuses to start when:

- the `file_watch` configuration section is missing;
- no watch directories are configured after defaults are applied;
- any configured watch directory does not exist;
- any configured watch directory points to a non-directory path;
- `regexp` is set but cannot be compiled.
- `debounce` cannot be parsed as a non-negative Go duration.

## Example

```yaml
file_watch:
  dirs:
    - ./lmx/results
    - ./lmx6/results
  regexp: '.*\.json$'
  debounce: 1s
  pool:
    num_workers: 2
```

The exact pool options are RoadRunner pool options. This plugin passes the `pool` block directly into `server.NewPool`.

## Worker Environment

Workers started for this plugin receive:

```text
RR_MODE=file_watch
```

Application workers can use this environment variable to route execution to file-watch handling code.
