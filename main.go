// xds-cli: command line tool used to control / interface X(cross) Development System.
package main

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/Sirupsen/logrus"
	common "github.com/iotbzh/xds-common/golib"
	socketio_client "github.com/sebd71/go-socket.io-client"
	"github.com/urfave/cli"
)

var appAuthors = []cli.Author{
	cli.Author{Name: "Sebastien Douheret", Email: "sebastien@iot.bzh"},
}

// AppName name of this application
var AppName = "xds-cli"

// AppNativeName native command name that this application can overload
var AppNativeName = "cli"

// AppVersion Version of this application
// (set by Makefile)
var AppVersion = "?.?.?"

// AppSubVersion is the git tag id added to version string
// Should be set by compilation -ldflags "-X main.AppSubVersion=xxx"
// (set by Makefile)
var AppSubVersion = "unknown-dev"

// Application details
const (
	appCopyright    = "Apache-2.0"
	defaultLogLevel = "error"
)

// Log Global variable that hold logger
var Log = logrus.New()

// HTTPCli Global variable that hold HTTP Client
var HTTPCli *common.HTTPClient

// IOsk Global variable that hold SocketIo client
var IOsk *socketio_client.Client

// exitError exists this program with the specified error
func exitError(code int, f string, a ...interface{}) {
	err := fmt.Sprintf(f, a...)
	fmt.Fprintf(os.Stderr, err+"\n")
	os.Exit(code)
}

// main
func main() {

	// Allow to set app name from cli (useful for debugging)
	if AppName == "" {
		AppName = os.Getenv("XDS_APPNAME")
	}
	if AppName == "" {
		panic("Invalid setup, AppName not define !")
	}
	if AppNativeName == "" {
		AppNativeName = AppName[4:]
	}
	appUsage := fmt.Sprintf("command line tool for X(cross) Development System.")
	appDescription := fmt.Sprintf("%s utility for X(cross) Development System\n", AppName)
	/* SEB UPDATE DOC
		appDescription += `
	   xds-cli configuration is driven either by environment variables or by command line
	   options or using a config file knowning that the following priority order is used:
	     1. use option value (for example use project ID set by --id option),
	     2. else use variable 'XDS_xxx' (for example 'XDS_PROJECT_ID' variable) when a
	        config file is specified with '--config|-c' option,
	     3. else use 'XDS_xxx' (for example 'XDS_PROJECT_ID') environment variable.
	`
	*/
	// Create a new App instance
	app := cli.NewApp()
	app.Name = AppName
	app.Usage = appUsage
	app.Version = AppVersion + " (" + AppSubVersion + ")"
	app.Authors = appAuthors
	app.Copyright = appCopyright
	app.Metadata = make(map[string]interface{})
	app.Metadata["version"] = AppVersion
	app.Metadata["git-tag"] = AppSubVersion
	app.Metadata["logger"] = Log

	// Create env vars help
	dynDesc := "\nENVIRONMENT VARIABLES:"
	for _, f := range app.Flags {
		var env, usage string
		switch f.(type) {
		case cli.StringFlag:
			fs := f.(cli.StringFlag)
			env = fs.EnvVar
			usage = fs.Usage
		case cli.BoolFlag:
			fb := f.(cli.BoolFlag)
			env = fb.EnvVar
			usage = fb.Usage
		default:
			exitError(1, "Un-implemented option type")
		}
		if env != "" {
			dynDesc += fmt.Sprintf("\n %s \t\t %s", env, usage)
		}
	}
	app.Description = appDescription + dynDesc

	// Declare global flags
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "config, c",
			EnvVar: "XDS_CONFIG",
			Usage:  "env config file to source on startup",
		},
		cli.StringFlag{
			Name:   "log, l",
			EnvVar: "XDS_LOGLEVEL",
			Usage:  "logging level (supported levels: panic, fatal, error, warn, info, debug)",
			Value:  defaultLogLevel,
		},
		cli.StringFlag{
			Name:   "url",
			EnvVar: "XDS_SERVER_URL",
			Value:  "localhost:8000",
			Usage:  "remote XDS server url",
		},
		cli.BoolFlag{
			Name:   "timestamp, ts",
			EnvVar: "XDS_TIMESTAMP",
			Usage:  "prefix output with timestamp",
		},
	}

	// Declare commands
	app.Commands = []cli.Command{}

	initCmdProjects(&app.Commands)
	initCmdSdks(&app.Commands)
	initCmdExec(&app.Commands)
	initCmdMisc(&app.Commands)

	sort.Sort(cli.FlagsByName(app.Flags))
	sort.Sort(cli.CommandsByName(app.Commands))

	app.Before = func(ctx *cli.Context) error {
		var err error
		loglevel := ctx.String("log")
		// Set logger level and formatter
		if Log.Level, err = logrus.ParseLevel(loglevel); err != nil {
			msg := fmt.Sprintf("Invalid log level : \"%v\"\n", loglevel)
			return cli.NewExitError(msg, 1)
		}
		Log.Formatter = &logrus.TextFormatter{}

		Log.Infof("%s version: %s", AppName, app.Version)
		// SEB Add again Log.Debugf("Environment: %v", os.Environ())

		if err = XdsConnInit(ctx); err != nil {
			// Directly call HandleExitCoder to avoid to print help (ShowAppHelp)
			// Note that this function wil never return and program will exit
			cli.HandleExitCoder(err)
		}

		return nil
	}

	// Close HTTP client and WS connection on exit
	defer func() {
		XdsConnClose()
	}()

	app.Run(os.Args)
}

// XdsConnInit Initialized HTTP and WebSocket connection to XDS agent
func XdsConnInit(ctx *cli.Context) error {
	var err error

	// Define HTTP and WS url
	baseURL := ctx.String("url")
	if !strings.HasPrefix(ctx.String("url"), "http://") {
		baseURL = "http://" + ctx.String("url")
	}

	// Create HTTP client
	Log.Debugln("Connect HTTP client on ", baseURL)
	conf := common.HTTPClientConfig{
		URLPrefix:           "/api/v1",
		HeaderClientKeyName: "Xds-Agent-Sid",
		CsrfDisable:         true,
		LogOut:              Log.Out,
		LogPrefix:           "XDSAGENT: ",
		LogLevel:            common.HTTPLogLevelWarning,
	}

	HTTPCli, err = common.HTTPNewClient(baseURL, conf)
	if err != nil {
		errmsg := err.Error()
		if m, err := regexp.MatchString("Get http.?://", errmsg); m && err == nil {
			i := strings.LastIndex(errmsg, ":")
			errmsg = "Cannot connection to " + baseURL + errmsg[i:]
		}
		return cli.NewExitError(errmsg, 1)
	}
	HTTPCli.SetLogLevel(ctx.String("loglevel"))

	// Create io Websocket client
	Log.Debugln("Connecting IO.socket client on ", baseURL)

	opts := &socketio_client.Options{
		Transport: "websocket",
		Header:    make(map[string][]string),
	}
	opts.Header["XDS-AGENT-SID"] = []string{HTTPCli.GetClientID()}

	IOsk, err = socketio_client.NewClient(baseURL, opts)
	if err != nil {
		return cli.NewExitError("IO.socket connection error: "+err.Error(), 1)
	}

	IOsk.On("error", func(err error) {
		fmt.Println("ERROR Websocket: ", err.Error())
	})

	ctx.App.Metadata["httpCli"] = HTTPCli
	ctx.App.Metadata["ioskCli"] = IOsk

	return nil
}

// XdsConnClose Terminate connection to XDS agent
func XdsConnClose() {
	Log.Debugf("Closing HTTP client session...")
	/* TODO
	if httpCli, ok := app.Metadata["httpCli"]; ok {
		c := httpCli.(*common.HTTPClient)
	}
	*/

	Log.Debugf("Closing WebSocket connection...")
	/*
		if ioskCli, ok := app.Metadata["ioskCli"]; ok {
			c := ioskCli.(*socketio_client.Client)
		}
	*/
}

// NewTableWriter Create a writer that inserts padding around tab-delimited
func NewTableWriter() *tabwriter.Writer {
	writer := new(tabwriter.Writer)
	writer.Init(os.Stdout, 0, 8, 0, '\t', 0)
	return writer
}
