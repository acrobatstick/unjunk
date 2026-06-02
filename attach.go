package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"
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

		if _, err := os.Stat(p); errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("%q is not a valid path directory", p)
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

	servicePath := path.Join(
		home,
		".config",
		"systemd",
		"user",
		fmt.Sprintf("unjunk.%s.service", base),
	)
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
		fmt.Println("systemd service file created")
	}
	return nil
}
