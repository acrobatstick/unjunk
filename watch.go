package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/pablodz/inotifywaitgo/inotifywaitgo"
)

func watch(dir string) error {
	if _, err := exec.LookPath("inotifywait"); err != nil {
		return fmt.Errorf("inotify-tools is not installed")
	}

	eventCh := make(chan inotifywaitgo.FileEvent)
	errCh := make(chan error)

	go inotifywaitgo.WatchPath(&inotifywaitgo.Settings{
		Dir:        dir,
		FileEvents: eventCh,
		ErrorChan:  errCh,
		Options: &inotifywaitgo.Options{
			Recursive: false,
			Events:    []inotifywaitgo.EVENT{inotifywaitgo.CLOSE_WRITE},
			Monitor:   true,
		},
		Verbose: false,
	})

	// FIXME: replace "Test" with real organized directories
	targetDir := path.Join(dir, "Test")
	if err := os.MkdirAll(targetDir, 0700); err != nil {
		return fmt.Errorf("unable to create target directory: %w", err)
	}

	for {
		select {
		case event := <-eventCh:
			if err := handleEvent(event, targetDir); err != nil {
				return err
			}
		case err := <-errCh:
			return fmt.Errorf("watcher error: %w", err)
		}
	}
}

func handleEvent(event inotifywaitgo.FileEvent, targetDir string) error {
	for _, e := range event.Events {
		if e != inotifywaitgo.CLOSE_WRITE {
			continue
		}

		src := event.Filename
		if !isSafeToRead(src) {
			continue
		}

		dst := path.Join(targetDir, path.Base(src))
		if err := move(src, dst); err != nil {
			return fmt.Errorf("failed to move %q: %w", src, err)
		}
	}
	return nil
}

func isSafeToRead(src string) bool {
	var prevSize int64 = -1
	for range 10 {
		info, err := os.Stat(src)
		if err != nil {
			return false
		}
		if info.Size() == prevSize {
			return true
		}
		prevSize = info.Size()
		time.Sleep(500 * time.Millisecond)
	}
	return false
}

func move(src, dst string) error {
	dst = resolveDestPath(dst)

	// try fast rename first (same filesystem)
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	// fallback: copy then delete
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read failed: %w", err)
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		return fmt.Errorf("write failed: %w", err)
	}
	return os.Remove(src)
}

// resolveDestPath returns a non-conflicting destination path,
// appending (1), (2), etc. if the file already exists.
func resolveDestPath(dst string) string {
	if _, err := os.Stat(dst); errors.Is(err, fs.ErrNotExist) {
		return dst
	}

	dir := path.Dir(dst)
	base := path.Base(dst)
	ext := path.Ext(base)
	name := strings.TrimSuffix(base, ext)

	for i := 1; ; i++ {
		candidate := path.Join(dir, fmt.Sprintf("%s(%d)%s", name, i, ext))
		if _, err := os.Stat(candidate); errors.Is(err, fs.ErrNotExist) {
			return candidate
		}
	}
}
