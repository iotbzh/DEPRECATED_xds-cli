package main

import (
	"encoding/json"

	"github.com/iotbzh/xds-agent/lib/xaapiv1"
	"github.com/urfave/cli"
)

var cacheXdsVersion *xaapiv1.XDSVersion

// XdsVersionGet Get version of XDS agent & server
func XdsVersionGet(ver *xaapiv1.XDSVersion) error {
	// Use cached data
	if cacheXdsVersion != nil {
		ver = cacheXdsVersion
		return nil
	}

	dataVer := xaapiv1.XDSVersion{}
	if err := HTTPCli.Get("/version", &dataVer); err != nil {
		return err
	}

	cacheXdsVersion = &dataVer
	*ver = dataVer
	return nil
}

// XdsServerIDGet returns the XDS Server ID
func XdsServerIDGet() string {
	ver := xaapiv1.XDSVersion{}
	if err := XdsVersionGet(&ver); err != nil {
		return ""
	}
	if len(ver.Server) < 1 {
		return ""
	}
	return ver.Server[XdsServerIndexGet()].ID
}

// XdsServerIndexGet returns the index number of XDS Server
func XdsServerIndexGet() int {
	// FIXME support multiple server
	return 0
}

// ProjectsListGet Get the list of existing projects
func ProjectsListGet(prjs *[]xaapiv1.ProjectConfig) error {
	var data []byte
	if err := HTTPCli.HTTPGet("/projects", &data); err != nil {
		return err
	}
	Log.Debugf("Result of /projects: %v", string(data[:]))

	return json.Unmarshal(data, &prjs)
}

// LogPost Helper to log a POST request
func LogPost(format string, data interface{}) {
	b, _ := json.Marshal(data)
	Log.Infof(format, string(b))
}

// GetID Return a string ID set with --id option or as simple parameter
func GetID(ctx *cli.Context) string {
	id := ctx.String("id")
	idArgs := ctx.Args().First()
	if id == "" && idArgs != "" {
		id = idArgs
	}
	return id
}
