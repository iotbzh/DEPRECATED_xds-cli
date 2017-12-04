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

	"github.com/iotbzh/xds-agent/lib/xaapiv1"
	"github.com/urfave/cli"
)

func initCmdMisc(cmdDef *[]cli.Command) {
	*cmdDef = append(*cmdDef, cli.Command{
		Name:     "misc",
		HideHelp: true,
		Usage:    "miscellaneous commands group",
		Subcommands: []cli.Command{
			{
				Name:    "version",
				Aliases: []string{"v"},
				Usage:   "Get version of XDS agent and XDS server",
				Action:  xdsVersion,
				Flags: []cli.Flag{
					cli.BoolFlag{
						Name:  "verbose, v",
						Usage: "display verbose output",
					},
				},
			},
			{
				Name:    "status",
				Aliases: []string{"sts"},
				Usage:   "Get XDS configuration status (including XDS server connection)",
				Action:  xdsStatus,
				Flags: []cli.Flag{
					cli.BoolFlag{
						Name:  "verbose, v",
						Usage: "display verbose output",
					},
				},
			},
		},
	})
}

func xdsVersion(ctx *cli.Context) error {
	verbose := ctx.Bool("verbose")

	// Get version
	ver := xaapiv1.XDSVersion{}
	if err := XdsVersionGet(&ver); err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	writer := NewTableWriter()
	fmt.Fprintln(writer, "Agent:")
	fmt.Fprintln(writer, "      ID:\t", ver.Client.ID)
	v := ver.Client.Version
	if verbose {
		v += " (" + ver.Client.VersionGitTag + ")"
	}
	fmt.Fprintln(writer, "      Version:\t", v)
	if verbose {
		fmt.Fprintln(writer, "      API Version:\t", ver.Client.APIVersion)
	}

	for _, svr := range ver.Server {
		fmt.Fprintln(writer, "Server:")
		fmt.Fprintln(writer, "       ID:\t", svr.ID)
		v = svr.Version
		if verbose {
			v += " (" + svr.VersionGitTag + ")"
		}
		fmt.Fprintln(writer, "       Version:\t", v)
		if verbose {
			fmt.Fprintln(writer, "       API Version:\t", svr.APIVersion)
		}
	}
	writer.Flush()

	return nil
}

func xdsStatus(ctx *cli.Context) error {
	cfg := xaapiv1.APIConfig{}
	if err := XdsConfigGet(&cfg); err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	writer := NewTableWriter()
	fmt.Fprintln(writer, "XDS Server:")
	for _, svr := range cfg.Servers {
		fmt.Fprintln(writer, "       ID:\t", svr.ID)
		fmt.Fprintln(writer, "       URL:\t", svr.URL)
		fmt.Fprintln(writer, "       Connected:\t", svr.Connected)
		fmt.Fprintln(writer, "       Connection retry:\t", svr.ConnRetry)
		fmt.Fprintln(writer, "       Disabled:\t", svr.Disabled)
	}
	writer.Flush()

	return nil
}
