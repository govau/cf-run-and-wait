# cf-run-and-wait

CloudFoundry CLI plugin to run a task, and wait for it to complete.

## Installing

```bash
git clone https://github.com/govau/cf-run-and-wait
cd cf-run-and-wait
go build -o run-and-wait
cf install-plugin ./run-and-wait
```

## Running a task

```bash
cf run-and-wait appname "echo foo"
```

If successful, will exit with status code of `0`.

If it fails, will print some debug info, and exit with non-zero status code.

Logs will be streamed while it runs.

## Building a new release

```bash
PLUGIN_NAME=run-and-wait

GOOS=linux   GOARCH=amd64   go build -o ${PLUGIN_NAME}.linux64
GOOS=linux   GOARCH=386     go build -o ${PLUGIN_NAME}.linux32
GOOS=windows GOARCH=amd64   go build -o ${PLUGIN_NAME}.win64
GOOS=windows GOARCH=386     go build -o ${PLUGIN_NAME}.win32
GOOS=darwin  GOARCH=amd64   go build -o ${PLUGIN_NAME}.osx

shasum -a 1 ${PLUGIN_NAME}.linux64
shasum -a 1 ${PLUGIN_NAME}.linux32
shasum -a 1 ${PLUGIN_NAME}.win64
shasum -a 1 ${PLUGIN_NAME}.win32
shasum -a 1 ${PLUGIN_NAME}.osx
```
