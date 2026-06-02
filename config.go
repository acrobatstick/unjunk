package main

import (
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/BurntSushi/toml"
)

type (
	Config struct {
		ConfigDir   string                `toml:"config_dir"`
		Directories map[string]*Directory `toml:"directories"`
	}

	Directory struct {
		Path string `toml:"path"`
	}
)

func (c *Config) AddDirectory(p string) (string, error) {
	dir, err := validatePath(p)
	if err != nil {
		return "", err
	}

	alias := path.Base(dir)
	// FIXME: handle duplicate alias conflict since we explicitly turning
	//        the base folder name as the alias
	_, exist := c.Directories[alias]
	if !exist {
		directory := Directory{
			Path: dir,
		}
		c.Directories[alias] = &directory
	}

	if err := c.overwrite(); err != nil {
		return "", err
	}

	return alias, nil
}

func (c *Config) RemoveDirectory(p string) error {
	dir, err := validatePath(p)
	if err != nil {
		return err
	}
	alias := path.Base(dir)
	delete(c.Directories, alias)
	return c.overwrite()
}

func (c *Config) DirectoryFullPath(dirName string) string {
	v, exist := c.Directories[dirName]
	if !exist {
		return ""
	}
	return v.Path
}

func (c *Config) overwrite() error {
	data, err := toml.Marshal(&c)
	if err != nil {
		return err
	}
	return os.WriteFile(c.ConfigDir, data, 0644)
}

// Global config
var cfg Config

// Load config from existing TOML file
func loadConfig() error {
	dir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(path.Join(dir, "unjunk"), 0744); err != nil {
		return err
	}

	cfg.Directories = make(map[string]*Directory)

	p := path.Join(dir, "unjunk", "config.toml")
	if _, err := os.Stat(p); errors.Is(err, os.ErrNotExist) {
		f, err := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		f.Close()

		cfg.ConfigDir = p

		if err := cfg.overwrite(); err != nil {
			return err
		}

		fmt.Printf("new config file created at %s\n", p)
		return nil
	}

	f, err := os.Open(p)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = toml.NewDecoder(f).Decode(&cfg)
	if err != nil {
		return fmt.Errorf("could not decode config.toml file: %v\n", err)
	}

	return nil
}
