package config

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/cohesivestack/valgo"
	"gopkg.in/yaml.v3"
)

type loadConfigOptions struct {
	fs *embed.FS
}

type LoadConfigOption func(*loadConfigOptions)

func WithFS(fs embed.FS) LoadConfigOption {
	return func(o *loadConfigOptions) {
		o.fs = &fs
	}
}

type Configurable interface {
	InitDefaults()
	Validation() *valgo.Validation
}

// Load reads configuration from a YAML file and/or environment variables.
// Param `yamlFile` can be left empty if environment variables are being
// exclusively used.
func Load(yamlFile string, out Configurable, opts ...LoadConfigOption) {
	var options loadConfigOptions
	for _, opt := range opts {
		opt(&options)
	}

	err := func() error {
		out.InitDefaults()

		if yamlFile != "" {
			var file io.ReadCloser
			var err error

			if options.fs != nil {
				file, err = options.fs.Open(yamlFile)
			} else {
				file, err = os.Open(yamlFile)
			}
			if err != nil {
				return fmt.Errorf("open config file: %w", err)
			}
			defer file.Close()

			decoder := yaml.NewDecoder(file)
			if err = decoder.Decode(out); err != nil {
				return fmt.Errorf("decode config file: %w", err)
			}
		}

		if err := env.Parse(out); err != nil {
			return fmt.Errorf("parse config environment variables: %w", err)
		}

		if err := out.Validation().ToError(); err != nil {
			return err
		}

		return nil
	}()

	if err != nil {
		fmt.Fprintln(os.Stderr, "Config errors:")
		var verr *valgo.Error
		if errors.As(err, &verr) {
			for _, valErr := range verr.Errors() {
				fmt.Fprintf(os.Stderr, "  %s: %s\n", valErr.Name(), strings.Join(valErr.Messages(), ","))
			}
		} else {
			fmt.Fprintln(os.Stderr, fmt.Errorf("  %s", err.Error()))
		}
		os.Exit(1)
	}
}
