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
func attach(alias string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	bin, err := os.Executable()
	if err != nil {
		return err
	}

	unit := fmt.Sprintf("unjunk.%s.service", alias)
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
			`, bin, alias)

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
		logger.Infof("watcher for %q started and enabled", alias)
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
	logger.Infof("%q is now detached", alias)
	return os.Remove(servicePath)
}

func start(alias string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	unit := fmt.Sprintf("unjunk.%s.service", alias)
	servicePath := path.Join(home, ".config", "systemd", "user", unit)

	bak := servicePath + ".bak"
	_, err = os.Stat(bak)
	if err == nil {
		// remove the masked service
		os.Remove(servicePath)
		if err := os.Rename(bak, servicePath); err != nil {
			return fmt.Errorf("error while using backup file: %v", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := systemctl.Options{UserMode: true}

	if err := systemctl.Start(ctx, unit, opts); err != nil {
		return fmt.Errorf("failed to start %q: %w", unit, err)
	}

	logger.Infof("watcher for %q started", alias)
	return nil
}

func stop(alias string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	unit := fmt.Sprintf("unjunk.%s.service", alias)
	servicePath := path.Join(home, ".config", "systemd", "user", unit)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	opts := systemctl.Options{UserMode: true}

	if err := systemctl.Stop(ctx, unit, opts); err != nil {
		return fmt.Errorf("failed to stop %q: %w", unit, err)
	}

	if err := os.Rename(servicePath, servicePath+".bak"); err != nil {
		return err
	}

	// reload all unit file to prevent the unit still being cached after backup
	if err := systemctl.DaemonReload(ctx, opts); err != nil {
		return fmt.Errorf("failed to mask service: %w", err)
	}

	// use mask instead of just stop, since stop does not persist the service from
	// running after rebooting or after a new login session
	if err := systemctl.Mask(ctx, unit, opts); err != nil {
		// it will always throw not exist error, because we renamed the
		// original service file
		if !errors.Is(err, systemctl.ErrDoesNotExist) {
			return fmt.Errorf("failed to mask service: %w", err)
		}
	}

	logger.Infof("watcher for %q stopped", alias)
	return nil
}

func isWatcherActive(alias string) bool {
	unit := fmt.Sprintf("unjunk.%s.service", alias)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := systemctl.Options{UserMode: true}

	isActive, err := systemctl.IsRunning(ctx, unit, opts)
	if err != nil {
		return false
	}

	return isActive
}
