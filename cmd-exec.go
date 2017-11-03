package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/iotbzh/xds-agent/lib/apiv1"
	common "github.com/iotbzh/xds-common/golib"
	"github.com/joho/godotenv"
	"github.com/urfave/cli"
)

func initCmdExec(cmdDef *[]cli.Command) {
	*cmdDef = append(*cmdDef, cli.Command{
		Name:   "exec",
		Usage:  "execute a command in XDS",
		Action: exec,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "id",
				EnvVar: "XDS_PROJECT_ID",
				Usage:  "project ID you want to build (mandatory variable)",
			},
			cli.StringFlag{
				Name:   "rpath",
				EnvVar: "XDS_RPATH",
				Usage:  "relative path into project",
			},
			cli.StringFlag{
				Name:   "sdkid",
				EnvVar: "XDS_SDK_ID",
				Usage:  "Cross Sdk ID to use to build project",
			},
		},
	})
}

func exec(ctx *cli.Context) error {
	prjID := ctx.String("id")
	confFile := ctx.String("config")
	rPath := ctx.String("rPath")
	sdkid := ctx.String("sdkid")

	// Check mandatory args
	if prjID == "" {
		return cli.NewExitError("project id must be set (see --id option)", 1)
	}

	// Load config file if requested
	envMap := make(map[string]string)
	if confFile != "" {
		if !common.Exists(confFile) {
			exitError(1, "Error env config file not found")
		}
		// Load config file variables that will overwrite env variables
		err := godotenv.Overload(confFile)
		if err != nil {
			exitError(1, "Error loading env config file "+confFile)
		}
		envMap, err = godotenv.Read(confFile)
		if err != nil {
			exitError(1, "Error reading env config file "+confFile)
		}
	}

	argsCommand := make([]string, len(ctx.Args()))
	copy(argsCommand, ctx.Args())
	Log.Infof("Execute: /exec %v", argsCommand)

	// Log useful info for debugging
	ver := apiv1.XDSVersion{}
	XdsVersionGet(&ver)
	Log.Infof("XDS version: %v", ver)

	// Process Socket IO events
	type exitResult struct {
		error error
		code  int
	}
	exitChan := make(chan exitResult, 1)

	IOsk.On("disconnection", func(err error) {
		exitChan <- exitResult{err, 2}
	})

	outFunc := func(timestamp, stdout, stderr string) {
		tm := ""
		if ctx.Bool("WithTimestamp") {
			tm = timestamp + "| "
		}
		if stdout != "" {
			fmt.Printf("%s%s", tm, stdout)
		}
		if stderr != "" {
			fmt.Fprintf(os.Stderr, "%s%s", tm, stderr)
		}
	}

	IOsk.On(apiv1.ExecOutEvent, func(ev apiv1.ExecOutMsg) {
		outFunc(ev.Timestamp, ev.Stdout, ev.Stderr)
	})

	IOsk.On(apiv1.ExecExitEvent, func(ev apiv1.ExecExitMsg) {
		exitChan <- exitResult{ev.Error, ev.Code}
	})

	// Retrieve the project definition
	prj := apiv1.ProjectConfig{}
	if err := HTTPCli.Get("/projects/"+prjID, &prj); err != nil {
		return cli.NewExitError(err, 1)
	}

	// Auto setup rPath if needed
	if rPath == "" {
		cwd, err := os.Getwd()
		if err == nil {
			fldRp := prj.ClientPath
			if !strings.HasPrefix(fldRp, "/") {
				fldRp = "/" + fldRp
			}
			Log.Debugf("Try to auto-setup rPath: cwd=%s ; ClientPath=%s", cwd, fldRp)
			if sp := strings.SplitAfter(cwd, fldRp); len(sp) == 2 {
				rPath = strings.Trim(sp[1], "/")
				Log.Debugf("Auto-setup rPath to: '%s'", rPath)
			}
		}
	}

	// Build env
	Log.Debugf("Command env: %v", envMap)
	env := []string{}
	for k, v := range envMap {
		env = append(env, k+"="+v)
	}

	// Send build command
	args := apiv1.ExecArgs{
		ID:         prjID,
		SdkID:      sdkid,
		Cmd:        strings.Trim(argsCommand[0], " "),
		Args:       argsCommand[1:],
		Env:        env,
		RPath:      rPath,
		CmdTimeout: 60,
	}

	LogPost("POST /exec %v", args)
	if err := HTTPCli.Post("/exec", args, nil); err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	// Wait exit
	select {
	case res := <-exitChan:
		errStr := ""
		if res.code == 0 {
			Log.Debugln("Exit successfully")
		}
		if res.error != nil {
			Log.Debugln("Exit with ERROR: ", res.error.Error())
			errStr = res.error.Error()
		}
		return cli.NewExitError(errStr, res.code)
	}
}
