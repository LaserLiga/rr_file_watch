# Build and Verification

## Go Module

The module path is:

```text
github.com/LaserLiga/rr_file_watch
```

The package name used by the source files is:

```text
roadrunner
```

The module targets Go:

```text
1.26.3
```

## Local Verification

Run:

```sh
go test ./...
```

At the time these docs were updated, the module compiled successfully and reported no test files.

When running in a sandboxed environment, Go may need a writable build cache:

```sh
mkdir -p .cache/go-build
GOCACHE="$PWD/.cache/go-build" go test ./...
```

## RoadRunner Build

This plugin is built as part of the parent `LaserArenaControl` project, not from a Dockerfile in this directory.

The parent project Velox configuration is:

```text
../docker/velox_rr.toml
```

That configuration includes this plugin as:

```toml
[github.plugins.fileWatch]
ref = "master"
owner = "LaserLiga"
repository = "rr_file_watch"
```

The parent Docker image builds RoadRunner in `../docker/Dockerfile` by copying `docker/velox_rr.toml` and running:

```sh
vx build -c velox_rr.toml -o /usr/bin/
```

The current parent configuration targets the RoadRunner `v2025` plugin ecosystem.

The parent Velox configuration currently pins RoadRunner to:

```toml
[roadrunner]
ref = "v2025.1.13"
```
