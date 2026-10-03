//go:build darwin && cgo

#import <Cocoa/Cocoa.h>
#import <ApplicationServices/ApplicationServices.h>
#import <Carbon/Carbon.h>

int rdMacCaptureAllowed(void) { return CGPreflightScreenCaptureAccess(); }
int rdMacInputAllowed(void) {
    if (!AXIsProcessTrusted()) return 0;
    return IsSecureEventInputEnabled() ? -1 : 1;
}

int rdMacKey(unsigned short key, int down, unsigned long long flags) {
    CGEventRef event = CGEventCreateKeyboardEvent(NULL, key, down != 0);
    if (!event) return 0;
    CGEventSetFlags(event, (CGEventFlags)flags);
    CGEventPost(kCGHIDEventTap, event);
    CFRelease(event);
    return 1;
}

int rdMacMouse(int action, int button, double x, double y, int dx, int dy, unsigned long long flags, int clicks) {
    CGMouseButton nativeButton = button == 2 ? kCGMouseButtonRight : button == 1 ? kCGMouseButtonCenter : kCGMouseButtonLeft;
    CGEventType type = kCGEventMouseMoved;
    if (action == 1) type = button == 0 ? kCGEventLeftMouseDown : button == 2 ? kCGEventRightMouseDown : kCGEventOtherMouseDown;
    else if (action == 2) type = button == 0 ? kCGEventLeftMouseUp : button == 2 ? kCGEventRightMouseUp : kCGEventOtherMouseUp;
    else if (action == 0 && button >= 0) type = button == 0 ? kCGEventLeftMouseDragged : button == 2 ? kCGEventRightMouseDragged : kCGEventOtherMouseDragged;
    CGEventRef event = CGEventCreateMouseEvent(NULL, type, CGPointMake(x, y), nativeButton);
    if (!event) return 0;
    CGEventSetFlags(event, (CGEventFlags)flags);
    if (action == 1 || action == 2) CGEventSetIntegerValueField(event, kCGMouseEventClickState, clicks);
    CGEventPost(kCGHIDEventTap, event);
    CFRelease(event);
    if (action == 3) {
        event = CGEventCreateScrollWheelEvent(NULL, kCGScrollEventUnitPixel, 2, -dy, -dx);
        if (!event) return 0;
        CGEventSetFlags(event, (CGEventFlags)flags);
        CGEventPost(kCGHIDEventTap, event);
        CFRelease(event);
    }
    return 1;
}

int rdMacClipboardSet(const char *text) {
    @autoreleasepool {
        NSString *value = [NSString stringWithUTF8String:text];
        if (!value) return 0;
        NSPasteboard *board = [NSPasteboard generalPasteboard];
        [board clearContents];
        return [board setString:value forType:NSPasteboardTypeString];
    }
}

char *rdMacClipboardGet(int *length) {
    @autoreleasepool {
        NSString *text = [[NSPasteboard generalPasteboard] stringForType:NSPasteboardTypeString];
        if (!text || [text length] > 16384) return NULL;
        NSData *data = [text dataUsingEncoding:NSUTF8StringEncoding];
        if (!data || [data length] > 16384) return NULL;
        *length = (int)[data length];
        char *result = malloc((size_t)*length + 1);
        if (!result) return NULL;
        memcpy(result, [data bytes], (size_t)*length);
        result[*length] = 0;
        return result;
    }
}
