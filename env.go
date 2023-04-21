package configs

import (
	"encoding"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

func (c *Config) ReadFromEnv(i any, prefix string) error {
	v := reflect.ValueOf(i)
	c.usedEnvKeys = make([]string, 0)

	var errs []error

	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, prefix) {
			continue
		}
		i := strings.Index(env, "=")
		if i == -1 {
			continue
		}
		key, value := env[len(prefix):i], env[i+1:]
		ok, err := setEnvText(v, key, value)
		if err != nil {
			errs = append(errs, fmt.Errorf("env %q: %w", key, err))
			continue
		}
		if ok {
			c.usedEnvKeys = append(c.usedEnvKeys, prefix+key)
		}
	}

	return errors.Join(errs...)
}

func setEnvText(v reflect.Value, key string, value string) (ok bool, err error) {
	if key == "" {
		return true, assignString(v, value)
	}
	if !matchesEnvKey(v.Type(), key) {
		return false, nil
	}
	v = deepNew(v)
	if v.Kind() != reflect.Struct {
		return false, nil
	}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag, err := ParseEnvTag(f.Tag.Get("env"))
		if err != nil {
			return false, fmt.Errorf("bad env tag on %T: %q", v.Type(), tag)
		}
		if tag.Name == "-" {
			continue
		}
		if tag.Name == "" {
			tag.Name = strings.ToUpper(f.Name)
		}
		if key == tag.Name {
			return true, assignString(v.Field(i), value)
		}
		if strings.HasPrefix(key, tag.Name+"_") {
			ok, err := setEnvText(v.Field(i), key[len(tag.Name)+1:], value)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
	}
	return false, nil
}

// func (c *Config) ReadFromEnv(i any, prefix string) error {
// 	v := reflect.ValueOf(i)
// 	c.UsedEnvKeys = make([]string, 0)
// 	return c.readFromEnv(v, prefix)
// }

// func (c *Config) readFromEnv(v reflect.Value, prefix string) error {
// 	if !hasEnv(v.Type(), prefix) {
// 		return nil
// 	}
// 	v = deepNew(v)
// 	t := v.Type()

// 	for i := 0; i < t.NumField(); i++ {
// 		f := t.Field(i)

// 		tag, err := ParseEnvTag(f.Tag.Get("env"))
// 		if err != nil {
// 			return fmt.Errorf("bad env tag on %T: %q", v.Type(), tag)
// 		}
// 		if tag.Name == "-" {
// 			continue
// 		}
// 		if tag.Name == "" {
// 			tag.Name = strings.ToUpper(f.Name)
// 		}
// 		key := prefix + tag.Name
// 		val, ok := os.LookupEnv(key)
// 		if ok {
// 			if err := assignString(v.Field(i), val); err != nil {
// 				return fmt.Errorf("failed to parse env %q: %v", tag.Name, err)
// 			}
// 			c.UsedEnvKeys = append(c.UsedEnvKeys, key)
// 			continue
// 		}
// 		if err := c.readFromEnv(v.Field(i), key+"_"); err != nil {
// 			return err
// 		}
// 	}

// 	return nil
// }

func matchesEnvKey(t reflect.Type, key string) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return false
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag, err := ParseEnvTag(f.Tag.Get("env"))
		if err != nil {
			return false
		}
		if tag.Name == "-" {
			continue
		}
		if tag.Name == "" {
			tag.Name = strings.ToUpper(f.Name)
		}
		if tag.Name == key {
			return true
		}
		if strings.HasPrefix(key, tag.Name+"_") {
			if matchesEnvKey(f.Type, key[len(tag.Name)+1:]) {
				return true
			}
		}
	}
	return false
}

type EnvTag struct {
	Name string
}

func ParseEnvTag(tag string) (EnvTag, error) {
	return EnvTag{Name: tag}, nil
}

func deepNew(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}
	return v
}

var byteSliceType = reflect.TypeOf((*[]byte)(nil)).Elem()

func assignString(v reflect.Value, str string) error {
	v = deepNew(v)

	i := v.Addr().Interface()
	if p, ok := i.(encoding.TextUnmarshaler); ok {
		return p.UnmarshalText([]byte(str))
	}

	if v.Type() == byteSliceType {
		data, err := base64.StdEncoding.DecodeString(str)
		if err != nil {
			return err
		}
		v.Set(reflect.ValueOf(data))
		return nil
	}

	switch v.Kind() {
	case reflect.String:
		v.SetString(str)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(str, 10, 64)
		if err != nil {
			return err
		}
		v.SetInt(i)
		return nil
	case reflect.Bool:
		b, err := strconv.ParseBool(str)
		if err != nil {
			return err
		}
		v.SetBool(b)
		return nil
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(str, 64)
		if err != nil {
			return err
		}
		v.SetFloat(f)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u, err := strconv.ParseUint(str, 10, 64)
		if err != nil {
			return err
		}
		v.SetUint(u)
		return nil
	}
	return fmt.Errorf("unsupported type %s", v.Type())
}

var EnvFile = ".env"

func LoadDotenv() error {
	dotenv, err := os.ReadFile(EnvFile)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(dotenv), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		i := strings.IndexByte(line, '=')
		if i < 0 {
			return fmt.Errorf("config: bad line in .env: %q", line)
		}
		key := strings.TrimSpace(line[:i])
		val := strings.TrimSpace(line[i+1:])
		if err := os.Setenv(key, val); err != nil {
			return err
		}
	}
	return nil
}
