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
				Usage: "attach new directory to watch",
				Arguments: []cli.Argument{
					&cli.StringArg{
						Name: "path",
					},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					p := c.StringArg("path")
					if len(p) == 0 {
						return errors.New("path is required as first argument")
					}

					dir, err := cfg.AddDirectory(p)
					if err != nil {
						return err
					}

					return attach(dir)
				},
			},
			{
				Name:  "watch",
				Usage: "start the watch daemon",
				Arguments: []cli.Argument{
					&cli.StringArg{
						Name: "directory_name",
					},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					dirName := c.StringArg("directory")
					if len(dirName) == 0 {
						return errors.New("path is required as first argument")
					}
					dir := cfg.DirectoryFullPath(dirName)
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
