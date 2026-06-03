package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/taigrr/systemctl"
)

func validatePath(target string) (string, error) {
	p := target
	if target == "." {
		dir, err := os.Getwd()
		if err != nil {
			return "", err
		}
		p = dir
	} else {
		if p == "~" {
			return os.UserHomeDir()
		}

		if strings.HasPrefix(p, "~") {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			p = path.Join(home, p[2:])
		}

		var err error
		p, err = filepath.Abs(p)
		if err != nil {
			return "", err
		}

		fi, err := os.Stat(p)
		if err != nil {
			return "", err
		}

		if !fi.IsDir() {
			return "", fmt.Errorf("%q is not a directory", p)
		}
	}

	return p, nil
}

// Create systemd service for the attached directory
func attach(base string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	bin, err := os.Executable()
	if err != nil {
		return err
	}

	unit := fmt.Sprintf("unjunk.%s.service", base)
	servicePath := path.Join(home, ".config", "systemd", "user", unit)

	_, err = os.Stat(servicePath)
	if err == nil {
		return err
	}

	// create systemd service file if it not exist
	if errors.Is(err, fs.ErrNotExist) {
		f, err := os.OpenFile(servicePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
		s := fmt.Sprintf(`[Unit]
Description=unjunk
After=default.target

[Service]
ExecStart=%s watch %s
Restart=always
RestartSec=1
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=default.target
RequiredBy=network.target
			`, bin, base)

		f.WriteString(s)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		opts := systemctl.Options{UserMode: true}

		systemctl.Disable(ctx, unit, opts)
		if err := systemctl.Enable(ctx, unit, opts); err != nil {
			return err
		}
		if err := systemctl.Start(ctx, unit, opts); err != nil {
			return err
		}
		fmt.Println("service started and enabled")
	}

	return nil
}

func detach(alias string) error {
	unit := fmt.Sprintf("unjunk.%s.service", alias)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := systemctl.Options{UserMode: true}

	if err := systemctl.Stop(ctx, unit, opts); err != nil {
		return fmt.Errorf("failed to stop %q: %w", unit, err)
	}
	if err := systemctl.Disable(ctx, unit, opts); err != nil {
		return fmt.Errorf("failed to disable %q: %w", unit, err)
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	servicePath := filepath.Join(configDir, "systemd", "user", unit)
	return os.Remove(servicePath)
}

func start(alias string) error {
	unit := fmt.Sprintf("unjunk.%s.service", alias)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := systemctl.Options{UserMode: true}

	if err := systemctl.Start(ctx, unit, opts); err != nil {
		return fmt.Errorf("failed to start %q: %w", unit, err)
	}
	return nil
}

func stop(alias string) error {
	unit := fmt.Sprintf("unjunk.%s.service", alias)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := systemctl.Options{UserMode: true}

	if err := systemctl.Stop(ctx, unit, opts); err != nil {
		return fmt.Errorf("failed to stop %q: %w", unit, err)
	}
	return nil
}
