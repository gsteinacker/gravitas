//go:build !(darwin && cgo)

package mac

import "galaxy/internal/i18n"

// Without Cocoa every native call is a no-op.

const native = false

func installMenu(string)                    {}
func openPanel(int, string, string)         {}
func savePanel(int, string, string, string) {}
func alert(string, string)                  {}
func about(string, string)                  {}
func systemLanguage() i18n.Lang             { return i18n.Current() }
func defineForm(Window, string, string)     {}
func showForm(Window, int)                  {}
func thermal() (int, bool)                  { return 0, false }
