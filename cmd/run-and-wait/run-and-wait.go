package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"code.cloudfoundry.org/cli/plugin"
	"code.cloudfoundry.org/cli/plugin/models"
)

type runAndWait struct{}

type runTaskReq struct {
	Command string `json:"command"`
}

type runTaskResp struct {
	GUID string `json:"guid"`
}

type paginatedGetTasksResponse struct {
	Pagination struct {
		TotalResults int `json:"total_results"`
	} `json:"pagination"`
	Resources []struct {
		GUID string `json:"guid"`
	} `json:"resources"`
}

type taskStatus struct {
	State string `json:"state"`
}

func doRunAndWait(cliConnection plugin.CliConnection, args []string) error {
	if len(args) != 3 {
		return errors.New("Expected 2 args: APPNAME cmd")
	}
	appName := args[1]
	cmd := args[2]

	app, err := getApp(cliConnection, appName)
	if err != nil {
		return err
	}

	b := &bytes.Buffer{}
	err = json.NewEncoder(b).Encode(&runTaskReq{Command: cmd})
	if err != nil {
		return err
	}

	fmt.Println("Kicking off task...")
	out, err := cliConnection.CliCommandWithoutTerminalOutput("curl", "-H", "Content-Type: application/json", "-d", string(b.Bytes()), "-X", "POST", fmt.Sprintf("/v3/apps/%s/tasks", app.Guid))
	if err != nil {
		return err
	}

	fmt.Println("Task started...")

	var tr runTaskResp
	err = json.NewDecoder(bytes.NewReader([]byte(strings.Join(out, "\n")))).Decode(&tr)
	if err != nil {
		return err
	}

	if tr.GUID == "" {
		return errors.New("Empty task ID")
	}

	return waitForCompletion(cliConnection, tr.GUID)
}

func (c *runAndWait) Run(cliConnection plugin.CliConnection, args []string) {
	var err error
	switch args[0] {
	case "run-and-wait":
		err = doRunAndWait(cliConnection, args)
	case "wait":
		err = doWait(cliConnection, args)
	}
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func getApp(cliConnection plugin.CliConnection, appName string) (*plugin_models.GetAppModel, error) {
	fmt.Println("Getting app id...")
	app, err := cliConnection.GetApp(appName)
	if err != nil {
		return nil, err
	}
	fmt.Println("App ID:", app.Guid)
	return &app, nil
}

func doWait(cliConnection plugin.CliConnection, args []string) error {
	if len(args) != 3 {
		return errors.New("Expected 2 args: APPNAME TASK")
	}
	appName := args[1]
	task := args[2]

	app, err := getApp(cliConnection, appName)
	if err != nil {
		return err
	}

	fmt.Println("Getting task id...")
	out, err := cliConnection.CliCommandWithoutTerminalOutput("curl", "-H", "Content-Type: application/json", fmt.Sprintf("/v3/apps/%s/tasks?names=%s", app.Guid, url.QueryEscape(task)))
	if err != nil {
		return err
	}

	var gtr paginatedGetTasksResponse
	err = json.NewDecoder(bytes.NewReader([]byte(strings.Join(out, "\n")))).Decode(&gtr)
	if err != nil {
		return err
	}

	if gtr.Pagination.TotalResults != 1 {
		return fmt.Errorf("Invalid number of tasks found for name %s: %d", task, gtr.Pagination.TotalResults)
	}

	if gtr.Resources[0].GUID == "" {
		return errors.New("Empty task ID")
	}

	return waitForCompletion(cliConnection, gtr.Resources[0].GUID)
}

func waitForCompletion(cliConnection plugin.CliConnection, taskID string) error {
	fmt.Println("Task ID:", taskID)

	sleepTime := time.Second
	for {
		fmt.Printf("Sleeping for %0.0f seconds...\n", float64(sleepTime)/float64(time.Second))
		time.Sleep(sleepTime)

		fmt.Println("Getting task status...")
		out, err := cliConnection.CliCommandWithoutTerminalOutput("curl", fmt.Sprintf("/v3/tasks/%s", taskID))
		if err != nil {
			return err
		}

		fullS := strings.Join(out, "\n")

		var ts taskStatus
		err = json.NewDecoder(bytes.NewReader([]byte(fullS))).Decode(&ts)
		if err != nil {
			return err
		}

		fmt.Println("Result:", ts.State)

		switch ts.State {
		case "SUCCEEDED":
			return nil // happy

		case "FAILED":
			fmt.Println(fullS)
			return errors.New("task failed")

		default:
			sleepTime *= 2
		}
	}
}

func (c *runAndWait) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "Run and Wait",
		Version: plugin.VersionType{
			Major: 0,
			Minor: 2,
			Build: 1,
		},
		MinCliVersion: plugin.VersionType{
			Major: 6,
			Minor: 7,
			Build: 0,
		},
		Commands: []plugin.Command{
			{
				Name:     "run-and-wait",
				HelpText: "Run task, and wait until complete.",
				UsageDetails: plugin.Usage{
					Usage: "run-and-wait\n   cf run-and-wait APPNAME \"cmd to run\"",
				},
			},
			{
				Name:     "wait",
				HelpText: "Wait for an existing task",
				UsageDetails: plugin.Usage{
					Usage: "wait\n   cf wait APPNAME TASK",
				},
			},
		},
	}
}

func main() {
	plugin.Start(&runAndWait{})
}
