package configs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func ReadFromFile(i any, filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	ext := filepath.Ext(filename)
	unmarshal := FileExtensions[ext]
	if unmarshal == nil {
		return fmt.Errorf("config: unknown file extension %q", ext)
	}
	return unmarshal(data, i)
}

var FileExtensions = map[string]func(data []byte, v any) error{
	".json": json.Unmarshal,
	".yaml": yaml.Unmarshal,
}
