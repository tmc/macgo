// mouse-click: Perform a mouse click at absolute screen coordinates using CGEvent APIs
// By default, smoothly moves to the target, clicks, and returns to original position
package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework CoreGraphics -framework ApplicationServices -framework Cocoa -framework QuartzCore
#include <CoreGraphics/CoreGraphics.h>
#include <ApplicationServices/ApplicationServices.h>
#include <unistd.h>
#import <Cocoa/Cocoa.h>
#import <QuartzCore/QuartzCore.h>

void moveMouse(int x, int y) {
    CGWarpMouseCursorPosition(CGPointMake(x, y));
}

void getCurrentPosition(int *x, int *y) {
    CGEventRef event = CGEventCreate(NULL);
    CGPoint cursor = CGEventGetLocation(event);
    CFRelease(event);
    *x = (int)cursor.x;
    *y = (int)cursor.y;
}

void clickMouse(int x, int y) {
    // Small delay to ensure cursor has moved
    usleep(100000); // 0.1 seconds

    // Create mouse down event
    CGEventRef mouseDown = CGEventCreateMouseEvent(
        NULL,
        kCGEventLeftMouseDown,
        CGPointMake(x, y),
        kCGMouseButtonLeft
    );

    // Create mouse up event
    CGEventRef mouseUp = CGEventCreateMouseEvent(
        NULL,
        kCGEventLeftMouseUp,
        CGPointMake(x, y),
        kCGMouseButtonLeft
    );

    // Post events
    CGEventPost(kCGHIDEventTap, mouseDown);
    CGEventPost(kCGHIDEventTap, mouseUp);

    // Clean up
    CFRelease(mouseDown);
    CFRelease(mouseUp);
}

// Global array to store visual indicator windows
static NSMutableArray *visualWindows = nil;

void initVisualIndicators() {
    if (!visualWindows) {
        visualWindows = [[NSMutableArray alloc] init];
    }
}

void drawVisualIndicator(int x, int y) {
    // Visual indicators disabled due to NSWindow/GCD threading issues from CGo
    // Use QuickTime Player's "Show Mouse Clicks in Recording" feature instead
    // See: QuickTime Player > File > New Screen Recording > Options > Show Mouse Clicks
}

void cleanupVisualIndicators() {
    // Visual indicators disabled - nothing to clean up
}
*/
import "C"
import (
	"flag"
	"fmt"
	"math"
	"os"
	"strconv"
	"time"
)

func getCurrentPosition() (int, int) {
	var x, y C.int
	C.getCurrentPosition(&x, &y)
	return int(x), int(y)
}

func moveMouse(x, y int) {
	C.moveMouse(C.int(x), C.int(y))
}

func clickMouse(x, y int) {
	C.clickMouse(C.int(x), C.int(y))
}

// smoothMove performs natural human-like movement from current position to target
func smoothMove(targetX, targetY int, showVisual bool) error {
	if showVisual {
		C.initVisualIndicators()
		defer C.cleanupVisualIndicators()
	}

	startX, startY := getCurrentPosition()

	if startX == targetX && startY == targetY {
		return nil
	}

	deltaX := float64(targetX - startX)
	deltaY := float64(targetY - startY)
	distance := math.Sqrt(deltaX*deltaX + deltaY*deltaY)

	const (
		minSteps   = 20
		maxSteps   = 80
		minDelay   = 8 * time.Millisecond
		maxDelay   = 25 * time.Millisecond
		noiseScale = 2.0
		overshoot  = 0.98
	)

	steps := int(distance / 4)
	if steps < minSteps {
		steps = minSteps
	} else if steps > maxSteps {
		steps = maxSteps
	}

	midX := (float64(startX) + float64(targetX)) / 2
	midY := (float64(startY) + float64(targetY)) / 2

	curveAmount := math.Min(distance*0.1, 30.0)

	perpX := -deltaY / distance
	perpY := deltaX / distance

	curveX := midX + perpX*curveAmount*0.3
	curveY := midY + perpY*curveAmount*0.3

	for i := 0; i <= steps; i++ {
		progress := float64(i) / float64(steps)

		var easedProgress float64
		if progress < 0.5 {
			easedProgress = 2 * progress * progress
		} else {
			easedProgress = 1 - 2*(1-progress)*(1-progress)
		}

		t := easedProgress

		currentX := (1-t)*(1-t)*float64(startX) + 2*(1-t)*t*curveX + t*t*float64(targetX)
		currentY := (1-t)*(1-t)*float64(startY) + 2*(1-t)*t*curveY + t*t*float64(targetY)

		noiseX := (math.Sin(progress*20) + math.Sin(progress*35)) * noiseScale * (1 - progress)
		noiseY := (math.Cos(progress*25) + math.Cos(progress*40)) * noiseScale * (1 - progress)

		currentX += noiseX
		currentY += noiseY

		if progress > overshoot {
			overshootFactor := (progress - overshoot) / (1.0 - overshoot)
			correctionX := float64(targetX) - currentX
			correctionY := float64(targetY) - currentY
			currentX += correctionX * overshootFactor * 0.5
			currentY += correctionY * overshootFactor * 0.5
		}

		x := int(math.Round(currentX))
		y := int(math.Round(currentY))
		moveMouse(x, y)

		// Draw visual indicator if enabled
		if showVisual {
			C.drawVisualIndicator(C.int(x), C.int(y))
		}

		var delay time.Duration
		if progress < 0.2 {
			delay = maxDelay - time.Duration(float64(maxDelay-minDelay)*progress*5)
		} else if progress > 0.8 {
			delay = minDelay + time.Duration(float64(maxDelay-minDelay)*(progress-0.8)*5)
		} else {
			delay = minDelay
		}

		if i < steps {
			time.Sleep(delay)
		}
	}

	moveMouse(targetX, targetY)

	return nil
}

func main() {
	var instant bool
	var noReturn bool
	var showVisual bool
	flag.BoolVar(&instant, "instant", false, "instant move (no smooth animation)")
	flag.BoolVar(&instant, "i", false, "instant move (shorthand)")
	flag.BoolVar(&noReturn, "no-return", false, "don't return to original position after click")
	flag.BoolVar(&noReturn, "n", false, "don't return (shorthand)")
	flag.BoolVar(&showVisual, "visual", false, "show visual trail of movement path")
	flag.BoolVar(&showVisual, "v", false, "show visual trail (shorthand)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <x> <y>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Click the mouse at absolute screen coordinates (x, y)\n")
		fmt.Fprintf(os.Stderr, "\nBy default: smoothly moves to target, clicks, and returns to original position\n")
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s 800 400              # Smooth click at (800,400) and return\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -visual 800 400      # Smooth click with blue trail\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -v -n 800 400        # Visual trail, stay at target\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -no-return 800 400   # Smooth click, stay at target\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -instant 800 400     # Instant click, return\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -i -n 800 400        # Instant click, stay at target\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nNote: -visual flag is accepted but currently disabled due to threading issues.\n")
		fmt.Fprintf(os.Stderr, "      Use QuickTime Player's 'Show Mouse Clicks in Recording' instead:\n")
		fmt.Fprintf(os.Stderr, "      QuickTime Player > File > New Screen Recording > Options > Show Mouse Clicks\n")
	}

	flag.Parse()

	if flag.NArg() != 2 {
		flag.Usage()
		os.Exit(1)
	}

	x, err := strconv.Atoi(flag.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid x coordinate: %v\n", err)
		os.Exit(1)
	}

	y, err := strconv.Atoi(flag.Arg(1))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid y coordinate: %v\n", err)
		os.Exit(1)
	}

	// Warn if visual requested with instant mode
	if instant && showVisual {
		fmt.Fprintf(os.Stderr, "Note: -visual only works with smooth movement (ignored with -instant)\n")
		showVisual = false
	}

	// Save original position
	origX, origY := getCurrentPosition()

	// Move to target
	if instant {
		moveMouse(x, y)
		time.Sleep(100 * time.Millisecond)
	} else {
		if err := smoothMove(x, y, showVisual); err != nil {
			fmt.Fprintf(os.Stderr, "Error moving to target: %v\n", err)
			os.Exit(1)
		}
	}

	// Perform click
	clickMouse(x, y)

	// Return to original position if requested
	if !noReturn {
		time.Sleep(100 * time.Millisecond)
		if instant {
			moveMouse(origX, origY)
		} else {
			if err := smoothMove(origX, origY, showVisual); err != nil {
				fmt.Fprintf(os.Stderr, "Error returning to origin: %v\n", err)
				os.Exit(1)
			}
		}
	}
}
