// Package config provides a small, reflection-based environment-variable loader.
//
// Declare a struct with tagged fields:
//
//	type Cfg struct {
//	    Port int           `env:"SERVER_PORT" required:"true"`
//	    Name string        `env:"APP_NAME" default:"fantasy-frc"`
//	    Debug bool         `env:"DEBUG" default:"false"`
//	    Timeout time.Duration `env:"TIMEOUT" default:"5s"`
//	}
//
// Then call Load(&cfg) or MustLoad(&cfg). Supported field types are string,
// int, int64, bool, time.Duration, and slices of those types (comma-separated).
package config

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Load reads environment variables into dst based on struct tags. It returns an
// error for missing required variables or values that cannot be parsed.
func Load(dst any) error {
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return fmt.Errorf("config.Load requires a non-nil pointer")
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("config.Load requires a pointer to a struct")
	}
	return loadStruct(v)
}

// MustLoad calls Load and panics on error.
func MustLoad(dst any) {
	if err := Load(dst); err != nil {
		panic(err)
	}
}

func loadStruct(v reflect.Value) error {
	t := v.Type()
	for i := range v.NumField() {
		field := v.Field(i)
		fieldType := t.Field(i)

		if fieldType.PkgPath != "" {
			// Unexported field.
			continue
		}

		if field.Kind() == reflect.Struct {
			if err := loadStruct(field); err != nil {
				return err
			}
			continue
		}

		envKey := fieldType.Tag.Get("env")
		if envKey == "" {
			continue
		}

		defaultVal := fieldType.Tag.Get("default")
		required := fieldType.Tag.Get("required") == "true"

		raw, present := os.LookupEnv(envKey)
		if !present {
			raw = defaultVal
		}
		if raw == "" {
			if required {
				return fmt.Errorf("required environment variable %s is not set", envKey)
			}
			continue
		}

		if err := setField(field, raw); err != nil {
			return fmt.Errorf("environment variable %s has invalid value: %w", envKey, err)
		}
	}
	return nil
}

func setField(field reflect.Value, raw string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(raw)
	case reflect.Int, reflect.Int64:
		if field.Type() == reflect.TypeOf(time.Duration(0)) {
			d, err := time.ParseDuration(raw)
			if err != nil {
				return fmt.Errorf("invalid duration %q: %w", raw, err)
			}
			field.Set(reflect.ValueOf(d))
			return nil
		}
		val, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid integer %q: %w", raw, err)
		}
		field.SetInt(val)
	case reflect.Bool:
		val, err := strconv.ParseBool(raw)
		if err != nil {
			return fmt.Errorf("invalid bool %q: %w", raw, err)
		}
		field.SetBool(val)
	case reflect.Slice:
		parts := strings.Split(raw, ",")
		return setSlice(field, parts)
	default:
		return fmt.Errorf("unsupported config type %s", field.Type())
	}
	return nil
}

func setSlice(field reflect.Value, parts []string) error {
	slice := reflect.MakeSlice(field.Type(), len(parts), len(parts))
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if err := setField(slice.Index(i), part); err != nil {
			return err
		}
	}
	field.Set(slice)
	return nil
}
