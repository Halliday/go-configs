package main

import (
	"errors"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/halliday/go-configs"
)

// Config is the configuration struct for this little program.
type Config struct {
	Listen   string `json:"listen"`
	Hostname string `json:"hostname"`
}

// config is the default configuration.
var config = Config{
	Listen:   ":80",
	Hostname: "example.com",
}

func main() {
	// the config file is usually "config.yaml" but can be overridden with the -config flag
	configFile := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	// we fill the config struct with
	// - values from the .env file (if any)
	// - values from the environment variables
	// - values from the config file (if any)
	var err error
	_, err = configs.Read(&config, "APP_", *configFile, "")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// if the config file doesn't exist we use the default values
			log.Printf("No config file found at %s. Using default values.", *configFile)
		} else {
			// any other error is fatal
			log.Fatal(err)
		}
	}

	log.Printf("Listening on %s as %s.", config.Listen, config.Hostname)

	err = http.ListenAndServe(config.Listen, nil)
	log.Fatal(err)
}
