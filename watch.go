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
	_, err := exec.LookPath("inotifywait")
	if err != nil {
		return fmt.Errorf("inotify-tools is not installed")
	}

	events := make(chan inotifywaitgo.FileEvent)
	errors := make(chan error)

	go inotifywaitgo.WatchPath(&inotifywaitgo.Settings{
		Dir:        dir,
		FileEvents: events,
		ErrorChan:  errors,
		Options: &inotifywaitgo.Options{
			Recursive: false,
			Events: []inotifywaitgo.EVENT{
				inotifywaitgo.CLOSE_WRITE,
			},
			Monitor: true,
		},
		Verbose: false,
	})

	// FIXME: replace "Test" with real organized directories
	err = os.MkdirAll(path.Join(dir, "Test"), 0700)
	if err != nil {
		return fmt.Errorf("unable to create target directory: %v\n", err)
	}

readLoop:
	for {
		select {
		case event := <-events:
			for _, e := range event.Events {
				src := event.Filename
				switch e {
				case inotifywaitgo.CLOSE_WRITE:
					if !isSafeToRead(src) {
						break
					}
					baseName := path.Base(src)
					dst := path.Join(dir, "Test", baseName)
					if err := move(src, dst); err != nil {
						return err
					}
				}
			}
		case err := <-errors:
			fmt.Printf("Error: %s\n", err)
			break readLoop
		}
	}

	return nil
}

func isSafeToRead(src string) bool {
	var prevsize int64 = -1
	for i := 0; i < 10; i++ {
		f, err := os.Stat(src)
		if err != nil {
			return false
		}
		size := f.Size()
		// determine the current file is the final form of a complete file
		// by comparing the previous iteration size check with new
		if size == prevsize {
			return true
		}
		prevsize = size
		time.Sleep(500 * time.Millisecond)
	}
	return false
}

func move(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	dir := path.Dir(dst)
	base := path.Base(dst)
	ext := path.Ext(base)
	name := strings.TrimSuffix(base, ext)

	final := path.Join(dir, base)

	i := 1
	for {
		_, err := os.Stat(final)
		if errors.Is(err, fs.ErrNotExist) {
			break
		}
		if err != nil {
			return err
		}

		final = path.Join(dir, fmt.Sprintf("%s(%d)%s", name, i, ext))
		i++
	}

	return os.WriteFile(final, data, 0644)
}
