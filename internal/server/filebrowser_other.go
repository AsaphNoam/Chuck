//go:build !darwin

package server

import "github.com/pkg/browser"

// Other platforms use their system file URL handler. Chuck's default-web-browser
// guarantee applies to its supported macOS host.
func openHTMLInBrowser(path string) error {
	return browser.OpenURL(htmlFileURL(path))
}
