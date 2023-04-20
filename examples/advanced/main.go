package main

import (
	"encoding/json"
	"errors"
	"flag"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/halliday/go-configs"
)

// Config is the configuration struct for this little program.
type Config struct {
	Listen   string `json:"listen"`
	Hostname string `json:"hostname"`
	Key      string `json:"key"`
}

// config is the default configuration.
var config = Config{
	Listen:   ":80",
	Hostname: "example.com",
}

var c *configs.Config

func main() {
	var err error

	// the config file is usually "config.yaml" but can be overridden with the "-config" flag
	configFile := flag.String("config", "config.yaml", "path to config file")
	overwritingFile := flag.String("local", "local.json", "path to local overwriting file")

	flag.Parse()

	// ====================

	// we fill the config struct with
	// - values from the .env file (if any)
	// - values from the environment variables
	// - values from the config file (if any)
	c = &configs.Config{
		EnvPrefix:      "APP_",
		Value:          &config,
		File:           *configFile,
		OverwritesFile: *overwritingFile,
	}
	if err := c.Read(); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// if the config file doesn't exist we use the default values
			log.Printf("No config file found at %s. Using default values.", *configFile)
		} else {
			// any other error is fatal
			log.Fatal(err)
		}
	}

	// if there are env vars with "APP_" prefix that do not apply to the config struct
	// we better log them so the user can remove them
	if unusedEnvKeys := c.UnusedEnvKeys(); len(unusedEnvKeys) > 0 {
		log.Printf("Unused environment variables: %s", strings.Join(unusedEnvKeys, ", "))
	}

	// now lets make some overwrites e.g. for generated keys that should persist between restarts
	ov := make(configs.Overwrites)
	if config.Key == "" {
		ov["key"] = strconv.Itoa(rand.Int())
		log.Printf("A random key was generated: %s", ov["key"])
	}
	if err := c.Overwrite(ov); err != nil {
		log.Fatal(err)
	}

	// we are done with the config, now we can start the server

	// ====================

	// add a http handler to overwrite the config
	http.HandleFunc("/config", handleOverwriteConfig)

	log.Printf("Listening on %s as %s.", config.Listen, config.Hostname)

	// this will block until the program terminates
	err = http.ListenAndServe(config.Listen, nil)
	log.Fatal(err)
}

// handleOverwriteConfig is the http handler for the /config endpoint.
func handleOverwriteConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Use HTTP POST request with this endpoint.", http.StatusMethodNotAllowed)
		return
	}
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Use Content-Type: application/json header with this endpoint.", http.StatusBadRequest)
		return
	}

	var ov configs.Overwrites
	if err := json.NewDecoder(r.Body).Decode(&ov); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := c.Overwrite(ov); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}
