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
						Usage:   "a shorthand or an alias for the watched directory",
						Aliases: []string{"a"},
					},
				},
				Arguments: []cli.Argument{
					&cli.StringArg{
						Name:      "path",
						Value:     "",
						UsageText: "<path>",
					},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					p := c.StringArg("path")
					if len(p) == 0 {
						return errors.New("path is required as first argument")
					}

					alias := c.String("alias")
					if len(alias) == 0 {
						return errors.New("alias flag is required")
					}

					dir, err := cfg.AddDirectory(p, alias)
					if err != nil {
						return err
					}

					return attach(dir)
				},
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
				Action: func(ctx context.Context, c *cli.Command) error {
					alias := c.StringArg("alias")
					if len(alias) == 0 {
						return errors.New("alias is required as the first argument")
					}
					if err := detach(alias); err != nil {
						return err
					}
					return cfg.RemoveDirectory(alias)
				},
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
				Action: func(ctx context.Context, c *cli.Command) error {
					dir, err := getFullPath(c, "alias")
					if err != nil {
						return err
					}
					return watch(dir)
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
}

func getFullPath(c *cli.Command, argName string) (string, error) {
	dirName := c.StringArg(argName)
	if dirName == "" {
		return "", errors.New("directory alias is required")
	}

	dir := cfg.DirectoryFullPath(dirName)
	if dir == "" {
		return "", fmt.Errorf("directory alias %q is not attached", dirName)
	}

	return dir, nil
}
