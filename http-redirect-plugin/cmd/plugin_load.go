package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"plugin"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/http-redirect-plugin/protocol"
)

// global but private, safe usage here in this file
var pluginPathList []string

func LoadConfig() {
	f, err := os.ReadFile("./data/config.json")
	if err != nil {
		// NOTE: in real cases, deal with this error
		panic(err)
	}
	json.Unmarshal(f, &pluginPathList)
}

var pluginList []*protocol.HttpRedirectPlugin

func LoadPlugins() {
	// Allocate a list for storing all our plugins
	pluginList = make([]*protocol.HttpRedirectPlugin, 0, len(pluginPathList))
	for _, p := range pluginPathList {
		// We use plugin.Open to load plugins by path
		plg, err := plugin.Open(p)
		if err != nil {
			// NOTE: in real cases, deal with this error
			panic(err)
		}

		// Search for variable named "Plugin"
		v, err := plg.Lookup("Plugin")
		if err != nil {
			// NOTE: in real cases, deal with this error
			panic(err)
		}

		// Cast symbol to protocol type
		castV, ok := v.(protocol.HttpRedirectPlugin)
		if !ok {
			// NOTE: in real cases, deal with this error
			panic(fmt.Sprintf("Could not cast plugin for %s", p))
		}

		pluginList = append(pluginList, &castV)
	}
}

// Let's throw this here so it loads the plugins as soon as we import this module
func init() {
	LoadConfig()
	LoadPlugins()
}

func PreRequestHook(req *http.Request) {
	for _, plg := range pluginList {
		// Plugin is a list of pointers, we need to dereference them
		// to use the proper function
		(*plg).PreRequestHook(req)
	}
}
