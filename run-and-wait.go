package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	plugin_models "code.cloudfoundry.org/cli/plugin/models"

	"code.cloudfoundry.org/cli/plugin"
	"github.com/cloudfoundry/noaa/consumer"
	"github.com/cloudfoundry/sonde-go/events"
)

type runAndWait struct{}

type runTaskReq struct {
	Command string `json:"command"`
}

type runTaskResp struct {
	GUID string `json:"guid"`
	Name string `json:"name"`
}

type paginatedGetTasksResponse struct {
	Pagination struct {
		TotalResults int `json:"total_results"`
	} `json:"pagination"`
	Resources []runTaskResp `json:"resources"`
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

	log.Println("Kicking off task...")
	out, err := cliConnection.CliCommandWithoutTerminalOutput("curl", "-H", "Content-Type: application/json", "-d", string(b.Bytes()), "-X", "POST", fmt.Sprintf("/v3/apps/%s/tasks", app.Guid))
	if err != nil {
		return err
	}

	log.Println("Task started...")

	var tr runTaskResp
	err = json.NewDecoder(bytes.NewReader([]byte(strings.Join(out, "\n")))).Decode(&tr)
	if err != nil {
		return err
	}

	if tr.GUID == "" {
		return errors.New("Empty task ID")
	}

	return waitForCompletion(cliConnection, app.Guid, tr.GUID, tr.Name)
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
		log.Println(err)
		os.Exit(1)
	}
}

func getApp(cliConnection plugin.CliConnection, appName string) (*plugin_models.GetAppModel, error) {
	log.Println("Getting app id...")
	app, err := cliConnection.GetApp(appName)
	if err != nil {
		return nil, err
	}
	log.Println("App ID:", app.Guid)
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

	log.Println("Getting task id...")
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

	return waitForCompletion(cliConnection, app.Guid, gtr.Resources[0].GUID, gtr.Resources[0].Name)
}

func waitForCompletion(cliConnection plugin.CliConnection, appGUID, taskID, taskName string) error {
	log.Println("Task ID / Name:", taskID, " / ", taskName)

	targetSourceType := fmt.Sprintf("APP/TASK/%s", taskName)

	dopplerEndpoint, err := cliConnection.DopplerEndpoint()
	if err != nil {
		return err
	}
	token, err := cliConnection.AccessToken()
	if err != nil {
		return err
	}

	cons := consumer.New(dopplerEndpoint, nil, nil)
	defer cons.Close()

	messages, errorChannel := cons.TailingLogs(appGUID, token)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		for {
			select {
			case m := <-messages:
				if m.GetSourceType() == targetSourceType {
					switch m.GetMessageType() {
					case events.LogMessage_OUT:
						os.Stdout.Write(m.GetMessage())
						os.Stdout.WriteString("\n")
					case events.LogMessage_ERR:
						os.Stderr.Write(m.GetMessage())
						os.Stderr.WriteString("\n")
					}
				}
			case e := <-errorChannel:
				log.Println("error reading logs:", e)
			case <-ctx.Done():
				return
			}
		}
	}()

	sleepTime := time.Second
	maxSleep := time.Second * 30
	for {
		time.Sleep(sleepTime)

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

		switch ts.State {
		case "SUCCEEDED":
			return nil // happy

		case "FAILED":
			log.Println(fullS)
			return errors.New("task failed")

		default:
			sleepTime *= 2
			if sleepTime > maxSleep {
				sleepTime = maxSleep
			}
		}
	}
}

func (c *runAndWait) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "Run and Wait",
		Version: plugin.VersionType{
			Major: 0,
			Minor: 3,
			Build: 0,
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
