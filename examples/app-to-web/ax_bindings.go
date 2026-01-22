package main

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/tmc/appledocs/generated/corefoundation"
	"github.com/tmc/appledocs/generated/coregraphics"
)

// AXUIElementRef is a reference to an accessibility object.
type AXUIElementRef uintptr

// AXError represents an error code from Accessibility API.
type AXError int32

const (
	kAXErrorSuccess = 0
)

// Attribute constants (we'll create CFStrings for these)
const (
	kAXRoleAttribute     = "AXRole"
	kAXSubroleAttribute  = "AXSubrole"
	kAXTitleAttribute    = "AXTitle"
	kAXPositionAttribute = "AXPosition"
	kAXSizeAttribute     = "AXSize"
	kAXChildrenAttribute = "AXChildren"
)

// AXValueType
type AXValueType uint32

const (
	kAXValueTypeCGPoint = 1
	kAXValueTypeCGSize  = 2
	kAXValueTypeCGRect  = 3 // Not always standard, usually pos/size are separate
)

const kCFStringEncodingUTF8 = 0x08000100

var (
	axUIElementCreateApplication  func(pid int32) AXUIElementRef
	axUIElementCopyAttributeNames func(element AXUIElementRef, names *uintptr) AXError
	axUIElementCopyAttributeValue func(element AXUIElementRef, attribute uintptr, value *uintptr) AXError
	axUIElementGetPid             func(element AXUIElementRef, pid *int32) AXError
	axValueGetValue               func(value uintptr, type_ AXValueType, valuePtr unsafe.Pointer) bool

	// CoreFoundation helpers
	cfRelease func(cf uintptr)
)

func initAX() {
	lib, err := purego.Dlopen("/System/Library/Frameworks/ApplicationServices.framework/Versions/A/ApplicationServices", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		panic(fmt.Errorf("failed to load ApplicationServices: %w", err))
	}

	purego.RegisterLibFunc(&axUIElementCreateApplication, lib, "AXUIElementCreateApplication")
	purego.RegisterLibFunc(&axUIElementCopyAttributeNames, lib, "AXUIElementCopyAttributeNames")
	purego.RegisterLibFunc(&axUIElementCopyAttributeValue, lib, "AXUIElementCopyAttributeValue")
	purego.RegisterLibFunc(&axUIElementGetPid, lib, "AXUIElementGetPid")
	purego.RegisterLibFunc(&axValueGetValue, lib, "AXValueGetValue")

	// Load CFRelease manually to ensure we have it for AX objects
	cfLib, err := purego.Dlopen("/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		panic(fmt.Errorf("failed to load CoreFoundation: %w", err))
	}
	purego.RegisterLibFunc(&cfRelease, cfLib, "CFRelease")
}

// AXNode represents a node in the accessibility tree for JSON output
type AXNode struct {
	ID       string    `json:"id,omitempty"` // Not real AX ID, but maybe path or hash
	Role     string    `json:"role"`
	Subrole  string    `json:"subrole,omitempty"`
	Title    string    `json:"title,omitempty"`
	Frame    Rect      `json:"frame"`
	Children []*AXNode `json:"children,omitempty"`
}

type Rect struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// Helper to create CFString
func createCFString(s string) uintptr {
	ptr, _ := syscall.BytePtrFromString(s)
	// kCFStringEncodingUTF8 = 0x08000100
	return uintptr(corefoundation.CFStringCreateWithCString(0, unsafe.Pointer(ptr), 0x08000100))
}

// Helper to get string from CFTypeRef (which might be CFString)
func getCFStringValue(ref uintptr) string {
	// corefoundation.CFStringGetLength(ref) // generated expects StringRef which is basic type
	// purego bindings for CFStringGetLength in generated code use StringRef as struct/alias
	// We might need to cast or use manual binding if generated types are tricky with raw uintptr

	// Let's assume we can cast uintptr to corefoundation.StringRef
	// generated: type StringRef struct { ID uintptr } ?? check generated
	// Checking generated code from my memory/previous steps:
	// StringRef seems to be alias or struct wrapping ID.
	// Let's assume we can use generated helpers if they take ID or similar.
	// Actually, easier to use manual CFString logic if needed, but let's try to reuse generated.

	// Assuming corefoundation.StringRef(ref) works if it's an alias.
	// If it's a struct with ID:
	/*
		type StringRef struct {
			objectivec.Object
		}
	*/

	// We'll trust that we can extract string via existing helpers or basic CString extraction?
	// corefoundation.CFStringGetCString is standard.
	// Let's rely on simple conversion if possible.

	// For now, minimal implementation:
	// Use CFStringGetCString to buffer
	return "" // TODO: Implement
}

func getAXTree(pid int32) (*AXNode, error) {
	appElem := axUIElementCreateApplication(pid)
	if appElem == 0 {
		return nil, fmt.Errorf("failed to create AX application")
	}
	

	return buildAXTree(appElem, 0)
}

func buildAXTree(element AXUIElementRef, depth int) (*AXNode, error) {
	if depth > 10 { // Limit depth
		return nil, nil
	}

	node := &AXNode{}

	// Get Role
	roleCF := getAttribute(element, kAXRoleAttribute)
	if roleCF != 0 {
		node.Role = cfStringToString(roleCF)
		cfRelease(roleCF)
	}

	// Get Subrole
	subroleCF := getAttribute(element, kAXSubroleAttribute)
	if subroleCF != 0 {
		node.Subrole = cfStringToString(subroleCF)
		cfRelease(subroleCF)
	}

	// Get Title
	titleCF := getAttribute(element, kAXTitleAttribute)
	if titleCF != 0 {
		node.Title = cfStringToString(titleCF)
		cfRelease(titleCF)
	}

	// Get Position
	posVal := getAttribute(element, kAXPositionAttribute)
	if posVal != 0 {
		var pt coregraphics.CGPoint
		if axValueGetValue(posVal, kAXValueTypeCGPoint, unsafe.Pointer(&pt)) {
			node.Frame.X = float64(pt.X)
			node.Frame.Y = float64(pt.Y)
		}
		cfRelease(posVal)
	}

	// Get Size
	sizeVal := getAttribute(element, kAXSizeAttribute)
	if sizeVal != 0 {
		var sz coregraphics.CGSize
		if axValueGetValue(sizeVal, kAXValueTypeCGSize, unsafe.Pointer(&sz)) {
			node.Frame.Width = float64(sz.Width)
			node.Frame.Height = float64(sz.Height)
		}
		cfRelease(sizeVal)
	}

	// Get Children
	childrenVal := getAttribute(element, kAXChildrenAttribute)
	if childrenVal != 0 {
		// childrenVal is CFArrayRef
		// We need to iterate it
		count := corefoundation.CFArrayGetCount(corefoundation.ArrayRef(childrenVal))
		for i := 0; i < int(count); i++ {
			// CFArrayGetValueAtIndex returns void* (uintptr)
			childPtr := corefoundation.CFArrayGetValueAtIndex(corefoundation.ArrayRef(childrenVal), corefoundation.Index(i))
			// Correctly casting the return value to AXUIElementRef
			// Wait, CFArrayGetValueAtIndex returns unsafe.Pointer or uintptr in generated?
			// Checking generated/corefoundation/functions.gen.go: func CFArrayGetValueAtIndex(theArray ArrayRef, idx Index) unsafe.Pointer
			// So childPtr is unsafe.Pointer

			childElem := AXUIElementRef(uintptr(childPtr)) // Cast safe pointer to uintptr

			childNode, _ := buildAXTree(childElem, depth+1)
			if childNode != nil {
				node.Children = append(node.Children, childNode)
			}
		}
		cfRelease(childrenVal)
	}

	return node, nil
}

func getAttribute(element AXUIElementRef, attrName string) uintptr {
	cfAttr := createCFString(attrName)
	

	var result uintptr
	err := axUIElementCopyAttributeValue(element, cfAttr, &result)
	if err == kAXErrorSuccess {
		return result
	}
	return 0
}

func cfStringToString(cfStr uintptr) string {
	if cfStr == 0 {
		return ""
	}

	strRef := corefoundation.StringRef(cfStr)
	length := corefoundation.CFStringGetLength(strRef)
	if length == 0 {
		return ""
	}

	// Try fast path with GetCStringPtr
	ptr := corefoundation.CFStringGetCStringPtr(strRef, corefoundation.StringEncoding(kCFStringEncodingUTF8))
	if ptr != nil {
		return unsafe.String((*byte)(ptr), length)
	}

	// Slow path: buffer copy
	// Max length in bytes could be length * 4 (for UTF8) + 1 null
	bufSize := int(length*4 + 1)
	buf := make([]byte, bufSize)

	// CFStringGetCString returns Boolean (true on success).
	// The generated wrapper returns unsafe.Pointer, which is likely casting the result.
	// We need to check if it's non-nil/non-zero?
	// Or maybe generated bindings return Go bool?
	// Checked grep: "func CFStringGetCString(...) unsafe.Pointer".
	// This implies return type mismatch in generation if C returns Boolean.
	// Let's assume non-zero means true.
	success := corefoundation.CFStringGetCString(strRef, unsafe.Pointer(&buf[0]), corefoundation.Index(bufSize), corefoundation.StringEncoding(kCFStringEncodingUTF8))

	// Check if success is not nil (if pointer) or not 0.
	if success != nil {
		// Find null terminator
		for i, b := range buf {
			if b == 0 {
				return string(buf[:i])
			}
		}
		return string(buf)
	}

	return ""
}
