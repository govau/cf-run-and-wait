# cf-run-and-wait

CloudFoundry CLI plugin to run a task, and wait for it to complete.

## Installing

```bash
go get github.com/govau/cf-run-and-wait/cmd/run-and-wait
cf install-plugin $GOPATH/bin/run-and-wait
```

## Running a task

```bash
cf run-and-wait appname "echo foo"
```

If successful, will exit with status code of `0`.

If it fails, will print some debug info, and exit with non-zero status code.
