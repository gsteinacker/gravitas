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
