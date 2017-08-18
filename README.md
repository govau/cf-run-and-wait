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

## Building a new release

```bash
PLUGIN_PATH=$GOPATH/src/github.com/govau/cf-run-and-wait/cmd/run-and-wait
PLUGIN_NAME=$(basename $PLUGIN_PATH)
cd $PLUGIN_PATH

GOOS=linux GOARCH=amd64 go build -o ${PLUGIN_NAME}.linux64
GOOS=linux GOARCH=386 go build -o ${PLUGIN_NAME}.linux32
GOOS=windows GOARCH=amd64 go build -o ${PLUGIN_NAME}.win64
GOOS=windows GOARCH=386 go build -o ${PLUGIN_NAME}.win32
GOOS=darwin GOARCH=amd64 go build -o ${PLUGIN_NAME}.osx

shasum -a 1 ${PLUGIN_NAME}.linux64
shasum -a 1 ${PLUGIN_NAME}.linux32
shasum -a 1 ${PLUGIN_NAME}.win64
shasum -a 1 ${PLUGIN_NAME}.win32
shasum -a 1 ${PLUGIN_NAME}.osx
```
