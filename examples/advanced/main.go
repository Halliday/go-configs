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

var configFile *string
var overwritingFile *string
var overwrites configs.Overwrites

func main() {
	// the config file is usually "config.yaml" but can be overridden with the "-config" flag
	configFile = flag.String("config", "config.yaml", "path to config file")
	overwritingFile = flag.String("local", "local.json", "path to local overwriting file")

	flag.Parse()

	// we fill the config struct with
	// - values from the .env file (if any)
	// - values from the environment variables
	// - values from the config file (if any)
	var err error
	overwrites, err = configs.Read(&config, "APP_", *configFile, *overwritingFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// if the config file doesn't exist we use the default values
			log.Printf("No config file found at %s. Using default values.", *configFile)
		} else {
			// any other error is fatal
			log.Fatal(err)
		}
	}

	ov := make(configs.Overwrites)
	if config.Key == "" {
		ov["key"] = strconv.Itoa(rand.Int())
		log.Printf("A random key was generated: %s", ov["key"])
	}
	if len(ov) > 0 {
		overwriteConfig(ov)
	}

	// add a http handler to overwrite the config
	http.HandleFunc("/config", handleOverwriteConfig)

	log.Printf("Listening on %s as %s.", config.Listen, config.Hostname)

	// this will block until the server is stopped or the program terminates
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

	if err := overwriteConfig(ov); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

// overwriteConfig changes the config and writes the overwrites to the local overwriting file.
func overwriteConfig(ov configs.Overwrites) error {
	if len(ov) == 0 {
		return nil
	}

	err := configs.OverwriteJSON(&config, ov, overwrites)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(overwrites, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(*overwritingFile, data, 0644); err != nil {
		panic(err)
	}

	return nil
}
