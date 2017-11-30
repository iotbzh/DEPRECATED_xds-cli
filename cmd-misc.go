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
