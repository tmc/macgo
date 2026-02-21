package launchservices

import (
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

// Framework handles
var (
	libCoreServices uintptr
	libAppServices  uintptr
	libCoreFound    uintptr
)

// Cached selectors
var (
	selAlloc             objc.SEL
	selInit              objc.SEL
	selInitWithCapacity  objc.SEL
	selAddObject         objc.SEL
	selSetObjectForKey   objc.SEL
	selNumberWithBool    objc.SEL
	selNumberWithInteger objc.SEL
	selStringWithUTF8    objc.SEL
	selUTF8String        objc.SEL
	selFileURLWithPath   objc.SEL
	selURLWithString     objc.SEL
	selSharedWorkspace   objc.SEL
	selFullPathForApp    objc.SEL
	selLocalizedDesc     objc.SEL
	selFirstObject       objc.SEL
)

// Cached classes
var (
	clsNSString            objc.Class
	clsNSURL               objc.Class
	clsNSArray             objc.Class
	clsNSMutableArray      objc.Class
	clsNSMutableDictionary objc.Class
	clsNSNumber            objc.Class
	clsNSWorkspace         objc.Class
)

// LaunchServices option key symbols (loaded via dlsym — global NSString* pointers)
var (
	symActivateKey                    objc.ID
	symHideKey                        objc.ID
	symAddToRecentsKey                objc.ID
	symPreferRunningInstanceKey       objc.ID
	symWaitForCheckInKey              objc.ID
	symArgumentsKey                   objc.ID
	symEnvironmentVariablesKey        objc.ID
	symStdInPathKey                   objc.ID
	symStdOutPathKey                  objc.ID
	symStdErrPathKey                  objc.ID
	symArchitectureKey                objc.ID
	symArchitectureSubtypeKey         objc.ID
	symLaunchWithoutRestoringStateKey objc.ID
)

// LaunchServices functions
var (
	fnLSOpenURLsWithCompletionHandler   func(urls, appURL, opts objc.ID, block objc.Block)
	fnLSCopyAppURLsForBundleID          func(bundleID objc.ID, err uintptr) objc.ID
	fnLSCopyDefaultAppURLForURL         func(url objc.ID, role uint32, err uintptr) objc.ID
	fnLSCopyDefaultAppURLForContentType func(contentType objc.ID, role uint32, err uintptr) objc.ID
	fnLSASNExtractHighAndLowParts       func(asn uintptr, high, low *uint32)
	fnGetProcessPID                     func(psn uintptr, pid *int32) int32
)

// CoreFoundation functions
var (
	fnCFRetain           func(cf uintptr) uintptr
	fnCFRelease          func(cf uintptr)
	fnCFErrorGetDomain   func(err uintptr) objc.ID
	fnCFErrorGetCode     func(err uintptr) int64
	fnCFRunLoopRunInMode func(mode uintptr, seconds float64, returnAfterSourceHandled bool) int32
)

var kCFRunLoopDefaultMode uintptr

var initOnce sync.Once

func initObjC() {
	initOnce.Do(func() {
		_, err := purego.Dlopen("/System/Library/Frameworks/Foundation.framework/Foundation", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err != nil {
			panic("failed to load Foundation: " + err.Error())
		}
		_, err = purego.Dlopen("/System/Library/Frameworks/AppKit.framework/AppKit", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err != nil {
			panic("failed to load AppKit: " + err.Error())
		}
		libCoreServices, err = purego.Dlopen("/System/Library/Frameworks/CoreServices.framework/CoreServices", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err != nil {
			panic("failed to load CoreServices: " + err.Error())
		}
		libAppServices, err = purego.Dlopen("/System/Library/Frameworks/ApplicationServices.framework/ApplicationServices", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err != nil {
			panic("failed to load ApplicationServices: " + err.Error())
		}
		libCoreFound, err = purego.Dlopen("/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err != nil {
			panic("failed to load CoreFoundation: " + err.Error())
		}

		// Cache selectors
		selAlloc = objc.RegisterName("alloc")
		selInit = objc.RegisterName("init")
		selInitWithCapacity = objc.RegisterName("initWithCapacity:")
		selAddObject = objc.RegisterName("addObject:")
		selSetObjectForKey = objc.RegisterName("setObject:forKey:")
		selNumberWithBool = objc.RegisterName("numberWithBool:")
		selNumberWithInteger = objc.RegisterName("numberWithInteger:")
		selStringWithUTF8 = objc.RegisterName("stringWithUTF8String:")
		selUTF8String = objc.RegisterName("UTF8String")
		selFileURLWithPath = objc.RegisterName("fileURLWithPath:")
		selURLWithString = objc.RegisterName("URLWithString:")
		selSharedWorkspace = objc.RegisterName("sharedWorkspace")
		selFullPathForApp = objc.RegisterName("fullPathForApplication:")
		selLocalizedDesc = objc.RegisterName("localizedDescription")
		selFirstObject = objc.RegisterName("firstObject")

		// Cache classes
		clsNSString = objc.GetClass("NSString")
		clsNSURL = objc.GetClass("NSURL")
		clsNSArray = objc.GetClass("NSArray")
		clsNSMutableArray = objc.GetClass("NSMutableArray")
		clsNSMutableDictionary = objc.GetClass("NSMutableDictionary")
		clsNSNumber = objc.GetClass("NSNumber")
		clsNSWorkspace = objc.GetClass("NSWorkspace")

		// LaunchServices private SPI
		purego.RegisterLibFunc(&fnLSOpenURLsWithCompletionHandler, libCoreServices, "_LSOpenURLsWithCompletionHandler")
		purego.RegisterLibFunc(&fnLSASNExtractHighAndLowParts, libCoreServices, "_LSASNExtractHighAndLowParts")

		// LaunchServices public API
		purego.RegisterLibFunc(&fnLSCopyAppURLsForBundleID, libCoreServices, "LSCopyApplicationURLsForBundleIdentifier")
		purego.RegisterLibFunc(&fnLSCopyDefaultAppURLForURL, libCoreServices, "LSCopyDefaultApplicationURLForURL")
		purego.RegisterLibFunc(&fnLSCopyDefaultAppURLForContentType, libCoreServices, "LSCopyDefaultApplicationURLForContentType")

		// ApplicationServices
		purego.RegisterLibFunc(&fnGetProcessPID, libAppServices, "GetProcessPID")

		// CoreFoundation
		purego.RegisterLibFunc(&fnCFRetain, libCoreFound, "CFRetain")
		purego.RegisterLibFunc(&fnCFRelease, libCoreFound, "CFRelease")
		purego.RegisterLibFunc(&fnCFErrorGetDomain, libCoreFound, "CFErrorGetDomain")
		purego.RegisterLibFunc(&fnCFErrorGetCode, libCoreFound, "CFErrorGetCode")
		purego.RegisterLibFunc(&fnCFRunLoopRunInMode, libCoreFound, "CFRunLoopRunInMode")

		// kCFRunLoopDefaultMode is a global CFStringRef
		sym, err := purego.Dlsym(libCoreFound, "kCFRunLoopDefaultMode")
		if err == nil {
			kCFRunLoopDefaultMode = derefGlobalPtr(sym)
		}

		// Load LaunchServices option key symbols.
		symActivateKey = loadLSKey("_kLSOpenOptionActivateKey")
		symHideKey = loadLSKey("_kLSOpenOptionHideKey")
		symAddToRecentsKey = loadLSKey("_kLSOpenOptionAddToRecentsKey")
		symPreferRunningInstanceKey = loadLSKey("_kLSOpenOptionPreferRunningInstanceKey")
		symWaitForCheckInKey = loadLSKey("_kLSOpenOptionWaitForApplicationToCheckInKey")
		symArgumentsKey = loadLSKey("_kLSOpenOptionArgumentsKey")
		symEnvironmentVariablesKey = loadLSKey("_kLSOpenOptionEnvironmentVariablesKey")
		symStdInPathKey = loadLSKey("_kLSOpenOptionLaunchStdInPathKey")
		symStdOutPathKey = loadLSKey("_kLSOpenOptionLaunchStdOutPathKey")
		symStdErrPathKey = loadLSKey("_kLSOpenOptionLaunchStdErrPathKey")
		symArchitectureKey = loadLSKey("_kLSOpenOptionArchitectureKey")
		symArchitectureSubtypeKey = loadLSKey("_kLSOpenOptionArchitectureSubtypeKey")
		symLaunchWithoutRestoringStateKey = loadLSKeyOptional("_kLSOpenOptionLaunchWithoutRestoringStateKey")
	})
}

// derefGlobalPtr reads an ObjC object pointer from a global variable address.
//
//go:nocheckptr
func derefGlobalPtr(addr uintptr) uintptr {
	return *(*uintptr)(unsafe.Pointer(addr)) //nolint:govet
}

func loadLSKey(name string) objc.ID {
	sym, err := purego.Dlsym(libCoreServices, name)
	if err != nil {
		panic("dlsym " + name + ": " + err.Error())
	}
	return objc.ID(derefGlobalPtr(sym))
}

func loadLSKeyOptional(name string) objc.ID {
	sym, err := purego.Dlsym(libCoreServices, name)
	if err != nil {
		return 0
	}
	return objc.ID(derefGlobalPtr(sym))
}
