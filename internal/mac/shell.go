package mac

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"galaxy/internal/app"
	"galaxy/internal/i18n"
	"galaxy/internal/prefs"
	"galaxy/internal/scene"
)

// Hooks are the parts of a command only the front end can do. They run on the update goroutine.
type Hooks struct {
	ExportImage  func(path string) // save the next frame as PNG
	PrefsChanged func()            // profile, star count or GPU setting changed
	Recalibrate  func()            // measure the machine again
}

var hardware struct {
	sync.Mutex
	text string
}

// SetHardware sets the description of the machine in the settings window.
func SetHardware(s string) {
	hardware.Lock()
	hardware.text = s
	hardware.Unlock()
}

func hardwareText() string {
	hardware.Lock()
	defer hardware.Unlock()
	return hardware.text
}

// Version is shown in the About panel; set by main from the build.
var Version = "dev"

// Available reports whether the native shell works on this platform and build.
func Available() bool { return native }

// shell is the controller the native callbacks talk to. Cocoa calls back on the main thread, so
// everything that touches the controller goes through Controller.Post.
var shell struct {
	mu      sync.Mutex
	c       *app.Controller
	hooks   Hooks
	pending []string // documents opened before Install
}

// Init applies the system language; call it before building any user interface.
func Init() {
	if native {
		i18n.Set(systemLanguage())
	}
}

// Install replaces Ebitengine's minimal menu by the full menu bar. Call it once the window
// exists (e.g. from the first Update); it returns immediately, Cocoa builds the menu on the main
// thread.
func Install(c *app.Controller, h Hooks) {
	if !native {
		return
	}
	shell.mu.Lock()
	shell.c, shell.hooks = c, h
	pending := shell.pending
	shell.pending = nil
	shell.mu.Unlock()
	installMenu(encode(menuBar()))
	defineForm(WindowInspector, encodeForm(inspectorForm()), i18n.T("Inspektor", "Inspector"))
	defineForm(WindowSettings, encodeForm(settingsForm()), i18n.T("Einstellungen", "Settings"))
	if c.Prefs == nil || c.Prefs.InspectorVisible() {
		showForm(WindowInspector, 1)
	}
	for _, p := range pending {
		onOpenFile(p)
	}
}

func controller() (*app.Controller, Hooks) {
	shell.mu.Lock()
	defer shell.mu.Unlock()
	return shell.c, shell.hooks
}

// Alert shows a warning sheet; safe to call from any goroutine.
func Alert(title, text string) {
	if native {
		alert(title, text)
	}
}

func onMenu(tag int) {
	c, _ := controller()
	if c == nil {
		return
	}
	if tag < commandBase {
		c.PostAction(app.Action(tag))
		return
	}
	T := i18n.T
	switch Command(tag - commandBase) {
	case CmdOpenScene:
		openPanel(int(CmdOpenScene), "json", T("Szene (JSON, siehe docs/scenes.md) öffnen", "Open a scene (JSON, see docs/scenes.md)"))
	case CmdOpenState:
		openPanel(int(CmdOpenState), "galaxy", T("Gesicherten Stand öffnen", "Open a saved state"))
	case CmdSaveState:
		savePanel(int(CmdSaveState), "galaxy", fileName(c.State().SceneName, ".galaxy"), T("Stand sichern", "Save the state"))
	case CmdExportImage:
		savePanel(int(CmdExportImage), "png", fileName(c.State().SceneName, ".png"), T("Bild exportieren", "Export an image"))
	case CmdInspector:
		showForm(WindowInspector, -1)
	case CmdSettings:
		showForm(WindowSettings, 1)
	case CmdAbout:
		about(Version, T("Interaktive N-Körper-Simulation mit Schwarzen Löchern und einstellbarer Lichtgeschwindigkeit.",
			"Interactive N-body simulation with black holes and an adjustable speed of light."))
	}
}

// fileName proposes a file name for the scene, e.g. "Milky Way 2026-10-24 21.05.galaxy".
func fileName(sceneName, ext string) string {
	name := strings.Map(func(r rune) rune {
		if strings.ContainsRune(`/\:`, r) {
			return '-'
		}
		return r
	}, sceneName)
	if name == "" {
		name = "Galaxy"
	}
	return name + " " + time.Now().Format("2006-01-02 15.04") + ext
}

func menuState(tag int) int {
	c, _ := controller()
	if c == nil {
		return 0
	}
	return itemState(tag, c.State())
}

func onPanel(id int, path string) {
	c, h := controller()
	if c == nil || path == "" {
		return
	}
	switch Command(id) {
	case CmdOpenScene:
		c.Post(func(c *app.Controller) { openScene(c, path) })
	case CmdOpenState:
		c.Post(func(c *app.Controller) { openState(c, path) })
	case CmdSaveState:
		c.Post(func(c *app.Controller) { c.Save(path) })
	case CmdExportImage:
		if h.ExportImage != nil {
			c.Post(func(*app.Controller) { h.ExportImage(path) })
		}
	}
}

// onOpenFile opens a document from the Finder: a saved state or a scene.
func onOpenFile(path string) {
	shell.mu.Lock()
	c := shell.c
	if c == nil {
		shell.pending = append(shell.pending, path)
		shell.mu.Unlock()
		return
	}
	shell.mu.Unlock()
	if strings.EqualFold(filepath.Ext(path), ".json") {
		c.Post(func(c *app.Controller) { openScene(c, path) })
	} else {
		c.Post(func(c *app.Controller) { openState(c, path) })
	}
}

func openScene(c *app.Controller, path string) {
	log.Printf("open scene %s", path)
	sp, err := scene.LoadSpec(path)
	if err != nil {
		Alert(i18n.T("Szene kann nicht geöffnet werden", "Cannot open the scene"), fmt.Sprintf("%s\n\n%v", filepath.Base(path), err))
		return
	}
	c.LoadScene(sp)
}

func openState(c *app.Controller, path string) {
	log.Printf("open state %s", path)
	if err := c.Load(path); err != nil {
		Alert(i18n.T("Stand kann nicht geöffnet werden", "Cannot open the saved state"), fmt.Sprintf("%s\n\n%v", filepath.Base(path), err))
	}
}

// formValues encodes the current values of a form's controls, one line per control.
func formValues(w Window) string {
	c, _ := controller()
	if c == nil {
		return ""
	}
	cs := inspectorForm()
	if w == WindowSettings {
		cs = settingsForm()
	}
	s := c.State()
	p := c.Prefs
	if p == nil {
		p = prefs.Default()
	}
	var b strings.Builder
	for _, ctl := range cs {
		if ctl.id == "" {
			continue
		}
		v, text, enabled := formValue(ctl.id, s, p, hardwareText())
		en := "0"
		if enabled {
			en = "1"
		}
		fmt.Fprintf(&b, "%s%s%g%s%s%s%s\n", ctl.id, fieldSep, v, fieldSep, strings.ReplaceAll(text, "\n", " "), fieldSep, en)
	}
	return b.String()
}

func onForm(id string, v float64) {
	c, h := controller()
	if c != nil {
		c.Post(func(c *app.Controller) { onFormChange(c, h, id, v) })
	}
}

// onFormVisible remembers whether the user wants the inspector.
func onFormVisible(w Window, visible bool) {
	c, _ := controller()
	if c != nil && w == WindowInspector {
		c.Post(func(c *app.Controller) {
			if c.Prefs != nil && c.Prefs.InspectorVisible() != visible {
				c.Prefs.SetInspectorVisible(visible)
				c.SavePrefs()
			}
		})
	}
}

// Thermal reports the thermal state of the Mac (perf.ThermalNominal … ThermalCritical) and whether
// low power mode is on; nominal and off elsewhere.
func Thermal() (state int, lowPower bool) { return thermal() }
