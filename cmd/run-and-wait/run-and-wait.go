package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"code.cloudfoundry.org/cli/plugin"
)

type runAndWait struct{}

type runTaskReq struct {
	Command string `json:"command"`
}

type runTaskResp struct {
	Guid string `json:"guid"`
}

type taskStatus struct {
	State string `json:"state"`
}

func (c *runAndWait) Run(cliConnection plugin.CliConnection, args []string) {
	if args[0] == "run-and-wait" {
		if len(args) != 3 {
			fmt.Println("Expected 2 args: APPNAME cmd")
			os.Exit(1)
		}
		appName := args[1]
		cmd := args[2]

		fmt.Println("Getting app id...")
		app, err := cliConnection.GetApp(appName)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		b := &bytes.Buffer{}
		err = json.NewEncoder(b).Encode(&runTaskReq{Command: cmd})
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Println("Kicking off task...")
		out, err := cliConnection.CliCommandWithoutTerminalOutput("curl", "-H", "Content-Type: application/json", "-d", string(b.Bytes()), "-X", "POST", fmt.Sprintf("/v3/apps/%s/tasks", app.Guid))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Println("Task started...")

		var tr runTaskResp
		err = json.NewDecoder(bytes.NewReader([]byte(strings.Join(out, "\n")))).Decode(&tr)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if tr.Guid == "" {
			fmt.Println("Empty task ID")
			os.Exit(1)
		}

		fmt.Println("Task ID:", tr.Guid)

		sleepTime := time.Second
		for {
			fmt.Printf("Sleeping for %0.0f seconds...\n", float64(sleepTime)/float64(time.Second))
			time.Sleep(sleepTime)

			fmt.Println("Getting task status...")
			out, err = cliConnection.CliCommandWithoutTerminalOutput("curl", fmt.Sprintf("/v3/tasks/%s", tr.Guid))
			if err != nil {
				os.Exit(1)
			}

			fullS := strings.Join(out, "\n")

			var ts taskStatus
			err = json.NewDecoder(bytes.NewReader([]byte(fullS))).Decode(&ts)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			fmt.Println("Result:", ts.State)

			switch ts.State {
			case "SUCCEEDED":
				return // happy

			case "FAILED":
				fmt.Println(fullS)
				os.Exit(1)

			default:
				sleepTime *= 2
			}
		}
	}
}

func (c *runAndWait) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "Run and Wait",
		Version: plugin.VersionType{
			Major: 0,
			Minor: 1,
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
		},
	}
}

func main() {
	plugin.Start(&runAndWait{})
}
