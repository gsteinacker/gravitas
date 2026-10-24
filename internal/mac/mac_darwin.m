// Cocoa side of the native shell (ADR-0009). The Go side is mac_darwin.go.

#include <stdlib.h>
#include "mac_darwin.h"
#import <Cocoa/Cocoa.h>
#import <UniformTypeIdentifiers/UniformTypeIdentifiers.h>
#import <objc/runtime.h>


@interface GalaxyMenuTarget : NSObject
@end

@implementation GalaxyMenuTarget
- (void)fire:(NSMenuItem *)item {
	galaxyMenuAction((int)item.tag);
}
- (BOOL)validateMenuItem:(NSMenuItem *)item {
	int s = galaxyMenuState((int)item.tag);
	item.state = (s & 1) ? NSControlStateValueOn : NSControlStateValueOff;
	return (s & 2) == 0;
}
@end

static GalaxyMenuTarget *menuTarget;

static NSWindow *mainWindow(void) {
	NSWindow *w = NSApp.mainWindow;
	return w != nil ? w : NSApp.keyWindow;
}


static NSEventModifierFlags parseMods(NSString *m) {
	NSEventModifierFlags f = 0;
	if ([m containsString:@"c"]) f |= NSEventModifierFlagCommand;
	if ([m containsString:@"s"]) f |= NSEventModifierFlagShift;
	if ([m containsString:@"o"]) f |= NSEventModifierFlagOption;
	if ([m containsString:@"t"]) f |= NSEventModifierFlagControl;
	return f;
}

static void dump(NSMenu *m, int depth) {
	for (NSMenuItem *it in m.itemArray) {
		if (it.isSeparatorItem) continue;
		BOOL enabled = YES;
		if (it.target == menuTarget) enabled = [menuTarget validateMenuItem:it];
		fprintf(stderr, "menu %*s%s key=%s mask=%lu tag=%ld state=%ld enabled=%d\n", depth * 2, "", it.title.UTF8String,
		        it.keyEquivalent.UTF8String, (unsigned long)it.keyEquivalentModifierMask, (long)it.tag, (long)it.state, enabled);
		if (it.submenu != nil) dump(it.submenu, depth + 1);
	}
}

// selftest prints the menu and offers some plain key presses to it, as AppKit does for every
// key-down (GALAXY_MENU_SELFTEST=1; for checks without screen access).
static void selftest(NSMenu *bar) {
	dump(bar, 0);
	NSWindow *w = mainWindow();
	for (NSString *k in @[ @"g", @"x", @"?", @"+" ]) {
		NSEvent *e = [NSEvent keyEventWithType:NSEventTypeKeyDown location:NSZeroPoint modifierFlags:0 timestamp:0
		                          windowNumber:w.windowNumber context:nil characters:k charactersIgnoringModifiers:k
		                             isARepeat:NO keyCode:0];
		fprintf(stderr, "key %s handled by menu: %d\n", k.UTF8String, [bar performKeyEquivalent:e]);
	}
	dispatch_after(dispatch_time(DISPATCH_TIME_NOW, NSEC_PER_SEC / 2), dispatch_get_main_queue(), ^{
		fprintf(stderr, "after keys:\n");
		dump(bar, 0);
	});
}

// mac_install_menu replaces the main menu by the one described in spec (see menu.go).
void mac_install_menu(char *spec) {
	NSString *s = [NSString stringWithUTF8String:spec];
	free(spec);
	dispatch_async(dispatch_get_main_queue(), ^{
		if (menuTarget == nil) {
			menuTarget = [GalaxyMenuTarget new];
		}
		NSMenu *bar = [NSMenu new];
		NSMutableArray<NSMenu *> *stack = [NSMutableArray arrayWithObject:bar];
		for (NSString *line in [s componentsSeparatedByString:@"\n"]) {
			NSArray<NSString *> *f = [line componentsSeparatedByString:@"\x1f"];
			if (f.count < 6) continue;
			unichar kind = [f[0] characterAtIndex:0];
			NSMenu *cur = stack.lastObject;
			switch (kind) {
			case 'M': {
				NSMenuItem *item = [cur addItemWithTitle:f[1] action:nil keyEquivalent:@""];
				NSMenu *sub = [[NSMenu alloc] initWithTitle:f[1]];
				item.submenu = sub;
				[stack addObject:sub];
				break;
			}
			case 'E':
				if (stack.count > 1) [stack removeLastObject];
				break;
			case 'S':
				[cur addItem:[NSMenuItem separatorItem]];
				break;
			case 'I':
			case 'A': {
				SEL sel = kind == 'I' ? @selector(fire:) : NSSelectorFromString(f[5]);
				NSMenuItem *item = [cur addItemWithTitle:f[1] action:sel keyEquivalent:f[2]];
				item.keyEquivalentModifierMask = parseMods(f[3]);
				if (kind == 'I') {
					item.target = menuTarget;
					item.tag = f[4].integerValue;
				}
				break;
			}
			case 'V': {
				NSMenuItem *item = [cur addItemWithTitle:NSLocalizedString(@"Services", nil) action:nil keyEquivalent:@""];
				NSMenu *services = [NSMenu new];
				item.submenu = services;
				NSApp.servicesMenu = services;
				break;
			}
			case 'W':
				NSApp.windowsMenu = cur;
				break;
			case 'H':
				NSApp.helpMenu = cur;
				break;
			}
		}
		NSApp.mainMenu = bar;
		if (getenv("GALAXY_MENU_SELFTEST") != NULL) {
			dispatch_after(dispatch_time(DISPATCH_TIME_NOW, NSEC_PER_SEC), dispatch_get_main_queue(), ^{
				selftest(bar);
			});
		}
	});
}

static NSArray<UTType *> *types(NSString *exts) {
	NSMutableArray<UTType *> *t = [NSMutableArray array];
	for (NSString *e in [exts componentsSeparatedByString:@","]) {
		UTType *u = [UTType typeWithFilenameExtension:e];
		if (u != nil) [t addObject:u];
	}
	return t;
}

static void finishPanel(int ident, NSModalResponse r, NSURL *url) {
	if (r == NSModalResponseOK && url != nil) {
		galaxyPanelDone(ident, (char *)url.fileSystemRepresentation);
	} else {
		galaxyPanelDone(ident, NULL);
	}
}

// mac_open_panel asks for an existing file with one of the comma-separated extensions.
void mac_open_panel(int ident, char *exts, char *title) {
	NSString *e = [NSString stringWithUTF8String:exts], *t = [NSString stringWithUTF8String:title];
	free(exts);
	free(title);
	dispatch_async(dispatch_get_main_queue(), ^{
		NSOpenPanel *p = [NSOpenPanel openPanel];
		p.message = t;
		p.allowedContentTypes = types(e);
		p.canChooseDirectories = NO;
		p.allowsMultipleSelection = NO;
		void (^done)(NSModalResponse) = ^(NSModalResponse r) { finishPanel(ident, r, p.URL); };
		NSWindow *w = mainWindow();
		if (w != nil) {
			[p beginSheetModalForWindow:w completionHandler:done];
		} else {
			[p beginWithCompletionHandler:done];
		}
	});
}

// mac_save_panel asks for a new file name with extension ext.
void mac_save_panel(int ident, char *ext, char *name, char *title) {
	NSString *e = [NSString stringWithUTF8String:ext], *n = [NSString stringWithUTF8String:name],
	         *t = [NSString stringWithUTF8String:title];
	free(ext);
	free(name);
	free(title);
	dispatch_async(dispatch_get_main_queue(), ^{
		NSSavePanel *p = [NSSavePanel savePanel];
		p.message = t;
		p.allowedContentTypes = types(e);
		p.nameFieldStringValue = n;
		void (^done)(NSModalResponse) = ^(NSModalResponse r) { finishPanel(ident, r, p.URL); };
		NSWindow *w = mainWindow();
		if (w != nil) {
			[p beginSheetModalForWindow:w completionHandler:done];
		} else {
			[p beginWithCompletionHandler:done];
		}
	});
}

void mac_alert(char *title, char *text) {
	NSString *t = [NSString stringWithUTF8String:title], *x = [NSString stringWithUTF8String:text];
	free(title);
	free(text);
	dispatch_async(dispatch_get_main_queue(), ^{
		NSAlert *a = [NSAlert new];
		a.alertStyle = NSAlertStyleWarning;
		a.messageText = t;
		a.informativeText = x;
		NSWindow *w = mainWindow();
		if (w != nil) {
			[a beginSheetModalForWindow:w completionHandler:nil];
		} else {
			[a runModal];
		}
	});
}

void mac_about(char *version, char *credits) {
	NSString *v = [NSString stringWithUTF8String:version], *c = [NSString stringWithUTF8String:credits];
	free(version);
	free(credits);
	dispatch_async(dispatch_get_main_queue(), ^{
		[NSApp orderFrontStandardAboutPanelWithOptions:@{
			NSAboutPanelOptionApplicationName : @"Galaxy",
			NSAboutPanelOptionApplicationVersion : v,
			NSAboutPanelOptionCredits : [[NSAttributedString alloc] initWithString:c],
		}];
	});
}

// mac_language is the first preferred language of the user, e.g. "de-DE".
char *mac_language(void) {
	NSString *l = NSLocale.preferredLanguages.firstObject;
	return strdup(l != nil ? l.UTF8String : "");
}

// Documents opened from the Finder arrive as application:openURLs: at the app delegate, which
// Ebitengine creates. The method is added to its class as soon as the app is about to finish
// launching, so files that started the app are not lost.
static void openURLs(id self, SEL _cmd, NSApplication *app, NSArray<NSURL *> *urls) {
	for (NSURL *u in urls) {
		if (u.isFileURL) galaxyOpenFile((char *)u.fileSystemRepresentation);
	}
}

__attribute__((constructor)) static void observeLaunch(void) {
	[[NSNotificationCenter defaultCenter]
		addObserverForName:NSApplicationWillFinishLaunchingNotification
		            object:nil
		             queue:nil
		        usingBlock:^(NSNotification *n) {
			id delegate = NSApp.delegate;
			if (delegate != nil) {
				class_addMethod([delegate class], @selector(application:openURLs:), (IMP)openURLs, "v@:@@");
			}
		}];
}

// ---- Forms: inspector panel and settings window --------------------------------------------

@interface GalaxyForm : NSObject <NSWindowDelegate>
@property(nonatomic) int ident;
@property(nonatomic, strong) NSWindow *window;
@property(nonatomic, strong) NSMutableDictionary<NSString *, NSControl *> *controls;
@property(nonatomic, strong) NSMutableDictionary<NSString *, NSTextField *> *texts;
@property(nonatomic, strong) NSMutableDictionary<NSString *, NSNumber *> *kinds;
@property(nonatomic, strong) NSTimer *timer;
@property(nonatomic) CFAbsoluteTime lastEdit;
@end

static GalaxyForm *forms[2];
static void snapshotForms(NSString *dir);

static NSTextField *label(NSString *s, BOOL bold, BOOL small) {
	NSTextField *l = [NSTextField labelWithString:s];
	CGFloat size = small ? NSFont.smallSystemFontSize : NSFont.systemFontSize;
	l.font = bold ? [NSFont boldSystemFontOfSize:size] : [NSFont systemFontOfSize:size];
	if (small) l.textColor = NSColor.secondaryLabelColor;
	return l;
}

@implementation GalaxyForm

- (void)changed:(NSControl *)sender {
	self.lastEdit = CFAbsoluteTimeGetCurrent();
	NSString *ident = sender.identifier;
	double v = 0;
	switch (self.kinds[ident].intValue) {
	case 'S': v = ((NSSlider *)sender).doubleValue; break;
	case 'C': v = ((NSButton *)sender).state == NSControlStateValueOn ? 1 : 0; break;
	case 'G': v = ((NSSegmentedControl *)sender).selectedSegment; break;
	case 'P': v = ((NSPopUpButton *)sender).indexOfSelectedItem; break;
	case 'R': v = sender.tag; ident = [ident componentsSeparatedByString:@"#"].firstObject; break;
	}
	galaxyFormChanged(self.ident, (char *)ident.UTF8String, v);
}

- (void)refresh {
	if (self.window.isVisible) [self update];
}

- (void)update {
	// Do not fight the user: skip updates while a control is being changed.
	BOOL editing = CFAbsoluteTimeGetCurrent() - self.lastEdit < 0.4;
	BOOL resize = NO;
	char *raw = galaxyFormValues(self.ident);
	NSString *all = [NSString stringWithUTF8String:raw];
	free(raw);
	for (NSString *line in [all componentsSeparatedByString:@"\n"]) {
		NSArray<NSString *> *f = [line componentsSeparatedByString:@"\x1f"];
		if (f.count < 4) continue;
		NSString *ident = f[0];
		double v = f[1].doubleValue;
		BOOL enabled = f[3].boolValue;
		NSTextField *t = self.texts[ident];
		if (t != nil && ![t.stringValue isEqualToString:f[2]]) {
			t.stringValue = f[2];
			resize = YES;
		}
		NSControl *c = self.controls[ident];
		if (c == nil) continue;
		c.enabled = enabled;
		if (editing) continue;
		switch (self.kinds[ident].intValue) {
		case 'S': if (((NSSlider *)c).doubleValue != v) ((NSSlider *)c).doubleValue = v; break;
		case 'C': ((NSButton *)c).state = v >= 0.5 ? NSControlStateValueOn : NSControlStateValueOff; break;
		case 'G': ((NSSegmentedControl *)c).selectedSegment = (NSInteger)v; break;
		case 'P': [((NSPopUpButton *)c) selectItemAtIndex:(NSInteger)v]; break;
		case 'R': if (f[2].length > 0) ((NSButton *)c).title = f[2]; break;
		}
	}
	if (resize) [self fit];
}

// fit adapts the window to its content (texts can wrap), keeping the top left corner.
- (void)fit {
	NSView *v = self.window.contentView;
	[v layoutSubtreeIfNeeded];
	NSSize want = v.fittingSize;
	NSRect r = [self.window contentRectForFrameRect:self.window.frame];
	if (fabs(want.height - r.size.height) < 1 && fabs(want.width - r.size.width) < 1) return;
	NSRect f = [self.window frameRectForContentRect:NSMakeRect(r.origin.x, NSMaxY(r) - want.height, want.width, want.height)];
	[self.window setFrame:f display:YES];
}

- (void)windowWillClose:(NSNotification *)n {
	galaxyFormClosed(self.ident);
}

- (void)build:(NSString *)spec title:(NSString *)title {
	self.controls = [NSMutableDictionary dictionary];
	self.texts = [NSMutableDictionary dictionary];
	self.kinds = [NSMutableDictionary dictionary];
	NSGridView *grid = [NSGridView gridViewWithNumberOfColumns:2 rows:0];
	grid.rowSpacing = 7;
	grid.columnSpacing = 10;
	for (NSString *line in [spec componentsSeparatedByString:@"\n"]) {
		NSArray<NSString *> *f = [line componentsSeparatedByString:@"\x1f"];
		if (f.count < 6) continue;
		unichar kind = [f[0] characterAtIndex:0];
		NSString *ident = f[1], *t = f[2];
		NSArray<NSString *> *opts = f[3].length > 0 ? [f[3] componentsSeparatedByString:@"|"] : @[];
		self.kinds[ident] = @(kind);
		NSView *left = t.length > 0 ? label(t, NO, NO) : [NSGridCell emptyContentView];
		NSView *right = nil;
		switch (kind) {
		case 'L': {
			NSGridRow *row = [grid addRowWithViews:@[ label(t, YES, NO) ]];
			row.topPadding = 10;
			[row mergeCellsInRange:NSMakeRange(0, 2)];
			[row cellAtIndex:0].xPlacement = NSGridCellPlacementLeading;
			continue;
		}
		case 'S': {
			NSSlider *s = [NSSlider sliderWithValue:f[4].doubleValue minValue:f[4].doubleValue maxValue:f[5].doubleValue
			                                 target:self action:@selector(changed:)];
			s.continuous = YES;
			[s.widthAnchor constraintGreaterThanOrEqualToConstant:150].active = YES;
			NSTextField *val = label(@"", NO, YES);
			[val.widthAnchor constraintEqualToConstant:70].active = YES;
			self.texts[ident] = val;
			NSStackView *h = [NSStackView stackViewWithViews:@[ s, val ]];
			right = h;
			self.controls[ident] = s;
			s.identifier = ident;
			break;
		}
		case 'C': {
			NSButton *b = [NSButton checkboxWithTitle:t target:self action:@selector(changed:)];
			b.identifier = ident;
			self.controls[ident] = b;
			left = [NSGridCell emptyContentView];
			right = b;
			break;
		}
		case 'G': {
			NSSegmentedControl *g = [NSSegmentedControl segmentedControlWithLabels:opts
			                                                           trackingMode:NSSegmentSwitchTrackingSelectOne
			                                                                 target:self
			                                                                 action:@selector(changed:)];
			g.segmentDistribution = NSSegmentDistributionFillEqually;
			g.identifier = ident;
			self.controls[ident] = g;
			right = g;
			break;
		}
		case 'P': {
			NSPopUpButton *p = [NSPopUpButton popUpButtonWithMenu:[NSMenu new] target:self action:@selector(changed:)];
			[p addItemsWithTitles:opts];
			p.identifier = ident;
			self.controls[ident] = p;
			right = p;
			break;
		}
		case 'R': {
			NSMutableArray<NSView *> *buttons = [NSMutableArray array];
			for (NSUInteger i = 0; i < opts.count; i++) {
				NSButton *b = [NSButton buttonWithTitle:opts[i] target:self action:@selector(changed:)];
				b.tag = (NSInteger)i;
				b.identifier = [NSString stringWithFormat:@"%@#%lu", ident, (unsigned long)i];
				self.kinds[b.identifier] = @('R');
				if (i == 0) self.controls[ident] = b;
				[buttons addObject:b];
			}
			right = [NSStackView stackViewWithViews:buttons];
			break;
		}
		case 'V': {
			NSTextField *v = [NSTextField wrappingLabelWithString:@""];
			v.preferredMaxLayoutWidth = 260;
			[v.widthAnchor constraintLessThanOrEqualToConstant:260].active = YES;
			self.texts[ident] = v;
			right = v;
			break;
		}
		}
		[grid addRowWithViews:@[ left, right ]];
	}
	[grid columnAtIndex:0].xPlacement = NSGridCellPlacementTrailing;
	grid.translatesAutoresizingMaskIntoConstraints = NO;

	NSView *content = [NSView new];
	[content addSubview:grid];
	[NSLayoutConstraint activateConstraints:@[
		[grid.topAnchor constraintEqualToAnchor:content.topAnchor constant:14],
		[grid.bottomAnchor constraintEqualToAnchor:content.bottomAnchor constant:-16],
		[grid.leadingAnchor constraintEqualToAnchor:content.leadingAnchor constant:16],
		[grid.trailingAnchor constraintEqualToAnchor:content.trailingAnchor constant:-16],
	]];

	if (self.ident == 0) {
		NSPanel *p = [[NSPanel alloc] initWithContentRect:NSMakeRect(0, 0, 360, 600)
		                                        styleMask:NSWindowStyleMaskTitled | NSWindowStyleMaskClosable |
		                                                  NSWindowStyleMaskUtilityWindow | NSWindowStyleMaskHUDWindow
		                                          backing:NSBackingStoreBuffered
		                                            defer:NO];
		p.floatingPanel = YES;
		p.becomesKeyOnlyIfNeeded = YES;
		p.hidesOnDeactivate = YES;
		self.window = p;
	} else {
		self.window = [[NSWindow alloc] initWithContentRect:NSMakeRect(0, 0, 420, 300)
		                                          styleMask:NSWindowStyleMaskTitled | NSWindowStyleMaskClosable
		                                            backing:NSBackingStoreBuffered
		                                              defer:NO];
	}
	self.window.releasedWhenClosed = NO;
	self.window.title = title;
	self.window.contentView = content;
	self.window.delegate = self;
	[self.window setContentSize:content.fittingSize];
	self.window.frameAutosaveName = self.ident == 0 ? @"GalaxyInspector" : @"GalaxySettings";
	self.timer = [NSTimer scheduledTimerWithTimeInterval:0.25 target:self selector:@selector(refresh) userInfo:nil repeats:YES];
}

- (void)place {
	if ([self.window setFrameUsingName:self.window.frameAutosaveName]) return;
	NSWindow *main = mainWindow();
	if (self.ident == 0 && main != nil) {
		NSRect m = main.frame, f = self.window.frame;
		[self.window setFrameTopLeftPoint:NSMakePoint(NSMaxX(m) - f.size.width - 20, NSMaxY(m) - 60)];
	} else {
		[self.window center];
	}
}
@end

void mac_form_define(int ident, char *spec, char *title) {
	NSString *s = [NSString stringWithUTF8String:spec], *t = [NSString stringWithUTF8String:title];
	free(spec);
	free(title);
	dispatch_async(dispatch_get_main_queue(), ^{
		GalaxyForm *f = [GalaxyForm new];
		f.ident = ident;
		[f build:s title:t];
		forms[ident] = f;
		char *dir = getenv("GALAXY_FORM_SNAPSHOT");
		if (dir != NULL && ident == 1) {
			NSString *d = [NSString stringWithUTF8String:dir];
			dispatch_after(dispatch_time(DISPATCH_TIME_NOW, 2 * NSEC_PER_SEC), dispatch_get_main_queue(), ^{
				snapshotForms(d);
			});
		}
	});
}

// snapshotForms writes each form window as PNG into dir (GALAXY_FORM_SNAPSHOT; for checks without
// screen access).
static void snapshotForms(NSString *dir) {
	for (int i = 0; i < 2; i++) {
		GalaxyForm *f = forms[i];
		if (f == nil) continue;
		[f update];
		[f fit];
		NSView *v = f.window.contentView;
		[v layoutSubtreeIfNeeded];
		NSBitmapImageRep *rep = [v bitmapImageRepForCachingDisplayInRect:v.bounds];
		NSGraphicsContext *ctx = [NSGraphicsContext graphicsContextWithBitmapImageRep:rep];
		[NSGraphicsContext saveGraphicsState];
		NSGraphicsContext.currentContext = ctx;
		[v.effectiveAppearance performAsCurrentDrawingAppearance:^{
			NSColor *bg = i == 0 ? [NSColor colorWithWhite:0.12 alpha:1] : NSColor.windowBackgroundColor;
			[bg setFill];
			NSRectFill(v.bounds);
		}];
		[NSGraphicsContext restoreGraphicsState];
		[v displayRectIgnoringOpacity:v.bounds inContext:ctx];
		NSData *png = [rep representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
		NSString *path = [dir stringByAppendingPathComponent:[NSString stringWithFormat:@"form%d.png", i]];
		[png writeToFile:path atomically:YES];
		fprintf(stderr, "form %d snapshot %s (%.0fx%.0f)\n", i, path.UTF8String, v.bounds.size.width, v.bounds.size.height);
	}
	// Click a checkbox like a user and check that the change made the round trip through Go.
	NSButton *trails = (NSButton *)forms[0].controls[@"trails"];
	long before = (long)trails.state;
	[trails performClick:nil];
	dispatch_after(dispatch_time(DISPATCH_TIME_NOW, NSEC_PER_SEC), dispatch_get_main_queue(), ^{
		forms[0].lastEdit = 0;
		[forms[0] update];
		fprintf(stderr, "click trails: %ld -> %ld\n", before, (long)trails.state);
	});
}

// mac_form_show shows (1), hides (0) or toggles (-1) a form window.
void mac_form_show(int ident, int show) {
	dispatch_async(dispatch_get_main_queue(), ^{
		GalaxyForm *f = forms[ident];
		if (f == nil) return;
		BOOL visible = f.window.isVisible;
		BOOL want = show < 0 ? !visible : show != 0;
		if (want && !visible) {
			galaxyFormOpened(ident);
			[f place];
			[f refresh];
			if (ident == 0) {
				[f.window orderFront:nil];
			} else {
				[f.window makeKeyAndOrderFront:nil];
			}
		} else if (!want && visible) {
			[f.window performClose:nil];
		}
	});
}
