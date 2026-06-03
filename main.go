//go:build linux

package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
)

func main() {
	if err := loadConfig(); err != nil {
		panic(err)
	}

	cmd := &cli.Command{
		Name:        "unjunk",
		Description: "a daemon to keep your folder organized",
		Commands: []*cli.Command{
			{
				Name:  "attach",
				Usage: "attaches new directory to watch",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "alias",
						Aliases: []string{"a"},
						Usage:   "a shorthand or an alias for the watched directory",
					},
				},
				Arguments: []cli.Argument{
					&cli.StringArg{
						Name:      "path",
						UsageText: "<path>",
					},
				},
				Action: cmdAttach,
			},
			{
				Name:  "detach",
				Usage: "detach existing watcher from watching the directory",
				Arguments: []cli.Argument{
					&cli.StringArg{
						Name:      "alias",
						UsageText: "<alias>",
					},
				},
				Action: cmdDetach,
			},
			{
				Name:  "start",
				Usage: "start directory watcher",
				Arguments: []cli.Argument{
					&cli.StringArg{
						Name:      "alias",
						UsageText: "<alias>",
					},
				},
				Action: cmdStart,
			},
			{
				Name:  "stop",
				Usage: "stop directory watcher",
				Arguments: []cli.Argument{
					&cli.StringArg{
						Name:      "alias",
						UsageText: "<alias>",
					},
				},
				Action: cmdStop,
			},
			{
				Name:  "watch",
				Usage: "start the watch daemon",
				Arguments: []cli.Argument{
					&cli.StringArg{
						Name:      "alias",
						UsageText: "<alias>",
					},
				},
				Action: cmdWatch,
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func cmdAttach(_ context.Context, c *cli.Command) error {
	p := c.StringArg("path")
	if p == "" {
		return errors.New("path is required as first argument")
	}
	alias := c.String("alias")
	if alias == "" {
		return errors.New("alias flag is required")
	}
	dir, err := cfg.AddDirectory(p, alias)
	if err != nil {
		return err
	}
	return attach(dir)
}

func cmdDetach(_ context.Context, c *cli.Command) error {
	alias := c.StringArg("alias")
	if alias == "" {
		return errors.New("alias is required as the first argument")
	}
	if err := detach(alias); err != nil {
		return err
	}
	return cfg.RemoveDirectory(alias)
}

func cmdWatch(_ context.Context, c *cli.Command) error {
	dir, err := getFullPath(c, "alias")
	if err != nil {
		return err
	}
	return watch(dir)
}

func cmdStart(_ context.Context, c *cli.Command) error {
	alias := c.StringArg("alias")
	if alias == "" {
		return errors.New("directory alias is required")
	}
	if exists := cfg.DirectoryAliasExists(alias); !exists {
		return errors.New("directory alias %q does not exist")
	}

	return start(alias)
}

func cmdStop(_ context.Context, c *cli.Command) error {
	alias := c.StringArg("alias")
	if alias == "" {
		return errors.New("directory alias is required")
	}
	if exists := cfg.DirectoryAliasExists(alias); !exists {
		return errors.New("directory alias %q does not exist")
	}

	return stop(alias)
}

func getFullPath(c *cli.Command, argName string) (string, error) {
	alias := c.StringArg(argName)
	if alias == "" {
		return "", errors.New("directory alias is required")
	}
	dir := cfg.DirectoryFullPath(alias)
	if dir == "" {
		return "", fmt.Errorf("directory alias %q is not attached", alias)
	}
	return dir, nil
}
