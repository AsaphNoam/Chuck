package server

import (
	"context"
	"errors"
	"os/exec"
	"reflect"
	"testing"
)

func TestHTMLDefaultBrowserDispatch(t *testing.T) {
	path := "/tmp/preview #?%;$(ignored).html"
	var commands [][]string
	err := openHTMLWithDefaultBrowser(context.Background(), path, func(cmd *exec.Cmd) ([]byte, error) {
		commands = append(commands, cmd.Args)
		if len(commands) == 1 {
			return []byte("/Applications/Chosen Browser.app\n"), nil
		}
		return nil, nil
	})
	want := [][]string{
		{"/usr/bin/osascript", "-l", "JavaScript", "-e", defaultBrowserScript},
		{"/usr/bin/open", "-a", "/Applications/Chosen Browser.app", path},
	}
	if err != nil || !reflect.DeepEqual(commands, want) {
		t.Fatalf("commands = %v, error = %v", commands, err)
	}
}

func TestHTMLDefaultBrowserFailures(t *testing.T) {
	for _, tc := range []struct {
		name, output         string
		lookupErr, launchErr error
	}{
		{name: "lookup", lookupErr: errors.New("lookup failure")},
		{name: "no browser"},
		{name: "malformed browser", output: "https://evil.example"},
		{name: "launch", output: "/Applications/Browser.app\n", launchErr: errors.New("launch failure")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			err := openHTMLWithDefaultBrowser(context.Background(), "/tmp/preview.html", func(cmd *exec.Cmd) ([]byte, error) {
				calls++
				if calls == 1 {
					return []byte(tc.output), tc.lookupErr
				}
				return nil, tc.launchErr
			})
			if err == nil || (tc.launchErr == nil && calls != 1) {
				t.Fatalf("calls = %d, error = %v", calls, err)
			}
		})
	}
}
