package main

import (
	"fmt"
	"strconv"

	"github.com/iotbzh/xds-agent/lib/apiv1"
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
				Name:    "add",
				Aliases: []string{"a"},
				Usage:   "Add a new SDK",
				Action:  sdksAdd,
			},
			{
				Name:   "get",
				Usage:  "Get a property of a SDK",
				Action: sdksGet,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "id",
						Usage: "sdk id",
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
						Name:  "verbose, v",
						Usage: "display verbose output",
					},
				},
			},
			{
				Name:    "remove",
				Aliases: []string{"rm"},
				Usage:   "Remove an existing SDK",
				Action:  sdksRemove,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "id",
						Usage: "sdk id",
					},
				},
			},
		},
	})
}

func sdksList(ctx *cli.Context) error {
	// Get SDKs list
	sdks := []apiv1.SDK{}
	if err := sdksListGet(&sdks); err != nil {
		return cli.NewExitError(err.Error(), 1)
	}
	_displaySdks(sdks, ctx.Bool("verbose"))
	return nil
}

func sdksGet(ctx *cli.Context) error {
	id := GetID(ctx)
	if id == "" {
		return cli.NewExitError("id parameter or option must be set", 1)
	}
	sdks := apiv1.SDK{}
	url := "server/" + strconv.Itoa(XdsServerIndexGet()) + "/sdks/" + id
	if err := HTTPCli.Get(url, &sdks); err != nil {
		return cli.NewExitError(err.Error(), 1)
	}
	_displaySdks([]apiv1.SDK{sdks}, true)
	return nil
}

func _displaySdks(sdks []apiv1.SDK, verbose bool) {
	// Display result
	first := true
	writer := NewTableWriter()
	for _, s := range sdks {
		if verbose {
			if !first {
				fmt.Fprintln(writer)
			}
			fmt.Fprintln(writer, "ID\t"+s.ID)
			fmt.Fprintln(writer, "Name\t"+s.Name)
			fmt.Fprintln(writer, "Profile\t"+s.Profile)
			fmt.Fprintln(writer, "Arch\t"+s.Arch)
			fmt.Fprintln(writer, "Version\t"+s.Version)
			fmt.Fprintln(writer, "Path\t"+s.Path)

		} else {
			if first {
				fmt.Fprintf(writer, "List of installed SDKs: \n")
				fmt.Fprintf(writer, "  ID\tNAME\n")
			}
			fmt.Fprintf(writer, "  %s\t%s\n", s.ID, s.Name)
		}
		first = false
	}
	writer.Flush()
}

func sdksListGet(sdks *[]apiv1.SDK) error {
	url := "server/" + strconv.Itoa(XdsServerIndexGet()) + "/sdks"
	if err := HTTPCli.Get(url, &sdks); err != nil {
		return err
	}
	Log.Debugf("Result of %s: %v", url, sdks)

	return nil
}

func sdksAdd(ctx *cli.Context) error {
	return fmt.Errorf("not supported yet")
}

func sdksRemove(ctx *cli.Context) error {
	id := GetID(ctx)
	if id == "" {
		return cli.NewExitError("id parameter or option must be set", 1)
	}

	return fmt.Errorf("not supported yet")
}
