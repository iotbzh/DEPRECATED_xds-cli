/*
 * Copyright (C) 2017 "IoT.bzh"
 * Author Sebastien Douheret <sebastien@iot.bzh>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package main

import (
	"fmt"
	"strings"

	"github.com/iotbzh/xds-agent/lib/xaapiv1"
	"github.com/urfave/cli"
)

func initCmdProjects(cmdDef *[]cli.Command) {
	*cmdDef = append(*cmdDef, cli.Command{
		Name:     "projects",
		Aliases:  []string{"prj"},
		HideHelp: true,
		Usage:    "project commands group",
		Subcommands: []cli.Command{
			{
				Name:    "add",
				Aliases: []string{"a"},
				Usage:   "Add a new project",
				Action:  projectsAdd,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "label, l",
						Usage: "project label (free form string)",
					},
					cli.StringFlag{
						Name:  "path, p",
						Usage: "project local path",
					},
					cli.StringFlag{
						Name:  "server-path, sp",
						Usage: "project server path (only used with pathmap type)",
					},
					cli.StringFlag{
						Name:  "type, t",
						Usage: "project type (pathmap|pm, cloudsync|sc)",
					},
				},
			},
			{
				Name:   "get",
				Usage:  "Get a property of a project",
				Action: projectsGet,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:   "id",
						Usage:  "project id",
						EnvVar: "XDS_PROJECT_ID",
					},
				},
			},
			{
				Name:    "list",
				Aliases: []string{"ls"},
				Usage:   "List existing projects",
				Action:  projectsList,
				Flags: []cli.Flag{
					cli.BoolFlag{
						Name:  "verbose, v",
						Usage: "display verbose output",
					},
				},
			},
			{
				Name:    "remove",
				Aliases: []string{"rm"},
				Usage:   "Remove an existing project",
				Action:  projectsRemove,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:   "id",
						Usage:  "project id",
						EnvVar: "XDS_PROJECT_ID",
					},
				},
			},
			{
				Name:    "sync",
				Aliases: []string{},
				Usage:   "Force synchronization of project sources",
				Action:  projectsSync,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:   "id",
						Usage:  "project id",
						EnvVar: "XDS_PROJECT_ID",
					},
				},
			},
		},
	})
}

func projectsList(ctx *cli.Context) error {
	// Get projects list
	prjs := []xaapiv1.ProjectConfig{}
	if err := ProjectsListGet(&prjs); err != nil {
		return cli.NewExitError(err.Error(), 1)
	}
	_displayProjects(prjs, ctx.Bool("verbose"))
	return nil
}

func projectsGet(ctx *cli.Context) error {
	id := GetID(ctx)
	if id == "" {
		return cli.NewExitError("id parameter or option must be set", 1)
	}
	prjs := make([]xaapiv1.ProjectConfig, 1)
	if err := HTTPCli.Get("/projects/"+id, &prjs[0]); err != nil {
		return cli.NewExitError(err, 1)
	}
	_displayProjects(prjs, true)
	return nil
}

func _displayProjects(prjs []xaapiv1.ProjectConfig, verbose bool) {
	// Display result
	first := true
	writer := NewTableWriter()
	for _, folder := range prjs {
		if verbose {
			if !first {
				fmt.Fprintln(writer)
			}
			fmt.Fprintln(writer, "ID:\t", folder.ID)
			fmt.Fprintln(writer, "Label:\t", folder.Label)
			fmt.Fprintln(writer, "Path type:\t", folder.Type)
			fmt.Fprintln(writer, "Local Path:\t", folder.ClientPath)
			if folder.Type != xaapiv1.TypeCloudSync {
				fmt.Fprintln(writer, "Server Path:\t", folder.ServerPath)
			}
			fmt.Fprintln(writer, "Status:\t", folder.Status)
			fmt.Fprintln(writer, "Is in Sync:\t", folder.IsInSync)
			ds := folder.DefaultSdk
			if ds == "" {
				ds = "-"
			}
			fmt.Fprintln(writer, "Default Sdk:\t", ds)

		} else {
			if first {
				fmt.Fprintln(writer, "ID\t Label\t LocalPath")
			}
			fmt.Fprintln(writer, folder.ID, "\t", folder.Label, "\t", folder.ClientPath)
		}
		first = false
	}
	writer.Flush()
}

func projectsAdd(ctx *cli.Context) error {

	// Decode project type
	var ptype xaapiv1.ProjectType
	switch strings.ToLower(ctx.String("type")) {
	case "pathmap", "pm":
		ptype = xaapiv1.TypePathMap
	case "cloudsync", "cs":
		ptype = xaapiv1.TypeCloudSync
	default:
		return cli.NewExitError("Unknown project type", 1)
	}

	prj := xaapiv1.ProjectConfig{
		ServerID:   XdsServerIDGet(),
		Label:      ctx.String("label"),
		Type:       ptype,
		ClientPath: ctx.String("path"),
		ServerPath: ctx.String("server-path"),
	}

	Log.Infof("POST /project %v", prj)
	newPrj := xaapiv1.ProjectConfig{}
	err := HTTPCli.Post("/projects", prj, &newPrj)
	if err != nil {
		return cli.NewExitError(err, 1)
	}

	fmt.Printf("New project '%s' (id %v) successfully created.\n", newPrj.Label, newPrj.ID)

	return nil
}

func projectsRemove(ctx *cli.Context) error {
	var res xaapiv1.ProjectConfig
	id := GetID(ctx)
	if id == "" {
		return cli.NewExitError("id parameter or option must be set", 1)
	}

	if err := HTTPCli.Delete("/projects/"+id, &res); err != nil {
		return cli.NewExitError(err, 1)
	}

	fmt.Println("Project ID " + res.ID + " successfully deleted.")
	return nil
}

func projectsSync(ctx *cli.Context) error {
	id := GetID(ctx)
	if id == "" {
		return cli.NewExitError("id parameter or option must be set", 1)
	}
	if err := HTTPCli.Post("/projects/sync/"+id, "", nil); err != nil {
		return cli.NewExitError(err, 1)
	}
	fmt.Println("Sync successfully resquested.")
	return nil
}
