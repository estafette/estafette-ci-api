package api

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	envTag = "env"
)

var (
	ErrInvalidMapItem     = errors.New("invalid map item")
	ErrLookuperNil        = errors.New("lookuper cannot be nil")
	ErrInvalidEnvvarName  = errors.New("invalid environment variable name")
	ErrMissingKey         = errors.New("missing key")
	ErrMissingRequired    = errors.New("missing required value")
	ErrNotPtr             = errors.New("input must be a pointer")
	ErrNotStruct          = errors.New("input must be a struct")
	ErrPrefixNotStruct    = errors.New("prefix is only valid on struct types")
	ErrPrivateField       = errors.New("cannot parse private fields")
	ErrRequiredAndDefault = errors.New("field cannot be required and have a default value")
	ErrUnknownOption      = errors.New("unknown option")
)

func OverrideFromEnv(config interface{}, prefix string, environmentVariables []string) error {
	return OverrideFromEnvMap(config, prefix, transformEnvironmentVariablesToMap(environmentVariables))
}

func OverrideFromEnvMap(config interface{}, prefix string, environmentVariables map[string]string) error {
	if !strings.HasSuffix(prefix, "_") {
		prefix = strings.ToUpper(prefix) + "_"
	}

	// check if any of the environment variables start with capitalized prefix + underscore
	environmentVariables = filterEnvironmentVariablesByPrefix(environmentVariables, prefix)
	if len(environmentVariables) == 0 {
		// no need to do anything
		return nil
	}

	v := reflect.ValueOf(config)
	if v.Kind() != reflect.Ptr {
		return ErrNotPtr
	}

	e := v.Elem()
	if e.Kind() != reflect.Struct {
		return ErrNotStruct
	}

	t := e.Type()

	for i := 0; i < t.NumField(); i++ {
		ef := e.Field(i)
		tf := t.Field(i)

		if !ef.CanSet() {
			continue
		}

		fieldEnvarName := prefix + strings.ToUpper(tf.Name)
		tag := tf.Tag.Get(envTag)
		if tag != "" {
			fieldEnvarName = prefix + strings.ToUpper(tag)
		}

		log.Debug().Msgf("Checking if envvar %v exists", fieldEnvarName)
		// check if environment variable for property exists
		if val, ok := environmentVariables[fieldEnvarName]; ok {
			log.Debug().Msgf("Envvar %v exists, overriding config value", fieldEnvarName)
			// set value
			if err := processField(val, ef); err != nil {
				return fmt.Errorf("%s(%q): %w", tf.Name, val, err)
			}
			continue
		}

		nestedFieldsPrefix := fieldEnvarName + "_"
		nestedEnvironmentVariables := filterEnvironmentVariablesByPrefix(environmentVariables, nestedFieldsPrefix)
		if len(nestedEnvironmentVariables) == 0 {
			continue
		}

		switch ef.Kind() {
		case reflect.Ptr:
			// init pointer if it's nil
			if ef.IsNil() {
				if ef.Type().Elem().Kind() != reflect.Struct {
					// This is a nil pointer to something that isn't a struct, like
					// *string. Move along.
					continue
				}

				// Nil pointer to a struct, create so we can traverse.
				ef.Set(reflect.New(ef.Type().Elem()))
			}

			if err := OverrideFromEnvMap(ef.Interface(), nestedFieldsPrefix, nestedEnvironmentVariables); err != nil {
				return err
			}
		case reflect.Struct:
			if err := OverrideFromEnvMap(ef.Addr().Interface(), nestedFieldsPrefix, nestedEnvironmentVariables); err != nil {
				return err
			}
		}
	}

	return nil
}

func transformEnvironmentVariablesToMap(environmentVariables []string) (environmentVariablesMap map[string]string) {
	environmentVariablesMap = make(map[string]string)

	for _, ev := range environmentVariables {
		evSplit := strings.Split(ev, "=")
		if len(evSplit) < 1 {
			continue
		}
		if len(evSplit) < 2 {
			environmentVariablesMap[evSplit[0]] = ""
			continue
		}

		environmentVariablesMap[evSplit[0]] = strings.Join(evSplit[1:], "=")
	}

	return
}

func filterEnvironmentVariablesByPrefix(environmentVariables map[string]string, prefix string) (filteredEnvironmentVariables map[string]string) {
	filteredEnvironmentVariables = make(map[string]string)

	for key, value := range environmentVariables {
		if strings.HasPrefix(key, prefix) {
			filteredEnvironmentVariables[key] = value
		}
	}

	return
}

func processField(v string, ef reflect.Value) error {
	// Handle pointers and uninitialized pointers.
	for ef.Type().Kind() == reflect.Ptr {
		if ef.IsNil() {
			ef.Set(reflect.New(ef.Type().Elem()))
		}
		ef = ef.Elem()
	}

	tf := ef.Type()
	tk := tf.Kind()

	// We don't check if the value is empty earlier, because the user might want
	// to define a custom decoder and treat the empty variable as a special case.
	// However, if we got this far, none of the remaining parsers will succeed, so
	// bail out now.
	if v == "" {
		return nil
	}

	switch tk {
	case reflect.Bool:
		b, err := strconv.ParseBool(v)
		if err != nil {
			return err
		}
		ef.SetBool(b)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(v, tf.Bits())
		if err != nil {
			return err
		}
		ef.SetFloat(f)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		i, err := strconv.ParseInt(v, 0, tf.Bits())
		if err != nil {
			return err
		}
		ef.SetInt(i)
	case reflect.Int64:
		// Special case time.Duration values.
		if tf.PkgPath() == "time" && tf.Name() == "Duration" {
			d, err := time.ParseDuration(v)
			if err != nil {
				return err
			}
			ef.SetInt(int64(d))
		} else {
			i, err := strconv.ParseInt(v, 0, tf.Bits())
			if err != nil {
				return err
			}
			ef.SetInt(i)
		}
	case reflect.String:
		ef.SetString(v)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		i, err := strconv.ParseUint(v, 0, tf.Bits())
		if err != nil {
			return err
		}
		ef.SetUint(i)

	case reflect.Interface:
		return fmt.Errorf("cannot decode into interfaces")

	// Maps
	case reflect.Map:
		vals := strings.Split(v, ",")
		mp := reflect.MakeMapWithSize(tf, len(vals))
		for _, val := range vals {
			pair := strings.SplitN(val, ":", 2)
			if len(pair) < 2 {
				return fmt.Errorf("%s: %w", val, ErrInvalidMapItem)
			}
			mKey, mVal := strings.TrimSpace(pair[0]), strings.TrimSpace(pair[1])

			k := reflect.New(tf.Key()).Elem()
			if err := processField(mKey, k); err != nil {
				return fmt.Errorf("%s: %w", mKey, err)
			}

			v := reflect.New(tf.Elem()).Elem()
			if err := processField(mVal, v); err != nil {
				return fmt.Errorf("%s: %w", mVal, err)
			}

			mp.SetMapIndex(k, v)
		}
		ef.Set(mp)

	// Slices
	case reflect.Slice:
		// Special case: []byte
		if tf.Elem().Kind() == reflect.Uint8 {
			ef.Set(reflect.ValueOf([]byte(v)))
		} else {
			vals := strings.Split(v, ",")
			s := reflect.MakeSlice(tf, len(vals), len(vals))
			for i, val := range vals {
				val = strings.TrimSpace(val)
				if err := processField(val, s.Index(i)); err != nil {
					return fmt.Errorf("%s: %w", val, err)
				}
			}
			ef.Set(s)
		}
	}

	return nil
}
