//go:build darwin && cgo

package mac

/*
#cgo CFLAGS: -fobjc-arc -Wno-deprecated-declarations
#cgo LDFLAGS: -framework Cocoa -framework UniformTypeIdentifiers
#include <stdlib.h>
#include "mac_darwin.h"
*/
import "C"

import (
	"unsafe"

	"galaxy/internal/i18n"
)

func installMenu(spec string) { C.mac_install_menu(C.CString(spec)) }

func openPanel(id int, exts, title string) {
	C.mac_open_panel(C.int(id), C.CString(exts), C.CString(title))
}

func savePanel(id int, ext, name, title string) {
	C.mac_save_panel(C.int(id), C.CString(ext), C.CString(name), C.CString(title))
}

func alert(title, text string) { C.mac_alert(C.CString(title), C.CString(text)) }

func about(version, credits string) { C.mac_about(C.CString(version), C.CString(credits)) }

func systemLanguage() i18n.Lang {
	l := C.mac_language()
	defer C.free(unsafe.Pointer(l))
	return i18n.FromLocale(C.GoString(l))
}

func defineForm(w Window, spec, title string) {
	C.mac_form_define(C.int(w), C.CString(spec), C.CString(title))
}

func showForm(w Window, show int) { C.mac_form_show(C.int(w), C.int(show)) }

//export galaxyFormValues
func galaxyFormValues(w C.int) *C.char { return C.CString(formValues(Window(w))) }

//export galaxyFormChanged
func galaxyFormChanged(w C.int, id *C.char, v C.double) { onForm(C.GoString(id), float64(v)) }

//export galaxyFormClosed
func galaxyFormClosed(w C.int) { onFormVisible(Window(w), false) }

//export galaxyFormOpened
func galaxyFormOpened(w C.int) { onFormVisible(Window(w), true) }

//export galaxyMenuAction
func galaxyMenuAction(tag C.int) { onMenu(int(tag)) }

//export galaxyMenuState
func galaxyMenuState(tag C.int) C.int { return C.int(menuState(int(tag))) }

//export galaxyPanelDone
func galaxyPanelDone(id C.int, path *C.char) {
	p := ""
	if path != nil {
		p = C.GoString(path)
	}
	onPanel(int(id), p)
}

//export galaxyOpenFile
func galaxyOpenFile(path *C.char) { onOpenFile(C.GoString(path)) }

const native = true
