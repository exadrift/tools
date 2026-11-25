package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type Set map[string]struct{}

func (s Set) Contains(v string) bool {
	_, ok := s[v]
	return ok
}

type ConfigOptionInterface interface {
	Name() string
	Props() map[string]any
}

type Options struct {
	// errorOnMissing will cause Apply to return an error if an environment variable is missing and has no default value
	errorOnMissing bool
}

type ConfigOption func(*Options)

// WithErrorOnMissing will cause the Apply function to return an error if any environment config is missing and has no default value
func WithErrorOnMissing() ConfigOption {
	return func(o *Options) {
		o.errorOnMissing = true
	}
}

func setDuration(fv reflect.Value, value string) error {
	var scalarPortion string
	var typePortion string
	for i := 0; i < len(value); i++ {
		if value[i] < 48 || value[i] > 57 {
			scalarPortion = value[:i]
			typePortion = value[i:]
			break
		}
	}
	s, err := strconv.ParseInt(scalarPortion, 10, 64)
	if err != nil {
		return err
	}

	var v int64

	switch typePortion {
	case "h":
		v = int64(time.Hour) * s
	case "m":
		v = int64(time.Minute) * s
	case "s":
		v = int64(time.Second) * s
	case "ms":
		v = int64(time.Millisecond) * s
	case "us":
		v = int64(time.Microsecond) * s
	case "ns":
		v = int64(time.Nanosecond) * s
	case "d":
		v = int64(time.Hour) * 24 * s
	case "w":
		v = int64(time.Hour) * 24 * 7 * s
	default:
		return fmt.Errorf("duration string had unknown value: '%s'", typePortion)
	}

	fv.SetInt(v)
	return nil
}

func setInt(fv reflect.Value, value string) error {
	v, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return err
	}

	fv.SetInt(v)
	return nil
}

func setFloat(fv reflect.Value, value string) error {
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return err
	}

	fv.SetFloat(v)
	return nil
}

func setBool(fv reflect.Value, value string) error {
	var bv bool
	lv := strings.ToLower(value)
	switch lv {
	case "t", "true", "1":
		bv = true
	case "f", "false", "0":
		bv = false
	default:
		return fmt.Errorf("string was not clearly representative of a boolean value")
	}

	fv.SetBool(bv)
	return nil
}

func setByteArray(fv reflect.Value, value string, isEncoded bool) error {
	if !isEncoded {
		fv.SetBytes([]byte(value))
		return nil
	}

	decBytes, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return fmt.Errorf("environment variable was not encoded in standard base64")
	}

	fv.SetBytes(decBytes)
	return nil
}

func setSet(fv reflect.Value, value string) error {
	newSet := Set{}
	items := strings.Split(value, ",")
	for _, item := range items {
		if len(item) > 0 {
			newSet[item] = struct{}{}
		}
	}

	fv.Set(reflect.ValueOf(newSet))

	return nil
}

func applyValue(fv reflect.Value, envValue string, isEncoded bool) error {
	var err error
	switch fv.Kind() {
	case reflect.Int:
		err = setInt(fv, envValue)
	case reflect.Int64:
		ifx := fv.Interface()
		switch ifx.(type) {
		case time.Duration:
			err = setDuration(fv, envValue)
		case int64:
			err = setInt(fv, envValue)
		default:
			return fmt.Errorf("unknown interface type built on int64")
		}
	case reflect.Int32:
		err = setInt(fv, envValue)
	case reflect.Int16:
		err = setInt(fv, envValue)
	case reflect.Int8:
		err = setInt(fv, envValue)
	case reflect.Float64:
		err = setFloat(fv, envValue)
	case reflect.Float32:
		err = setFloat(fv, envValue)
	case reflect.String:
		fv.SetString(envValue)
	case reflect.Bool:
		err = setBool(fv, envValue)
	case reflect.Map:
		ifx := fv.Interface()
		switch ifx.(type) {
		case Set:
			err = setSet(fv, envValue)
		default:
			return fmt.Errorf("unknown interface type built on map")
		}
	case reflect.Slice:
		ifx := fv.Interface()
		switch ifx.(type) {
		case []byte:
			err = setByteArray(fv, envValue, isEncoded)
		default:
			return fmt.Errorf("unknown interface type built on array")
		}
	default:
		return fmt.Errorf("unexpected field type %s", fv.Kind().String())
	}

	return err
}

func Apply(target any, options ...ConfigOption) error {
	opt := &Options{}
	for _, o := range options {
		o(opt)
	}

	t := reflect.TypeOf(target)
	v := reflect.ValueOf(target)
	k := t.Kind()
	if k != reflect.Pointer {
		return fmt.Errorf("target must be a pointer type")
	}
	e := t.Elem()
	ve := v.Elem()
	k = e.Kind()
	if k != reflect.Struct {
		return fmt.Errorf("target must point to a struct type")
	}

	for i := 0; i < e.NumField(); i++ {
		f := ve.Field(i)
		v := e.Field(i)
		tag := v.Tag
		envVar := tag.Get("env")
		if envVar == "" {
			continue
		}

		defaultVal := tag.Get("default")
		isEnc := false
		encoded := tag.Get("encoded")
		switch encoded {
		case "true":
			isEnc = true
		case "false", "":
			isEnc = false
		default:
			return fmt.Errorf("encoded tag may only contain a value of true, false, or empty")
		}

		envValue := os.Getenv(envVar)
		if envValue == "" {
			envValue = defaultVal
		}

		if opt.errorOnMissing && envValue == "" {
			return fmt.Errorf("env var %s was not provided and has no default", envVar)
		}
		if envValue != "" {
			err := applyValue(f, envValue, isEnc)
			if err != nil {
				return fmt.Errorf("parse error for env var %s: %w", envVar, err)
			}

		}
	}

	return nil
}
