// Package main demonstrates ScreenCaptureKit usage via purego.
//
// This example replicates the functionality of the darwinkit screencapturekit
// example but uses purego for direct Objective-C runtime access instead of
// the darwinkit bindings.
package main

import (
	"flag"
	"fmt"
	"log"
	"runtime"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

var (
	e2e = flag.Bool("e2e", false, "Run end-to-end test mode")
)

// ScreenCaptureKit framework handle
var scFramework uintptr

// Objective-C classes
var (
	classNSAutoreleasePool objc.Class
	classNSApplication     objc.Class
	classNSWindow          objc.Class
	classNSTextField       objc.Class
	classNSButton          objc.Class
	classNSScrollView      objc.Class
	classNSTextView        objc.Class
	classNSFont            objc.Class
	classNSColor           objc.Class
	classNSString          objc.Class
	classSCShareableContent objc.Class
	classSCContentFilter   objc.Class
	classSCStreamConfiguration objc.Class
	classSCStream          objc.Class
)

// Selectors
var (
	selAlloc              objc.SEL
	selInit               objc.SEL
	selRelease            objc.SEL
	selAutorelease        objc.SEL
	selDrain              objc.SEL
	selSharedApplication  objc.SEL
	selRun                objc.SEL
	selInitWithContentRect objc.SEL
	selSetTitle           objc.SEL
	selCenter             objc.SEL
	selMakeKeyAndOrderFront objc.SEL
	selContentView        objc.SEL
	selAddSubview         objc.SEL
	selSetStringValue     objc.SEL
	selSetEditable        objc.SEL
	selSetBezeled         objc.SEL
	selSetDrawsBackground objc.SEL
	selSetFont            objc.SEL
	selSetTextColor       objc.SEL
	selSetBezelStyle      objc.SEL
	selSetEnabled         objc.SEL
	selSetTarget          objc.SEL
	selSetAction          objc.SEL
	selSetHasVerticalScroller objc.SEL
	selSetAutohidesScrollers objc.SEL
	selSetBorderType      objc.SEL
	selSetDocumentView    objc.SEL
	selSetBackgroundColor objc.SEL
	selSetString          objc.SEL
)

// NSRect represents an Objective-C CGRect/NSRect
type NSRect struct {
	Origin NSPoint
	Size   NSSize
}

// NSPoint represents an Objective-C CGPoint/NSPoint
type NSPoint struct {
	X, Y float64
}

// NSSize represents an Objective-C CGSize/NSSize
type NSSize struct {
	Width, Height float64
}

func init() {
	runtime.LockOSThread()

	// Load ScreenCaptureKit framework
	var err error
	scFramework, err = purego.Dlopen("/System/Library/Frameworks/ScreenCaptureKit.framework/ScreenCaptureKit", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		panic(fmt.Sprintf("failed to load ScreenCaptureKit framework: %v", err))
	}

	// Get Objective-C classes
	classNSAutoreleasePool = objc.GetClass("NSAutoreleasePool")
	classNSApplication = objc.GetClass("NSApplication")
	classNSWindow = objc.GetClass("NSWindow")
	classNSTextField = objc.GetClass("NSTextField")
	classNSButton = objc.GetClass("NSButton")
	classNSScrollView = objc.GetClass("NSScrollView")
	classNSTextView = objc.GetClass("NSTextView")
	classNSFont = objc.GetClass("NSFont")
	classNSColor = objc.GetClass("NSColor")
	classNSString = objc.GetClass("NSString")
	classSCShareableContent = objc.GetClass("SCShareableContent")
	classSCContentFilter = objc.GetClass("SCContentFilter")
	classSCStreamConfiguration = objc.GetClass("SCStreamConfiguration")
	classSCStream = objc.GetClass("SCStream")

	// Get selectors
	selAlloc = objc.RegisterName("alloc")
	selInit = objc.RegisterName("init")
	selRelease = objc.RegisterName("release")
	selAutorelease = objc.RegisterName("autorelease")
	selDrain = objc.RegisterName("drain")
	selSharedApplication = objc.RegisterName("sharedApplication")
	selRun = objc.RegisterName("run")
	selInitWithContentRect = objc.RegisterName("initWithContentRect:styleMask:backing:defer:")
	selSetTitle = objc.RegisterName("setTitle:")
	selCenter = objc.RegisterName("center")
	selMakeKeyAndOrderFront = objc.RegisterName("makeKeyAndOrderFront:")
	selContentView = objc.RegisterName("contentView")
	selAddSubview = objc.RegisterName("addSubview:")
	selSetStringValue = objc.RegisterName("setStringValue:")
	selSetEditable = objc.RegisterName("setEditable:")
	selSetBezeled = objc.RegisterName("setBezeled:")
	selSetDrawsBackground = objc.RegisterName("setDrawsBackground:")
	selSetFont = objc.RegisterName("setFont:")
	selSetTextColor = objc.RegisterName("setTextColor:")
	selSetBezelStyle = objc.RegisterName("setBezelStyle:")
	selSetEnabled = objc.RegisterName("setEnabled:")
	selSetTarget = objc.RegisterName("setTarget:")
	selSetAction = objc.RegisterName("setAction:")
	selSetHasVerticalScroller = objc.RegisterName("setHasVerticalScroller:")
	selSetAutohidesScrollers = objc.RegisterName("setAutohidesScrollers:")
	selSetBorderType = objc.RegisterName("setBorderType:")
	selSetDocumentView = objc.RegisterName("setDocumentView:")
	selSetBackgroundColor = objc.RegisterName("setBackgroundColor:")
	selSetString = objc.RegisterName("setString:")
}

// createNSString creates an NSString from a Go string
func createNSString(s string) objc.ID {
	cstr := append([]byte(s), 0)
	selUTF8String := objc.RegisterName("stringWithUTF8String:")
	return objc.ID(classNSString).Send(selUTF8String, uintptr(unsafe.Pointer(&cstr[0])))
}

// createTextFieldWithFrame creates an NSTextField with the given frame
func createTextFieldWithFrame(frame NSRect) objc.ID {
	selInitWithFrame := objc.RegisterName("initWithFrame:")
	textField := objc.ID(classNSTextField).Send(selAlloc)

	// NSRect is passed by value in obj-c messages
	return textField.Send(selInitWithFrame,
		uintptr(unsafe.Pointer(&frame)))
}

// createButtonWithFrame creates an NSButton with the given frame
func createButtonWithFrame(frame NSRect) objc.ID {
	selInitWithFrame := objc.RegisterName("initWithFrame:")
	button := objc.ID(classNSButton).Send(selAlloc)
	return button.Send(selInitWithFrame, uintptr(unsafe.Pointer(&frame)))
}

// createScrollViewWithFrame creates an NSScrollView with the given frame
func createScrollViewWithFrame(frame NSRect) objc.ID {
	selInitWithFrame := objc.RegisterName("initWithFrame:")
	scrollView := objc.ID(classNSScrollView).Send(selAlloc)
	return scrollView.Send(selInitWithFrame, uintptr(unsafe.Pointer(&frame)))
}

// createTextViewWithFrame creates an NSTextView with the given frame
func createTextViewWithFrame(frame NSRect) objc.ID {
	selInitWithFrame := objc.RegisterName("initWithFrame:")
	textView := objc.ID(classNSTextView).Send(selAlloc)
	return textView.Send(selInitWithFrame, uintptr(unsafe.Pointer(&frame)))
}

func main() {
	flag.Parse()

	// Create autorelease pool
	pool := objc.ID(classNSAutoreleasePool).Send(selAlloc)
	pool.Send(selInit)
	

	// Get shared application
	app := objc.ID(classNSApplication).Send(selSharedApplication)

	// Create window
	const (
		NSWindowStyleMaskTitled         = 1 << 0
		NSWindowStyleMaskClosable       = 1 << 1
		NSWindowStyleMaskMiniaturizable = 1 << 2
		NSWindowStyleMaskResizable      = 1 << 3
		NSBackingStoreBuffered          = 2
	)

	windowRect := NSRect{
		Origin: NSPoint{X: 0, Y: 0},
		Size:   NSSize{Width: 900, Height: 700},
	}

	styleMask := NSWindowStyleMaskTitled | NSWindowStyleMaskClosable |
		NSWindowStyleMaskMiniaturizable | NSWindowStyleMaskResizable

	window := objc.ID(classNSWindow).Send(selAlloc)
	window = window.Send(selInitWithContentRect,
		uintptr(unsafe.Pointer(&windowRect)),
		uintptr(styleMask),
		uintptr(NSBackingStoreBuffered),
		uintptr(0)) // defer

	// Set window title
	title := createNSString("ScreenCaptureKit Demo - Purego")
	window.Send(selSetTitle, uintptr(title))

	// Center and show window
	window.Send(selCenter)
	window.Send(selMakeKeyAndOrderFront, 0)

	// Create UI elements
	contentView := window.Send(selContentView)

	// Title label
	titleLabel := createTextFieldWithFrame(NSRect{
		Origin: NSPoint{X: 20, Y: 660},
		Size:   NSSize{Width: 860, Height: 30},
	})
	titleText := createNSString("ScreenCaptureKit - Purego Implementation")
	titleLabel.Send(selSetStringValue, uintptr(titleText))
	titleLabel.Send(selSetEditable, 0)
	titleLabel.Send(selSetBezeled, 0)
	titleLabel.Send(selSetDrawsBackground, 0)

	// Set font (bold, size 18)
	selBoldSystemFontOfSize := objc.RegisterName("boldSystemFontOfSize:")
	boldFont := objc.ID(classNSFont).Send(selBoldSystemFontOfSize, uintptr(18))
	titleLabel.Send(selSetFont, uintptr(boldFont))

	// Set text color
	selLabelColor := objc.RegisterName("labelColor")
	labelColor := objc.ID(classNSColor).Send(selLabelColor)
	titleLabel.Send(selSetTextColor, uintptr(labelColor))

	contentView.Send(selAddSubview, uintptr(titleLabel))

	// Info label
	infoLabel := createTextFieldWithFrame(NSRect{
		Origin: NSPoint{X: 20, Y: 630},
		Size:   NSSize{Width: 860, Height: 20},
	})
	infoText := createNSString("Demonstrates screen capture using pure Go and purego")
	infoLabel.Send(selSetStringValue, uintptr(infoText))
	infoLabel.Send(selSetEditable, 0)
	infoLabel.Send(selSetBezeled, 0)
	infoLabel.Send(selSetDrawsBackground, 0)
	contentView.Send(selAddSubview, uintptr(infoLabel))

	// Status label
	statusLabel := createTextFieldWithFrame(NSRect{
		Origin: NSPoint{X: 20, Y: 600},
		Size:   NSSize{Width: 860, Height: 20},
	})
	statusText := createNSString("Ready - Click 'Get Shareable Content' to start")
	statusLabel.Send(selSetStringValue, uintptr(statusText))
	statusLabel.Send(selSetEditable, 0)
	statusLabel.Send(selSetBezeled, 0)
	statusLabel.Send(selSetDrawsBackground, 0)
	contentView.Send(selAddSubview, uintptr(statusLabel))

	// Get Shareable Content button
	getContentBtn := createButtonWithFrame(NSRect{
		Origin: NSPoint{X: 20, Y: 560},
		Size:   NSSize{Width: 180, Height: 32},
	})
	btnTitle := createNSString("Get Shareable Content")
	getContentBtn.Send(selSetTitle, uintptr(btnTitle))
	const NSBezelStyleRounded = 1
	getContentBtn.Send(selSetBezelStyle, NSBezelStyleRounded)
	contentView.Send(selAddSubview, uintptr(getContentBtn))

	// Start Capture button
	startCaptureBtn := createButtonWithFrame(NSRect{
		Origin: NSPoint{X: 210, Y: 560},
		Size:   NSSize{Width: 150, Height: 32},
	})
	startTitle := createNSString("Start Capture")
	startCaptureBtn.Send(selSetTitle, uintptr(startTitle))
	startCaptureBtn.Send(selSetBezelStyle, NSBezelStyleRounded)
	startCaptureBtn.Send(selSetEnabled, 0)
	contentView.Send(selAddSubview, uintptr(startCaptureBtn))

	// Stop Capture button
	stopCaptureBtn := createButtonWithFrame(NSRect{
		Origin: NSPoint{X: 370, Y: 560},
		Size:   NSSize{Width: 150, Height: 32},
	})
	stopTitle := createNSString("Stop Capture")
	stopCaptureBtn.Send(selSetTitle, uintptr(stopTitle))
	stopCaptureBtn.Send(selSetBezelStyle, NSBezelStyleRounded)
	stopCaptureBtn.Send(selSetEnabled, 0)
	contentView.Send(selAddSubview, uintptr(stopCaptureBtn))

	// Results scroll view
	scrollView := createScrollViewWithFrame(NSRect{
		Origin: NSPoint{X: 20, Y: 20},
		Size:   NSSize{Width: 860, Height: 530},
	})
	scrollView.Send(selSetHasVerticalScroller, 1)
	scrollView.Send(selSetAutohidesScrollers, 1)
	scrollView.Send(selSetBorderType, 1) // NSBezelBorder
	contentView.Send(selAddSubview, uintptr(scrollView))

	// Results text view
	resultsTextView := createTextViewWithFrame(NSRect{
		Origin: NSPoint{X: 0, Y: 0},
		Size:   NSSize{Width: 860, Height: 530},
	})
	resultsTextView.Send(selSetEditable, 0)

	// Set monospaced font
	selMonospacedSystemFontOfSize := objc.RegisterName("monospacedSystemFontOfSize:weight:")
	monoFont := objc.ID(classNSFont).Send(selMonospacedSystemFontOfSize, uintptr(11), uintptr(0))
	resultsTextView.Send(selSetFont, uintptr(monoFont))

	resultsTextView.Send(selSetTextColor, uintptr(labelColor))

	selTextBackgroundColor := objc.RegisterName("textBackgroundColor")
	bgColor := objc.ID(classNSColor).Send(selTextBackgroundColor)
	resultsTextView.Send(selSetBackgroundColor, uintptr(bgColor))

	scrollView.Send(selSetDocumentView, uintptr(resultsTextView))

	// Set initial message
	initialMsg := createNSString(`ScreenCaptureKit Demo - Purego Implementation

This example demonstrates the ScreenCaptureKit framework using purego for direct Objective-C runtime access.

Steps:
1. Click 'Get Shareable Content' to retrieve available displays and windows
2. Click 'Start Capture' to begin capturing the main display
3. Click 'Stop Capture' to end the capture

Note: This is a simplified demonstration showing the basic structure.
Full capture functionality requires implementing stream delegates and completion handlers.`)

	resultsTextView.Send(selSetString, uintptr(initialMsg))

	// Set activation policy and activate
	const NSApplicationActivationPolicyRegular = 0
	selSetActivationPolicy := objc.RegisterName("setActivationPolicy:")
	app.Send(selSetActivationPolicy, NSApplicationActivationPolicyRegular)

	selActivateIgnoringOtherApps := objc.RegisterName("activateIgnoringOtherApps:")
	app.Send(selActivateIgnoringOtherApps, 1)

	log.Println("ScreenCaptureKit Demo - Purego Implementation")
	log.Println("Window created and shown")
	log.Println("Note: Full capture implementation requires completion handler support")

	// Run the application
	app.Send(selRun)
}
