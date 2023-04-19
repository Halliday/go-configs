package configs

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

var env = map[string]string{
	"TEST_KEY7":  "value7",
	"TEST_KEY8":  "dmFsdWU4", // base64 encoded "value8"
	"TEST_KEY9":  "1",        // boolean true
	"TEST_KEY10": "value10",
	"TEST_KEY12": "value12_2",
}

var expected = Config{
	// values from config.json
	Key1: "value1",
	Key2: 2,
	Key3: &Field3{
		Key4: true,
		Key5: "value5",
	},
	// unchanged from default
	Key6: "default6",
	// overwritten by env
	Key7: "value7",
	Key8: []byte("value8"),
	Key9: true,
	Key10: &Field10{
		valid: true,
	},
	// unchanged from default
	Key11: nil,
	// overwritten by overwrite1.json
	Key12: "value12_3",
}

var config = Config{
	Key1: "default1",
	Key6: "default6",
}

func TestRead(t *testing.T) {
	for k, v := range env {
		t.Setenv(k, v)
	}

	_, err := Read(&config, "TEST_", "test/config1.json", "test/overwrite1.json")
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(config, expected) {
		t.Fatalf("invalid config: (expected <> got)\n%s\n%s", stringify(expected), stringify(config))
	}
	t.Logf("%+v", config)
}

type Field3 struct {
	Key4 bool   `json:"key4" yaml:"key4" env:"KEY4"`
	Key5 string `json:"key5" yaml:"key5" env:"KEY5"`
}

type Config struct {
	Key1  string `json:"key1" yaml:"key1" env:"KEY1"`
	Key2  int    `json:"key2" yaml:"key2" env:"KEY2"`
	Key3  *Field3
	Key6  string    `json:"key6" yaml:"key6" env:"KEY6"`
	Key7  string    `json:"key7" yaml:"key7" env:"KEY7"`
	Key8  []byte    `json:"key8" yaml:"key8" env:"KEY8"`
	Key9  bool      `json:"key9" yaml:"key9" env:"KEY9"`
	Key10 *Field10  `json:"key10" yaml:"key10" env:"KEY10"`
	Key11 *struct{} `json:"key11" yaml:"key11" env:"KEY11"`
	Key12 string    `json:"key12" yaml:"key12" env:"KEY12"`
}

type Field10 struct {
	valid bool
}

func (f *Field10) ParseString(s string) error {
	if s == "value10" {
		f.valid = true
		return nil
	}
	return fmt.Errorf("invalid Field10 value: %q", s)
}

func stringify(v interface{}) string {
	data, _ := json.Marshal(v)
	return string(data)
}
