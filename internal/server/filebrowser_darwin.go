package server

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// This asks LaunchServices for the HTTPS handler; it does not open or fetch the
// lookup URL. No caller-supplied text is interpolated into JavaScript.
const defaultBrowserScript = `ObjC.import('AppKit'); var app = $.NSWorkspace.sharedWorkspace.URLForApplicationToOpenURL($.NSURL.URLWithString('https://example.invalid')); if (app.isNil()) { throw new Error('No default web browser'); } ObjC.unwrap(app.path);`

func openHTMLInBrowser(path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return openHTMLWithDefaultBrowser(ctx, path, func(cmd *exec.Cmd) ([]byte, error) { return cmd.Output() })
}

// The runner seam proves the native argv boundary without launching apps.
func openHTMLWithDefaultBrowser(ctx context.Context, path string, run func(*exec.Cmd) ([]byte, error)) error {
	out, err := run(exec.CommandContext(ctx, osascriptPath, "-l", "JavaScript", "-e", defaultBrowserScript))
	if err != nil {
		return errors.New("default web browser lookup failed")
	}
	app := strings.TrimSuffix(string(out), "\n")
	if !filepath.IsAbs(app) || strings.ContainsRune(app, 0) || !strings.EqualFold(filepath.Ext(app), ".app") {
		return errors.New("default web browser lookup returned no application")
	}
	if _, err := run(exec.CommandContext(ctx, "/usr/bin/open", "-a", app, path)); err != nil {
		return errors.New("default web browser launch failed")
	}
	return nil
}
