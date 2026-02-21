package macgo

import (
	"fmt"
	"sync"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

var initAppKitOnce sync.Once

func initAppKit() {
	initAppKitOnce.Do(func() {
		_, err := purego.Dlopen("/System/Library/Frameworks/Foundation.framework/Foundation", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err != nil {
			panic("failed to load Foundation: " + err.Error())
		}
		_, err = purego.Dlopen("/System/Library/Frameworks/AppKit.framework/AppKit", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err != nil {
			panic("failed to load AppKit: " + err.Error())
		}
	})
}

// SetUIMode switches the application's activation policy at runtime.
//
// This changes how the app appears in the macOS UI without requiring a relaunch:
//   - UIModeBackground: no UI, no Dock icon (NSApplicationActivationPolicyProhibited)
//   - UIModeAccessory: can show windows but no Dock icon (NSApplicationActivationPolicyAccessory)
//   - UIModeRegular: full app with Dock icon and menu bar (NSApplicationActivationPolicyRegular)
//
// Switching to UIModeRegular uses a three-step workaround (deactivate, set policy,
// activate) to ensure the menu bar appears correctly.
func SetUIMode(mode UIMode) error {
	initAppKit()

	// Map UIMode to NSApplicationActivationPolicy values.
	// NSApplicationActivationPolicyRegular    = 0
	// NSApplicationActivationPolicyAccessory  = 1
	// NSApplicationActivationPolicyProhibited = 2
	var policy int
	switch mode {
	case UIModeRegular:
		policy = 0
	case UIModeAccessory:
		policy = 1
	case UIModeBackground:
		policy = 2
	default:
		return fmt.Errorf("unknown UIMode %q", mode)
	}

	cls := objc.GetClass("NSApplication")
	if cls == 0 {
		return fmt.Errorf("NSApplication class not found")
	}

	app := objc.ID(cls).Send(objc.RegisterName("sharedApplication"))
	if app == 0 {
		return fmt.Errorf("failed to get NSApplication sharedApplication")
	}

	selSetPolicy := objc.RegisterName("setActivationPolicy:")

	if mode == UIModeRegular {
		// Three-step workaround: deactivate, set policy, activate.
		// Without this, switching to Regular doesn't show the menu bar.
		app.Send(objc.RegisterName("deactivate"))
		ok := objc.Send[bool](app, selSetPolicy, policy)
		if !ok {
			return fmt.Errorf("setActivationPolicy failed for policy %d", policy)
		}
		app.Send(objc.RegisterName("activateIgnoringOtherApps:"), true)
		return nil
	}

	ok := objc.Send[bool](app, selSetPolicy, policy)
	if !ok {
		return fmt.Errorf("setActivationPolicy failed for policy %d", policy)
	}
	return nil
}
