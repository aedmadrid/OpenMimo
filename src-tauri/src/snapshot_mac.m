#import <Cocoa/Cocoa.h>
#import <WebKit/WebKit.h>

typedef void (*SnapshotCallback)(int status);

static void apply_traffic_light_layout(NSWindow *window, CGFloat headerHeight) {
    if (!window) return;
    NSButton *closeBtn = [window standardWindowButton:NSWindowCloseButton];
    NSButton *minBtn   = [window standardWindowButton:NSWindowMiniaturizeButton];
    NSButton *zoomBtn  = [window standardWindowButton:NSWindowZoomButton];
    if (!closeBtn || !minBtn || !zoomBtn) return;

    NSView *titlebarView = closeBtn.superview;
    NSView *container = titlebarView.superview;
    if (!titlebarView || !container) return;

    // Expand titlebar container to match our CSD header height so traffic lights sit cleanly centered vertically
    NSRect cFrame = container.frame;
    CGFloat winH = window.contentView.superview.frame.size.height;
    if (winH > headerHeight) {
        cFrame.size.height = headerHeight;
        cFrame.origin.y = winH - headerHeight;
        container.frame = cFrame;
    }

    NSRect tFrame = titlebarView.frame;
    tFrame.size.height = headerHeight;
    tFrame.origin.y = 0;
    titlebarView.frame = tFrame;

    CGFloat btnH = closeBtn.frame.size.height;
    CGFloat targetY = floor((headerHeight - btnH) / 2.0);
    CGFloat xStart = 14.0;
    CGFloat gap = 20.0;

    NSButton *btns[3] = { closeBtn, minBtn, zoomBtn };
    for (int i = 0; i < 3; i++) {
        NSRect f = btns[i].frame;
        f.origin.x = xStart + i * gap;
        f.origin.y = targetY;
        btns[i].frame = f;
    }
}

void openmimo_center_traffic_lights(void *ns_window_ptr, double header_height) {
    if (!ns_window_ptr) return;
    NSWindow *window = (__bridge NSWindow *)ns_window_ptr;
    CGFloat hh = (CGFloat)header_height;
    dispatch_async(dispatch_get_main_queue(), ^{
        apply_traffic_light_layout(window, hh);
        [[NSNotificationCenter defaultCenter]
            addObserverForName:NSWindowDidResizeNotification
                        object:window
                         queue:[NSOperationQueue mainQueue]
                    usingBlock:^(NSNotification * _Nonnull note) {
                        (void)note;
                        apply_traffic_light_layout(window, hh);
                    }];
        [[NSNotificationCenter defaultCenter]
            addObserverForName:NSWindowDidExitFullScreenNotification
                        object:window
                         queue:[NSOperationQueue mainQueue]
                    usingBlock:^(NSNotification * _Nonnull note) {
                        (void)note;
                        apply_traffic_light_layout(window, hh);
                    }];
    });
}

void openmimo_take_snapshot(void *wk_webview_ptr, const char *out_path, SnapshotCallback cb) {
    if (!wk_webview_ptr || !out_path) {
        if (cb) cb(1);
        return;
    }
    WKWebView *webView = (__bridge WKWebView *)wk_webview_ptr;
    NSString *path = [NSString stringWithUTF8String:out_path];
    NSString *dir = [path stringByDeletingLastPathComponent];
    if (dir && dir.length > 0) {
        [[NSFileManager defaultManager] createDirectoryAtPath:dir
                                  withIntermediateDirectories:YES
                                                   attributes:nil
                                                        error:nil];
    }

    WKSnapshotConfiguration *config = [[WKSnapshotConfiguration alloc] init];
    config.afterScreenUpdates = YES;
    [webView takeSnapshotWithConfiguration:config completionHandler:^(NSImage * _Nullable image, NSError * _Nullable error) {
        if (error || !image) {
            if (cb) cb(2);
            return;
        }
        NSRect rect = NSMakeRect(0, 0, image.size.width, image.size.height);
        CGImageRef cgImg = [image CGImageForProposedRect:&rect context:nil hints:nil];
        if (!cgImg) {
            if (cb) cb(3);
            return;
        }
        NSBitmapImageRep *rep = [[NSBitmapImageRep alloc] initWithCGImage:cgImg];
        NSData *png = [rep representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
        BOOL ok = (png != nil) && [png writeToFile:path atomically:YES];
        if (cb) cb(ok ? 0 : 4);
    }];
}
