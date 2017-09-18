package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"gopkg.in/yaml.v2"
)

// globalConfig is our top-level configuration object
type globalConfig struct {
	AppDir   string `app_dir` // directory with app configurations
	LogLevel string
	Listen   httpConfig

	Applications map[string]*appConfig
}

func (gc *globalConfig) init() error {
	// Load application config files
	if gc.AppDir != "" {
		fileInfos, err := ioutil.ReadDir(filepath.Join(gc.AppDir))
		if err != nil {
			return err
		}

		gc.Applications = map[string]*appConfig{}

		for _, fi := range fileInfos {
			if filepath.Ext(fi.Name()) != ".yaml" || fi.IsDir() {
				continue
			}
			path := filepath.Join(gc.AppDir, fi.Name())

			// Check whether symlink and if it points to a regular file
			if !fi.Mode().IsRegular() {
				fi2, err := os.Stat(path)
				if err != nil {
					return err
				}

				if !fi2.Mode().IsRegular() {
					continue
				}
			}

			yml, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			ac := &appConfig{}
			if err = yaml.UnmarshalStrict(yml, ac); err != nil {
				return err
			}
			// "/foo/bar.yaml" -> "bar"
			appName := filepath.Base(path[:(len(path) - 5)])
			gc.Applications[appName] = ac
		}
	}

	for name := range gc.Applications {
		if err := gc.Applications[name].init(); err != nil {
			return err
		}
	}
	return nil
}

// httpConfig represents the configuration of the http server.
type httpConfig struct {
	Address string
	Port    int
}

// appConfig represents a single nginx "application" to export log files for.
type appConfig struct {
	Format        string
	FromBeginning bool `from_beginning`
	Labels        map[string]string
	LogFiles      []string `log_files`

	Exclude []filterConfig
	Include []filterConfig
	Replace []replaceConfig

	orderedLabelNames  []string
	orderedLabelValues []string
}

// compileRegexes compiles the various regex strings in appConfig.
func (ac *appConfig) compileRegexes() error {
	for i := range (*ac).Exclude {
		if err := (*ac).Exclude[i].compileRegex(); err != nil {
			return err
		}
	}

	for i := range (*ac).Include {
		if err := (*ac).Include[i].compileRegex(); err != nil {
			return err
		}
	}

	for i := range (*ac).Replace {
		if err := (*ac).Replace[i].compileRegex(); err != nil {
			return err
		}
	}
	return nil
}

// init prepares an application config for use
func (ac *appConfig) init() error {
	for _, lf := range ac.LogFiles {
		if !filepath.IsAbs(lf) {
			return errors.New(fmt.Sprintf("log file '%s': not an absolute path", lf))
		}
	}

	keys := make([]string, len(ac.Labels))
	values := make([]string, len(ac.Labels))

	for k := range ac.Labels {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for i, k := range keys {
		values[i] = ac.Labels[k]
	}

	ac.orderedLabelNames = keys
	ac.orderedLabelValues = values

	if err := ac.compileRegexes(); err != nil {
		return err
	}
	return nil
}

// filterConfig represents an include or exclude filter for paths.
type filterConfig struct {
	Path    *string
	Methods []string

	pathRe *regexp.Regexp
}

// compileRegex compiles the path field regex of filterConfig.
func (fc *filterConfig) compileRegex() error {
	var err error
	if fc.pathRe, err = regexp.Compile(*fc.Path); err != nil {
		return err
	}
	return nil
}

// match checks if a method/path combination matches the filter.
func (fc *filterConfig) match(method string, path string) bool {
	if !fc.pathRe.MatchString(path) {
		return false
	}

	if len(fc.Methods) == 0 {
		return true
	}

	for i := range fc.Methods {
		if method == fc.Methods[i] {
			return true
		}
	}
	return false
}

// replaceConfig represents a replacement option for paths.
type replaceConfig struct {
	filterConfig `,inline`

	With string
}

// replace performs replaceConfig's string replacement.
func (rc *replaceConfig) replace(s string) string {
	return rc.pathRe.ReplaceAllString(s, rc.With)
}

// newConfig creates a new global configuration from configuration file
func newConfig(fileName string) (*globalConfig, error) {
	yml, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	gc := &globalConfig{
		LogLevel: "INFO",
		Listen: httpConfig{
			Address: "0.0.0.0",
			Port:    9900,
		},
	}
	if err = yaml.UnmarshalStrict(yml, gc); err != nil {
		return nil, err
	}

	if err = gc.init(); err != nil {
		return nil, err
	}

	return gc, nil
}
