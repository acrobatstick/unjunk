package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/taigrr/systemctl"
)

func detach(dir string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	unit := fmt.Sprintf("unjunk.%s.service", path.Base(dir))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := systemctl.Options{
		UserMode: true,
	}

	if err := systemctl.Stop(ctx, unit, opts); err != nil {
		return err
	}

	if err := systemctl.Disable(ctx, unit, opts); err != nil {
		return err
	}

	servicePath := path.Join(home, ".config", "systemd", "user", unit)
	return os.Remove(servicePath)
}
