package configs_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/halliday/go-configs"
)

var env = map[string]string{
	"TEST_KEY7":      "value7",
	"TEST_KEY8":      "dmFsdWU4", // base64 encoded "value8"
	"TEST_KEY9":      "1",        // boolean true
	"TEST_KEY10":     "value10",
	"TEST_KEY12":     "value12_2",
	"TEST_KEY99":     "unused value",
	"TEST_KEY13_FOO": "foo",
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
	//
	Key13Foo: "foo",
}

var config = Config{
	Key1: "default1",
	Key6: "default6",
}

func TestRead(t *testing.T) {
	for k, v := range env {
		t.Setenv(k, v)
	}

	c := configs.Config{
		Value:          &config,
		EnvPrefix:      "TEST_",
		File:           "test/config1.json",
		OverwritesFile: "test/overwrite1.json",
	}
	if err := c.Read(); err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(config, expected) {
		t.Fatalf("invalid config: (expected <> got)\n%s\n%s", stringify(expected), stringify(config))
	}

	unusedEnvKeys := c.UnusedEnvKeys()
	if len(unusedEnvKeys) != 1 || unusedEnvKeys[0] != "TEST_KEY99" {
		t.Fatalf("invalid unused env keys: %v", unusedEnvKeys)
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

	Key13       string `json:"key13" yaml:"key13" env:"KEY13"`
	Key13Foo    string `json:"key13_foo" yaml:"key13_foo" env:"KEY13_FOO"`
	Key13FooTwo string `json:"key13_foo_bar" yaml:"key13_foo_bar" env:"KEY13_FOO_BAR"`
}

type Field10 struct {
	valid bool
}

func (f *Field10) UnmarshalText(text []byte) error {
	if string(text) == "value10" {
		f.valid = true
		return nil
	}
	return fmt.Errorf("invalid Field10 value: %s", text)
}

func stringify(v interface{}) string {
	data, _ := json.Marshal(v)
	return string(data)
}
