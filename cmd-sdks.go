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
	"os"
	"regexp"

	"github.com/iotbzh/xds-agent/lib/xaapiv1"
	"github.com/urfave/cli"
)

func initCmdSdks(cmdDef *[]cli.Command) {
	*cmdDef = append(*cmdDef, cli.Command{
		Name:     "sdks",
		Aliases:  []string{"sdk"},
		HideHelp: true,
		Usage:    "SDKs commands group",
		Subcommands: []cli.Command{
			{
				Name:   "get",
				Usage:  "Get a property of a SDK",
				Action: sdksGet,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:   "id",
						Usage:  "sdk id",
						EnvVar: "XDS_SDK_ID",
					},
				},
			},
			{
				Name:    "list",
				Aliases: []string{"ls"},
				Usage:   "List installed SDKs",
				Action:  sdksList,
				Flags: []cli.Flag{
					cli.BoolFlag{
						Name:  "all, a",
						Usage: "display all existing sdks (installed + downloadable)",
					},
					cli.StringFlag{
						Name:  "filter, f",
						Usage: "regexp to filter output (filtering done only on ID, Name, Version and Arch fields) ",
					},
					cli.BoolFlag{
						Name:  "verbose, v",
						Usage: "display verbose output",
					},
				},
			},
			{
				Name:    "install",
				Aliases: []string{"i"},
				Usage:   "Install a SDK",
				Action:  sdksInstall,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:   "id",
						Usage:  "sdk id to install",
						EnvVar: "XDS_SDK_ID",
					},
					cli.StringFlag{
						Name:  "file, f",
						Usage: "use this file to install SDK",
					},
					cli.BoolFlag{
						Name:  "debug",
						Usage: "enable debug mode (useful to investigate install issue)",
					},
					cli.BoolFlag{
						Name:  "force",
						Usage: "force SDK installation when already installed",
					},
				},
			},
			{
				Name:    "uninstall",
				Aliases: []string{"rm"},
				Usage:   "UnInstall an existing SDK",
				Action:  sdksUnInstall,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:   "id",
						Usage:  "sdk id to un-install",
						EnvVar: "XDS_SDK_ID",
					},
				},
			},
			{
				Name:    "abort",
				Aliases: []string{"a"},
				Usage:   "Abort an install action",
				Action:  sdksAbort,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:   "id",
						Usage:  "sdk id to which abort action",
						EnvVar: "XDS_SDK_ID",
					},
				},
			},
		},
	})
}

func sdksList(ctx *cli.Context) error {
	// Get SDKs list
	sdks := []xaapiv1.SDK{}
	if err := _sdksListGet(&sdks); err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	_displaySdks(sdks, ctx.Bool("verbose"), ctx.Bool("all"), ctx.String("filter"))
	return nil
}

func sdksGet(ctx *cli.Context) error {
	id := GetID(ctx)
	if id == "" {
		return cli.NewExitError("id parameter or option must be set", 1)
	}
	sdks := xaapiv1.SDK{}
	url := XdsServerComputeURL("/sdks/" + id)
	if err := HTTPCli.Get(url, &sdks); err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	_displaySdks([]xaapiv1.SDK{sdks}, true, true, "")
	return nil
}

func _displaySdks(sdks []xaapiv1.SDK, verbose bool, all bool, filter string) {
	// Display result
	first := true
	writer := NewTableWriter()
	for _, s := range sdks {
		if s.Status != xaapiv1.SdkStatusInstalled && !all {
			continue
		}
		if filter != "" {
			re := regexp.MustCompile(filter)
			if !(re.MatchString(s.ID) || re.MatchString(s.Name) ||
				re.MatchString(s.Profile) || re.MatchString(s.Arch) ||
				re.MatchString(s.Version)) {
				continue
			}
		}

		if verbose {
			if !first {
				fmt.Fprintln(writer)
			}
			fmt.Fprintln(writer, "ID\t"+s.ID)
			fmt.Fprintln(writer, "Name\t"+s.Name)
			fmt.Fprintln(writer, "Description\t"+s.Description)
			fmt.Fprintln(writer, "Profile\t"+s.Profile)
			fmt.Fprintln(writer, "Arch\t"+s.Arch)
			fmt.Fprintln(writer, "Version\t"+s.Version)
			fmt.Fprintln(writer, "Status\t"+s.Status)
			fmt.Fprintln(writer, "Path\t"+s.Path)
			fmt.Fprintln(writer, "Url\t"+s.URL)

		} else {
			if first {
				if all {
					fmt.Fprintf(writer, "List of available SDKs: \n")
				} else {
					fmt.Fprintf(writer, "List of installed SDKs: \n")
				}
				fmt.Fprintf(writer, "ID\t NAME\t STATUS\t VERSION\t ARCH\n")
			}
			fmt.Fprintf(writer, "%s\t %s\t %s\t %s\t %s\n", s.ID[:8], s.Name, s.Status, s.Version, s.Arch)
		}
		first = false
	}
	writer.Flush()
}

func _sdksListGet(sdks *[]xaapiv1.SDK) error {
	url := XdsServerComputeURL("/sdks")
	if err := HTTPCli.Get(url, &sdks); err != nil {
		return err
	}
	Log.Debugf("Result of %s: %v", url, sdks)

	return nil
}

func sdksInstall(ctx *cli.Context) error {
	id := GetID(ctx)
	file := ctx.String("file")
	force := ctx.Bool("force")

	if id == "" && file == "" {
		return cli.NewExitError("id or file parameter or option must be set", 1)
	}

	// Process Socket IO events
	type exitResult struct {
		error string
		code  int
	}
	exitChan := make(chan exitResult, 1)

	IOsk.On("disconnection", func(err error) {
		Log.Debugf("WS disconnection event with err: %v\n", err)
		errMsg := ""
		if err != nil {
			errMsg = err.Error()
		}
		exitChan <- exitResult{errMsg, 2}
	})

	IOsk.On(xaapiv1.EVTSDKInstall, func(ev xaapiv1.EventMsg) {
		sdkEvt, _ := ev.DecodeSDKMsg()

		if sdkEvt.Stdout != "" {
			fmt.Printf("%s", sdkEvt.Stdout)
		}
		if sdkEvt.Stderr != "" {
			fmt.Fprintf(os.Stderr, "%s", sdkEvt.Stderr)
		}

		if sdkEvt.Exited {
			exitChan <- exitResult{sdkEvt.Error, sdkEvt.Code}
		}
	})

	evReg := xaapiv1.EventRegisterArgs{Name: xaapiv1.EVTSDKInstall}
	if err := HTTPCli.Post("/events/register", &evReg, nil); err != nil {
		return cli.NewExitError(err, 1)
	}

	url := XdsServerComputeURL("/sdks")
	sdks := xaapiv1.SDKInstallArgs{
		ID:       id,
		Filename: file,
		Force:    force,
	}

	if ctx.Bool("debug") {
		sdks.InstallArgs = []string{"--debug"}
	}

	newSdk := xaapiv1.SDK{}
	if err := HTTPCli.Post(url, &sdks, &newSdk); err != nil {
		return cli.NewExitError(err, 1)
	}
	Log.Debugf("Result of %s: %v", url, newSdk)
	fmt.Printf("Installation of '%s' SDK successfully started.\n", newSdk.Name)

	// TODO: trap CTRL+C and print question: "Installation of xxx is in progress, press 'a' to abort, 'b' to continue in background or 'c' to continue installation"

	// Wait exit
	select {
	case res := <-exitChan:
		if res.code == 0 {
			Log.Debugln("Exit successfully")
			fmt.Println("SDK ID " + newSdk.ID + " successfully installed.")
		}
		if res.error != "" {
			Log.Debugln("Exit with ERROR: ", res.error)
		}
		return cli.NewExitError(res.error, res.code)
	}
}

func sdksUnInstall(ctx *cli.Context) error {
	id := GetID(ctx)
	if id == "" {
		return cli.NewExitError("id parameter or option must be set", 1)
	}

	delSdk := xaapiv1.SDK{}
	url := XdsServerComputeURL("/sdks/" + id)
	if err := HTTPCli.Delete(url, &delSdk); err != nil {
		return cli.NewExitError(err, 1)
	}

	Log.Debugf("Result of %s: %v", url, delSdk)

	fmt.Println("SDK ID " + delSdk.ID + " successfully deleted.")
	return nil
}

func sdksAbort(ctx *cli.Context) error {
	id := GetID(ctx)
	if id == "" {
		return cli.NewExitError("id parameter or option must be set", 1)
	}

	sdks := xaapiv1.SDKInstallArgs{ID: id}
	newSdk := xaapiv1.SDK{}
	url := XdsServerComputeURL("/sdks/abortinstall")
	if err := HTTPCli.Post(url, &sdks, &newSdk); err != nil {
		return cli.NewExitError(err, 1)
	}

	Log.Debugf("Result of %s: %v", url, newSdk)
	return nil
}
