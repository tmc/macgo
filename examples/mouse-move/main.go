// mouse-move: Move mouse cursor with natural human-like movement using CGEvent APIs
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

// smoothMove performs natural human-like movement from current position to target
// Based on desktop-automation's sophisticated movement algorithm
func smoothMove(targetX, targetY int, showVisual bool) error {
	if showVisual {
		C.initVisualIndicators()

	}
	// Get current mouse position
	startX, startY := getCurrentPosition()

	// If already at target position, no need to move
	if startX == targetX && startY == targetY {
		return nil
	}

	// Calculate distance and movement parameters
	deltaX := float64(targetX - startX)
	deltaY := float64(targetY - startY)
	distance := math.Sqrt(deltaX*deltaX + deltaY*deltaY)

	// Configure natural human-like movement parameters
	const (
		minSteps   = 20                    // Minimum steps for very short movements
		maxSteps   = 80                    // Maximum steps for very long movements
		minDelay   = 8 * time.Millisecond  // Fastest movement delay
		maxDelay   = 25 * time.Millisecond // Slowest movement delay (start/end)
		noiseScale = 2.0                   // Scale for natural movement noise
		overshoot  = 0.98                  // Slight overshoot factor for natural feel
	)

	// Calculate number of steps based on distance (longer distances = more steps)
	steps := int(distance / 4) // Roughly 4 pixels per step
	if steps < minSteps {
		steps = minSteps
	} else if steps > maxSteps {
		steps = maxSteps
	}

	// Create a natural curved path with slight arc
	// Humans rarely move in perfectly straight lines
	midX := (float64(startX) + float64(targetX)) / 2
	midY := (float64(startY) + float64(targetY)) / 2

	// Add a natural curve perpendicular to the movement direction
	// The curve magnitude depends on distance
	curveAmount := math.Min(distance*0.1, 30.0) // Max curve of 30 pixels

	// Calculate perpendicular direction for natural arc
	perpX := -deltaY / distance
	perpY := deltaX / distance

	// Add curve to midpoint
	curveX := midX + perpX*curveAmount*0.3 // 30% of curve amount
	curveY := midY + perpY*curveAmount*0.3

	for i := 0; i <= steps; i++ {
		// Progress from 0.0 to 1.0
		progress := float64(i) / float64(steps)

		// Natural ease-in-out curve (starts slow, accelerates, then decelerates)
		// This mimics how humans naturally move: hesitant start, confident middle, careful end
		var easedProgress float64
		if progress < 0.5 {
			// Ease in (accelerating)
			easedProgress = 2 * progress * progress
		} else {
			// Ease out (decelerating)
			easedProgress = 1 - 2*(1-progress)*(1-progress)
		}

		// Create natural quadratic Bezier curve path using start, curve point, and end
		t := easedProgress

		// Quadratic Bezier: P(t) = (1-t)²P₀ + 2(1-t)tP₁ + t²P₂
		currentX := (1-t)*(1-t)*float64(startX) + 2*(1-t)*t*curveX + t*t*float64(targetX)
		currentY := (1-t)*(1-t)*float64(startY) + 2*(1-t)*t*curveY + t*t*float64(targetY)

		// Add subtle natural noise to simulate hand tremor/imperfection
		// This makes the movement feel more human
		noiseX := (math.Sin(progress*20) + math.Sin(progress*35)) * noiseScale * (1 - progress)
		noiseY := (math.Cos(progress*25) + math.Cos(progress*40)) * noiseScale * (1 - progress)

		currentX += noiseX
		currentY += noiseY

		// Apply slight overshoot correction near the end for natural feel
		if progress > overshoot {
			overshootFactor := (progress - overshoot) / (1.0 - overshoot)
			correctionX := float64(targetX) - currentX
			correctionY := float64(targetY) - currentY
			currentX += correctionX * overshootFactor * 0.5
			currentY += correctionY * overshootFactor * 0.5
		}

		// Move to calculated position
		x := int(math.Round(currentX))
		y := int(math.Round(currentY))
		moveMouse(x, y)

		// Draw visual indicator if enabled
		if showVisual {
			C.drawVisualIndicator(C.int(x), C.int(y))
		}

		// Dynamic timing: slow at start/end, faster in middle
		var delay time.Duration
		if progress < 0.2 {
			// Start slow
			delay = maxDelay - time.Duration(float64(maxDelay-minDelay)*progress*5)
		} else if progress > 0.8 {
			// End slow
			delay = minDelay + time.Duration(float64(maxDelay-minDelay)*(progress-0.8)*5)
		} else {
			// Middle fast
			delay = minDelay
		}

		// Don't delay on the last step
		if i < steps {
			time.Sleep(delay)
		}
	}

	// Ensure we end up exactly at the target position
	moveMouse(targetX, targetY)

	return nil
}

func main() {
	var smooth bool
	var showVisual bool
	flag.BoolVar(&smooth, "smooth", false, "use smooth human-like movement (default: instant)")
	flag.BoolVar(&smooth, "s", false, "use smooth movement (shorthand)")
	flag.BoolVar(&showVisual, "visual", false, "show visual trail of movement path")
	flag.BoolVar(&showVisual, "v", false, "show visual trail (shorthand)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <x> <y>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Move the mouse cursor to absolute screen coordinates (x, y)\n")
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s 800 400                  # Instant move to (800,400)\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -smooth 800 400          # Smooth human-like move\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -s -v 500 300            # Smooth move with visual trail\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -smooth -visual 800 400  # Full smooth + visual\n", os.Args[0])
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

	if smooth {
		if err := smoothMove(x, y, showVisual); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Instant move (no visual for instant moves)
		moveMouse(x, y)
		if showVisual {
			fmt.Fprintf(os.Stderr, "Note: -visual only works with -smooth mode\n")
		}
	}
}
