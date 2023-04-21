package configs

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"
)

type Config struct {
	Value          any
	Overwrites     Overwrites
	EnvPrefix      string
	File           string
	IgnoreEnv      bool
	IgnoreDotEnv   bool
	OverwritesFile string

	usedEnvKeys []string
}

func (c *Config) UsedEnvKeys() []string {
	return c.usedEnvKeys
}

func (c *Config) UnusedEnvKeys() []string {
	var keys []string
	for _, key := range os.Environ() {
		if !strings.HasPrefix(key, c.EnvPrefix) {
			continue
		}
		i := strings.Index(key, "=")
		if i == -1 {
			continue
		}
		key = key[:i]
		if !stringsSliceContains(c.usedEnvKeys, key) {
			keys = append(keys, key)
		}
	}
	return keys
}

func stringsSliceContains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func (c *Config) Read() (err error) {
	if !c.IgnoreEnv {
		if err = c.ReadFromEnv(c.Value, c.EnvPrefix); err != nil {
			return err
		}
	}
	if c.File != "" {
		if !c.IgnoreDotEnv {
			if err = LoadDotenv(); err != nil {
				if !errors.Is(err, fs.ErrNotExist) {
					return err
				}
			}
		}
		if err = ReadFromFile(c.Value, c.File); err != nil {
			return err
		}
	}
	if c.OverwritesFile != "" {
		if c.Overwrites, err = ReadOverwritingFile(c.Value, c.OverwritesFile); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				// the overwriting file is optional
				c.Overwrites = make(Overwrites)
				err = nil
			} else {
				return err
			}
		}
	}
	return nil
}

func Read(i any, envPrefix string, file string) (err error) {
	c := Config{
		Value:     i,
		File:      file,
		EnvPrefix: envPrefix,
	}
	return c.Read()
}

//

func (c *Config) Overwrite(ov Overwrites) error {
	if len(ov) == 0 {
		return nil
	}
	if c.Overwrites == nil {
		c.Overwrites = make(map[string]any)
	}
	keys := make([]string, 0, len(ov))
	for key := range ov {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := ov[key]
		if err := overwrite(c.Value, key, value); err != nil {
			return err
		}
		// add key and value to 'overwrites' map
		// all keys in overwrites that are a child of 'key' will be removed
		// (this is to prevent overwrites from being overwritten)
		c.Overwrites[key] = value
		childKey := key + "."
		for k := range c.Overwrites {
			if strings.HasPrefix(k, childKey) {
				delete(c.Overwrites, k)
			}
		}
	}

	if c.OverwritesFile != "" {
		data, err := json.MarshalIndent(c.Overwrites, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(c.OverwritesFile, data, 0644); err != nil {
			return err
		}
	}

	return nil
}

func overwrite(cfg any, key string, value any) error {
	seps := strings.Split(key, ".")
	var b bytes.Buffer
	for _, s := range seps {
		b.WriteByte('{')
		keyData, _ := json.Marshal(s)
		b.Write(keyData)
		b.WriteByte(':')
	}
	valueData, err := json.Marshal(value)
	if err != nil {
		return err
	}
	b.Write(valueData)
	for i := 0; i < len(seps); i++ {
		b.WriteByte('}')
	}
	if err := json.Unmarshal(b.Bytes(), cfg); err != nil {
		return err
	}
	return nil
}

type Overwrites map[string]any

func ReadOverwritingFile(cfg any, filename string) (ov Overwrites, err error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	ov = make(Overwrites)
	if err := json.Unmarshal(data, &ov); err != nil {
		return nil, fmt.Errorf("%s: %w", filename, err)
	}
	errs := make([]error, 0)
	for key, value := range ov {
		if err := overwrite(cfg, key, value); err != nil {
			errs = append(errs, fmt.Errorf("%s: \"%s\": %w", filename, key, err))
		}
	}
	if len(errs) > 0 {
		return ov, errors.Join(errs...)
	}
	return ov, nil
}
