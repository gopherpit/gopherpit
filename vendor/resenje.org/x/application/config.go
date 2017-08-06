// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package application

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/kelseyhightower/envconfig"
	yaml "gopkg.in/yaml.v2"
)

// Config holds the common information for options: name and
// directories from where to load values.
type Config struct {
	Name    string
	Dirs    []string
	options []options
}

type options struct {
	name string
	o    Options
}

// NewConfig creates a new instance of Config.
func NewConfig(name string, dirs ...string) (c *Config) {
	return &Config{
		Name: name,
		Dirs: dirs,
	}
}

// Register adds new Options to the Config.
func (c *Config) Register(name string, o Options) {
	c.options = append(c.options, options{name: name, o: o})
}

// Load reads configuration values from json and yaml files
// in config directories, and also from environment variables.
func (c *Config) Load() (err error) {
	for _, o := range c.options {
		for _, dir := range c.Dirs {
			f := filepath.Join(dir, o.name+".yaml")
			if _, err := os.Stat(f); !os.IsNotExist(err) {
				if err := loadYAML(f, o.o); err != nil {
					return fmt.Errorf("load yaml %q config: %v", o.name, err)
				}
			}
			f = filepath.Join(dir, o.name+".json")
			if _, err := os.Stat(f); !os.IsNotExist(err) {
				if err := loadJSON(f, o.o); err != nil {
					return fmt.Errorf("load json %q config: %v", o.name, err)
				}
			}
		}
		prefix := strings.Replace(c.Name, "-", "_", -1)
		if strings.ToLower(c.Name) != strings.ToLower(o.name) {
			prefix += "_" + o.name
		}
		if err := envconfig.Process(prefix, o.o); err != nil {
			return fmt.Errorf("load %q env variables: %v", o.name, err)
		}
	}
	return
}

// String returns the YAML-encoded multi document representation
// of current configuration state.
func (c *Config) String() string {
	buf := []byte{}
	for _, o := range c.options {
		data, err := yaml.Marshal(o.o)
		if err != nil {
			continue
		}
		buf = append(buf, []byte("# "+o.name+"\n---\n")...)
		buf = append(buf, data...)
		buf = append(buf, []byte("\n")...)
	}
	if len(c.Dirs) > 0 {
		data, err := yaml.Marshal(c.Dirs)
		if err == nil {
			buf = append(buf, []byte("# config directories\n---\n")...)
			buf = append(buf, data...)
			buf = append(buf, []byte("\n")...)
		}
	}
	return string(buf)
}

// Options defines methods that are required for options.
type Options interface {
	VerifyAndPrepare() (err error)
}

// VerifyAndPrepare executes the same named method on options
// in config.
func (c *Config) VerifyAndPrepare() (err error) {
	for _, o := range c.options {
		err = o.o.VerifyAndPrepare()
		if err != nil {
			return
		}
	}
	return
}

// HelpError is an error type that can hold additional information
// about the error.
type HelpError struct {
	Err  error
	Help string
}

// Error implements the error interface.
func (e *HelpError) Error() string {
	return e.Err.Error()
}

func loadJSON(filename string, o interface{}) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("%s: %v", filename, err)
	}
	if err = json.Unmarshal(data, o); err != nil {
		getLineColFromOffset := func(data []byte, offset int64) (line, col int) {
			start := bytes.LastIndex(data[:offset], []byte{10}) + 1
			return bytes.Count(data[:start], []byte{10}) + 1, int(offset) - start
		}
		switch e := err.(type) {
		case *json.SyntaxError:
			line, col := getLineColFromOffset(data, e.Offset)
			return fmt.Errorf("%s:%d:%d: %v", filename, line, col, err)
		case *json.UnmarshalTypeError:
			line, col := getLineColFromOffset(data, e.Offset)
			return fmt.Errorf("%s:%d:%d: expected json %s value but got %s", filename, line, col, e.Type, e.Value)
		}
		return fmt.Errorf("%s: %v", filename, err)
	}
	return nil
}

func loadYAML(filename string, o interface{}) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("%s: %v", filename, err)
	}
	if err = yaml.Unmarshal(data, o); err != nil {
		return fmt.Errorf("%s: %v", filename, err)
	}
	return nil
}
