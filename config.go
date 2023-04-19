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

func Read(i any, envPrefix string, configFile string, overwritingFile string) (ov Overwrites, err error) {
	if err := LoadDotenv(); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}
	if err = ReadFromEnv(i, envPrefix); err != nil {
		return nil, err
	}
	if configFile != "" {
		if err = ReadFromFile(i, configFile); err != nil {
			return nil, err
		}
	}
	if overwritingFile != "" {
		if ov, err = ReadOverwritingFile(i, overwritingFile); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				// the overwriting file is optional
				ov = make(Overwrites)
				err = nil
			} else {
				return nil, err
			}
		}
	}
	return ov, nil
}

func OverwriteJSON(cfg any, newOv Overwrites, oldOv Overwrites) error {
	keys := make([]string, 0, len(newOv))
	for key := range newOv {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := newOv[key]
		if err := overwriteJSON(cfg, key, value); err != nil {
			return err
		}
		if oldOv != nil {
			// add key and value to 'overwrites' map
			// all keys in overwrites that are a child of 'key' will be removed
			// (this is to prevent overwrites from being overwritten)
			oldOv[key] = value
			childKey := key + "."
			for k := range oldOv {
				if strings.HasPrefix(k, childKey) {
					delete(oldOv, k)
				}
			}
		}
	}

	return nil
}

func overwriteJSON(cfg any, key string, value any) error {
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
		if err := overwriteJSON(cfg, key, value); err != nil {
			errs = append(errs, fmt.Errorf("%s: \"%s\": %w", filename, key, err))
		}
	}
	if len(errs) > 0 {
		return ov, errors.Join(errs...)
	}
	return ov, nil
}
