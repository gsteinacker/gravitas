// Declarations shared by mac_darwin.go and mac_darwin.m.

// Implemented in Objective-C (mac_darwin.m). They take ownership of their string arguments.
void mac_install_menu(char *spec);
void mac_open_panel(int ident, char *exts, char *title);
void mac_save_panel(int ident, char *ext, char *name, char *title);
void mac_alert(char *title, char *text);
void mac_about(char *version, char *credits);
char *mac_language(void);

// Implemented in Go (exported from mac_darwin.go).
extern void galaxyMenuAction(int tag);
extern int galaxyMenuState(int tag);
extern void galaxyPanelDone(int id, char *path);
extern void galaxyOpenFile(char *path);

// Forms (inspector, settings).
void mac_form_define(int ident, char *spec, char *title);
void mac_form_show(int ident, int show);
extern char *galaxyFormValues(int ident);
extern void galaxyFormChanged(int ident, char *id, double value);
extern void galaxyFormClosed(int ident);
extern void galaxyFormOpened(int ident);

// Thermal state (0–3) and, in bit 8, low power mode; callable from any thread.
int mac_thermal(void);
