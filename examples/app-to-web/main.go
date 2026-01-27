package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/tmc/appledocs/generated/appkit"
	"github.com/tmc/appledocs/generated/corefoundation"
	"github.com/tmc/appledocs/generated/coregraphics"
	"github.com/tmc/appledocs/generated/dispatch"
	"github.com/tmc/appledocs/generated/imageio"
	"github.com/tmc/appledocs/generated/objc"
	"github.com/tmc/appledocs/generated/screencapturekit"
	"github.com/tmc/macgo"
)

// Define missing constants
const (
	kCGEventLeftMouseDown = 1
	kCGEventLeftMouseUp   = 2
	kCGHIDEventTap        = 0
	kCGMouseButtonLeft    = 0
	// kCFStringEncodingUTF8 = 0x08000100
)

var (
	// Manual binding for CFDataGetBytePtr because generated one returns interface{}
	cfDataGetBytePtr func(uintptr) uintptr
)

func init() {
	lib, err := purego.Dlopen("/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		panic(fmt.Errorf("failed to load CoreFoundation: %w", err))
	}
	purego.RegisterLibFunc(&cfDataGetBytePtr, lib, "CFDataGetBytePtr")
}

type Window struct {
	SCWindow screencapturekit.Window
	ID       uint32
	PID      int32
	AppName  string
	Title    string
	Frame    coregraphics.CGRect
}

var (
	targetWindow Window
	mu           sync.Mutex
	verbose      = flag.Bool("v", false, "Verbose logging")
	veryVerbose  = flag.Bool("vv", false, "Very verbose logging")
)

func main() {
	runtime.LockOSThread()
	// Initialize macgo for TCC identity
	cfg := macgo.NewConfig().WithAppName("AppToWeb")
	cfg.BundleID = "com.tmc.macgo.examples.apptoweb"

	if err := macgo.Start(cfg); err != nil {
		log.Fatal(err)
	}
	defer macgo.Cleanup()

	// Initialize NSApplication to ensure WindowServer connection
	// Use appkit bindings for type safety
	nsApp := appkit.GetApplicationClass().SharedApplication()

	// Initialize Accessibility bindings
	initAX()

	appName := flag.String("app", "", "Application name to capture")
	port := flag.Int("port", 8081, "Port to serve on")
	flag.Parse()

	if *appName == "" {
		log.Fatal("Please provide an app name using -app")
	}

	// Move logic to dispatch queue to free up main thread for NSApplication.Run
	queue := dispatch.GetGlobalQueue(dispatch.QOSUserInteractive)
	queue.Async(func() {
		// Retry loop to find window
		found := false
		for i := 0; i < 5; i++ {
			windows, err := getWindowList()
			if err != nil {
				log.Printf("Error getting window list: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}

			for _, w := range windows {
				if strings.Contains(strings.ToLower(w.AppName), strings.ToLower(*appName)) {
					// Filter out small windows or empty titles if needed
					if w.Frame.Size.Width > 50 && w.Frame.Size.Height > 50 {
						targetWindow = w
						found = true
						fmt.Printf("Selected window: %s - %s (ID: %d) %.0fx%.0f\n",
							w.AppName, w.Title, w.ID, w.Frame.Size.Width, w.Frame.Size.Height)
						break
					}
				}
			}
			if found {
				break
			}
			time.Sleep(1 * time.Second)
		}

		if !found {
			log.Fatalf("Could not find window for app: %s", *appName)
		}

		// Start AX reader
		go startAXReaderImproved()

		// Start Window Frame Updater
		go startWindowFrameUpdater()

		// Start server
		http.HandleFunc("/", serveIndex)
		http.HandleFunc("/stream", serveStream)
		http.HandleFunc("/click", handleClick)
		http.HandleFunc("/accessibility", serveAccessibilitySSE_Real)
		http.HandleFunc("/element-image", serveElementImage)

		addr := fmt.Sprintf(":%d", *port)
		fmt.Printf("Serving at http://localhost%s\n", addr)

		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Fatal(err)
		}
	})

	// Run main application loop on the main thread
	nsApp.Run()
}

func startWindowFrameUpdater() {
	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		mu.Lock()
		id := targetWindow.ID
		mu.Unlock()

		if id == 0 {
			continue
		}

		windows, err := getWindowList()
		if err != nil {
			continue
		}

		for _, w := range windows {
			if w.ID == id {
				mu.Lock()
				targetWindow = w
				mu.Unlock()
				break
			}
		}
	}
}

func getWindowList() ([]Window, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var result []Window
	var resultErr error
	done := make(chan struct{})

	handler := func(content screencapturekit.ShareableContent, err error) {
		defer close(done)
		if err != nil {
			resultErr = err
			return
		}

		// Workaround for purego slice panic: manually access NSArray
		windowsID := objc.Send[objc.ID](content.ID, objc.Sel("windows"))
		count := objc.Send[uint](windowsID, objc.Sel("count"))

		for i := 0; i < int(count); i++ {
			winID := objc.Send[objc.ID](windowsID, objc.Sel("objectAtIndex:"), uint(i))
			scWin := screencapturekit.WindowFrom(unsafe.Pointer(winID))

			// Frame() returns corefoundation.CGRect, need to convert to coregraphics.CGRect
			frameCF := scWin.Frame()
			frame := coregraphics.CGRect{
				Origin: coregraphics.CGPoint{X: frameCF.Origin.X, Y: frameCF.Origin.Y},
				Size:   coregraphics.CGSize{Width: frameCF.Size.Width, Height: frameCF.Size.Height},
			}

			w := Window{
				SCWindow: scWin,
				ID:       uint32(scWin.WindowID()),
				Title:    scWin.Title(),
				Frame:    frame,
			}

			app := scWin.OwningApplication()
			if app.GetID() != 0 {
				w.AppName = app.ApplicationName()
				// ProcessID returns unsafe.Pointer, cast to int32
				w.PID = int32(uintptr(app.ProcessID()))
			}

			result = append(result, w)
		}
	}

	scClass := screencapturekit.GetShareableContentClass()

	// Pass completion handler
	scClass.GetShareableContentExcludingDesktopWindowsOnScreenWindowsOnlyCompletionHandler(true, true, handler)

	select {
	case <-done:
		return result, resultErr
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

var (
	listenersMu sync.Mutex
	listeners   = make(map[chan *AXNode]struct{})
)

func broadcastAX(tree *AXNode) {
	listenersMu.Lock()
	defer listenersMu.Unlock()
	for ch := range listeners {
		select {
		case ch <- tree:
		default:
			// Skip if blocked
		}
	}
}

func startAXReaderImproved() {
	ticker := time.NewTicker(2 * time.Second)
	var lastTreeJSON []byte

	for range ticker.C {
		mu.Lock()
		pid := targetWindow.PID
		mu.Unlock()
		if pid == 0 {
			continue
		}

		tree, err := getAXTree(pid)
		if err != nil {
			log.Printf("AX Error: %v", err)
			continue
		}

		// Check for diff
		newJSON, _ := json.Marshal(tree)
		if string(newJSON) == string(lastTreeJSON) {
			continue
		}
		lastTreeJSON = newJSON

		broadcastAX(tree)
	}
}

func serveAccessibilitySSE_Real(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "No streaming", 500)
		return
	}

	// Subscribe
	ch := make(chan *AXNode, 1)
	listenersMu.Lock()
	listeners[ch] = struct{}{}
	listenersMu.Unlock()

	defer func() {
		listenersMu.Lock()
		delete(listeners, ch)
		listenersMu.Unlock()
	}()

	// Send immediate initial if avail (could store last cached in global)
	// For now we wait for first update or next tick.

	for {
		select {
		case <-r.Context().Done():
			return
		case tree := <-ch:
			mu.Lock()
			winFrame := targetWindow.Frame
			mu.Unlock()

			resp := struct {
				Tree        *AXNode             `json:"tree"`
				WindowFrame coregraphics.CGRect `json:"windowFrame"`
			}{
				Tree:        tree,
				WindowFrame: winFrame,
			}
			d, _ := json.Marshal(resp)
			fmt.Fprintf(w, "data: %s\n\n", d)
			flusher.Flush()
		}
	}
}

func serveStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")

	for {
		mu.Lock()
		win := targetWindow
		mu.Unlock()

		imgData, width, height, err := captureWindowImage(win.SCWindow)
		if err != nil {
			log.Printf("Capture error: %v", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		fmt.Fprintf(w, "--frame\r\n")
		fmt.Fprintf(w, "Content-Type: image/jpeg\r\n")
		fmt.Fprintf(w, "Content-Length: %d\r\n", len(imgData))
		fmt.Fprintf(w, "X-Image-Width: %d\r\n", width)
		fmt.Fprintf(w, "X-Image-Height: %d\r\n", height)
		fmt.Fprintf(w, "\r\n")
		w.Write(imgData)
		fmt.Fprintf(w, "\r\n")

		time.Sleep(50 * time.Millisecond)
	}
}

func captureWindowImage(scWindow screencapturekit.Window) ([]byte, int, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var imgKilled bool
	var imgData []byte
	var resultErr error
	done := make(chan struct{}, 1) // Buffered to prevent blocking

	// Get dimensions from window frame
	frame := scWindow.Frame()
	width := int(frame.Size.Width)
	height := int(frame.Size.Height)

	// Create Filter
	filter := screencapturekit.NewContentFilterWithDesktopIndependentWindow(scWindow)

	// Create Config
	config := screencapturekit.NewStreamConfiguration()
	config.SetWidth(uintptr(width))
	config.SetHeight(uintptr(height))
	config.SetShowsCursor(true)

	// Handler receives ImageRef
	handler := func(imageInt screencapturekit.ImageRef, err error) {
		if imgKilled {
			return
		}

		if err != nil {
			resultErr = err
			done <- struct{}{}
			return
		}

		// Convert int to CGImageRef (uintptr)
		image := coregraphics.ImageRef(uintptr(imageInt))

		if image == 0 {
			resultErr = fmt.Errorf("got nil image")
			done <- struct{}{}
			return
		}

		// Retain and store current image for element slicing
		imgMu.Lock()
		if currentImage != 0 {
			cfRelease(uintptr(currentImage))
		}
		currentImage = coregraphics.CGImageRetain(image)
		imgMu.Unlock()

		// Create dest
		data := corefoundation.CFDataCreateMutable(0, 0)

		// Create "public.jpeg" string
		// kCFStringEncodingUTF8 is 0x08000100
		jpegBytePtr, _ := syscall.BytePtrFromString("public.jpeg")
		jpegTypeCF := corefoundation.CFStringCreateWithCString(0, unsafe.Pointer(jpegBytePtr), 0x08000100)

		// CGImageDestinationCreateWithData
		dest := imageio.CGImageDestinationCreateWithData(uintptr(data), uintptr(jpegTypeCF), 1, 0)

		if dest == 0 {
			resultErr = fmt.Errorf("failed to create image destination")
			done <- struct{}{}
			return
		}

		imageio.CGImageDestinationAddImage(dest, imageio.ImageRef(image), 0)
		if imageio.CGImageDestinationFinalize(dest) {
			// Get bytes using manual binding
			l := corefoundation.CFDataGetLength(corefoundation.DataRef(data))
			ptr := cfDataGetBytePtr(uintptr(data))

			// Copy bytes
			imgData = make([]byte, int(l))
			if l > 0 {
				src := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), int(l))
				copy(imgData, src)
			}
		} else {
			resultErr = fmt.Errorf("failed to finalize image destination")
		}
		done <- struct{}{}
	}

	// Capture
	screencapturekit.GetScreenshotManagerClass().CaptureImageWithFilterConfigurationCompletionHandler(filter, config, handler)

	select {
	case <-done:
		return imgData, width, height, resultErr
	case <-ctx.Done():
		imgKilled = true
		return nil, 0, 0, fmt.Errorf("timeout capturing image")
	}
}

func handleClick(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		X      int `json:"x"`
		Y      int `json:"y"`
		Width  int `json:"width"`
		Height int `json:"height"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	mu.Lock()
	win := targetWindow
	mu.Unlock()

	scaleX := float64(payload.Width) / win.Frame.Size.Width
	scaleY := float64(payload.Height) / win.Frame.Size.Height

	// Avoid div by zero
	if scaleX == 0 {
		scaleX = 1
	}
	if scaleY == 0 {
		scaleY = 1
	}

	clickPointX := float64(payload.X) / scaleX
	clickPointY := float64(payload.Y) / scaleY

	absX := win.Frame.Origin.X + clickPointX
	absY := win.Frame.Origin.Y + clickPointY

	if *verbose {
		log.Printf("Debug Click:\n\tPayload: %dx%d @ %d,%d\n\tWinFrame: %.0fx%.0f @ %.0f,%.0f\n\tScale: %.3f, %.3f\n\tRelPoint: %.1f, %.1f\n\tAbsPoint: %.1f, %.1f",
			payload.Width, payload.Height, payload.X, payload.Y,
			win.Frame.Size.Width, win.Frame.Size.Height, win.Frame.Origin.X, win.Frame.Origin.Y,
			scaleX, scaleY,
			clickPointX, clickPointY,
			absX, absY,
		)
	}

	// Restore mouse position
	// Get current location to restore later
	eventSource := coregraphics.EventSourceRef(0)
	currentEvent := coregraphics.CGEventCreate(eventSource)
	currentLoc := coregraphics.CGEventGetLocation(currentEvent)
	cfRelease(uintptr(currentEvent))

	point := coregraphics.CGPoint{X: absX, Y: absY}

	if *verbose {
		fmt.Printf("Clicking at: %.0f, %.0f (Window relative: %.0f, %.0f)\n", absX, absY, clickPointX, clickPointY)
	}

	// Constants for events
	const (
		kCGEventLeftMouseDown = 1
		kCGEventLeftMouseUp   = 2
		kCGEventMouseMoved    = 5
		kCGMouseButtonLeft    = 0
	)

	mouseDown := coregraphics.CGEventCreateMouseEvent(0, coregraphics.EventType(kCGEventLeftMouseDown), point, coregraphics.MouseButton(kCGMouseButtonLeft))
	defer cfRelease(uintptr(mouseDown))

	// Fallback to standard Session Event Tap for reliability.
	// PID posting requires perfect ABI and Perms.
	// kCGSessionEventTap = 0 is default and usually works for automation if app has permissions.
	if *verbose {
		fmt.Printf("Posting MouseDown at %.0f,%.0f\n", point.X, point.Y)
	}
	coregraphics.CGEventPost(coregraphics.EventTapLocation(1), mouseDown)

	// Small delay
	time.Sleep(10 * time.Millisecond)

	mouseUp := coregraphics.CGEventCreateMouseEvent(0, coregraphics.EventType(kCGEventLeftMouseUp), point, coregraphics.MouseButton(kCGMouseButtonLeft))
	defer cfRelease(uintptr(mouseUp))

	if *verbose {
		fmt.Println("Posting MouseUp")
	}
	coregraphics.CGEventPost(coregraphics.EventTapLocation(1), mouseUp)

	// Restore previous location
	mouseRestore := coregraphics.CGEventCreateMouseEvent(0, coregraphics.EventType(kCGEventMouseMoved), currentLoc, 0)
	defer cfRelease(uintptr(mouseRestore))

	if *verbose {
		fmt.Printf("Restoring mouse to %.0f,%.0f\n", currentLoc.X, currentLoc.Y)
	}
	coregraphics.CGEventPost(coregraphics.EventTapLocation(1), mouseRestore)

	w.WriteHeader(200)
}

func serveElementImage(w http.ResponseWriter, r *http.Request) {
	xStr := r.URL.Query().Get("x")
	yStr := r.URL.Query().Get("y")
	wStr := r.URL.Query().Get("w")
	hStr := r.URL.Query().Get("h")

	if xStr == "" || yStr == "" || wStr == "" || hStr == "" {
		http.Error(w, "missing params", 400)
		return
	}

	x, _ := strconv.Atoi(xStr)
	y, _ := strconv.Atoi(yStr)
	wd, _ := strconv.Atoi(wStr)
	ht, _ := strconv.Atoi(hStr)

	imgMu.Lock()
	if currentImage == 0 {
		imgMu.Unlock()
		http.Error(w, "no image", 404)
		return
	}

	imgW := int(coregraphics.CGImageGetWidth(currentImage))
	imgH := int(coregraphics.CGImageGetHeight(currentImage))

	// Basic sanitation
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	// Check bounds
	if x >= imgW || y >= imgH {
		imgMu.Unlock()
		http.Error(w, fmt.Sprintf("out of bounds: %d,%d vs %dx%d", x, y, imgW, imgH), 400)
		return
	}

	// Clamp width/height
	if x+wd > imgW {
		wd = imgW - x
	}
	if y+ht > imgH {
		ht = imgH - y
	}

	if wd <= 0 || ht <= 0 {
		imgMu.Unlock()
		http.Error(w, "invalid dimensions", 400)
		return
	}

	rect := coregraphics.CGRect{
		Origin: coregraphics.CGPoint{X: float64(x), Y: float64(y)},
		Size:   coregraphics.CGSize{Width: float64(wd), Height: float64(ht)},
	}

	subImg := coregraphics.CGImageCreateWithImageInRect(currentImage, rect)
	imgMu.Unlock()

	if subImg == 0 {
		http.Error(w, fmt.Sprintf("failed to slice: rect=%+v on img %dx%d", rect, imgW, imgH), 500)
		return
	}
	defer cfRelease(uintptr(subImg))

	// Convert to JPEG
	data := corefoundation.CFDataCreateMutable(0, 0)
	defer cfRelease(uintptr(data))

	jpegBytePtr, _ := syscall.BytePtrFromString("public.jpeg")
	jpegTypeCF := corefoundation.CFStringCreateWithCString(0, unsafe.Pointer(jpegBytePtr), 0x08000100)
	defer cfRelease(uintptr(jpegTypeCF))

	dest := imageio.CGImageDestinationCreateWithData(uintptr(data), uintptr(jpegTypeCF), 1, 0)
	if dest == 0 {
		http.Error(w, "failed to create dest", 500)
		return
	}
	defer cfRelease(uintptr(dest))

	imageio.CGImageDestinationAddImage(dest, imageio.ImageRef(subImg), 0)
	if imageio.CGImageDestinationFinalize(dest) {
		l := corefoundation.CFDataGetLength(corefoundation.DataRef(data))
		ptr := cfDataGetBytePtr(uintptr(data))

		bytes := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), int(l))
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", l))
		w.Write(bytes)
	} else {
		http.Error(w, "failed to finalize", 500)
	}
}

var (
	currentImage coregraphics.ImageRef
	imgMu        sync.Mutex
)

func serveIndex(w http.ResponseWriter, r *http.Request) {
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>App Stream</title>
    <style>
        body { margin: 0; background: #333; display: flex; justify-content: center; align-items: center; height: 100vh; overflow: hidden; }
        #container { position: relative; width: 100vw; height: 100vh; display: flex; justify-content: center; align-items: center; }
        
        #stream {
            max-width: 100%;
            max-height: 100%;
            width: auto;
            height: auto;
            display: block; /* Removes bottom spacing */
            /* opacity: 0.2; Removed opacity to make stream visible */
            cursor: crosshair;
        }

        #ax-container {
             position: absolute;
             /* dimensions will be set by JS to match #stream */
             pointer-events: none; /* Let clicks pass through empty spaces */
        }
        
        .click-indicator {
            position: absolute; width: 20px; height: 20px; background: rgba(255, 0, 0, 0.7);
            border-radius: 50%; pointer-events: none; transform: translate(-50%, -50%);
            transition: opacity 0.5s ease-out; z-index: 2000;
        }
        
        .ax-node {
            position: absolute;
            background: transparent; /* No images for efficiency */
            /* border: 1px solid rgba(0, 0, 0, 0.1); */
            cursor: pointer;
            box-sizing: border-box;
            pointer-events: auto; /* Capture interactions on nodes */
        }
        .ax-node:hover {
            border: 1px solid cyan;
            z-index: 10;
        }
    </style>
</head>
<body>
    <div id="container">
        <img id="stream" src="/stream" />
        <div id="ax-container"></div>
    </div>
    <script>
        const img = document.getElementById('stream');
        const overlay = document.getElementById('ax-container');
        const container = document.getElementById('container');
        
        let windowFrame = null;
        
        function showClick(x, y) {
            const indicator = document.createElement('div');
            indicator.className = 'click-indicator';
            indicator.style.left = x + 'px';
            indicator.style.top = y + 'px';
            document.body.appendChild(indicator);
            requestAnimationFrame(() => {
                setTimeout(() => {
                    indicator.style.opacity = '0';
                    setTimeout(() => document.body.removeChild(indicator), 500);
                }, 50);
            });
        }
        
        function renderTree(node) {
            if (!node) return;
            
            if (node.frame && node.frame.width > 0 && node.frame.height > 0) {
                if (windowFrame) {
                    const rect = img.getBoundingClientRect();
                    
                    const scaleX = rect.width / windowFrame.Size.Width;
                    const scaleY = rect.height / windowFrame.Size.Height;
                    
                    const relX = node.frame.x - windowFrame.Origin.X;
                    const relY = node.frame.y - windowFrame.Origin.Y;
                    
                    const renderW = node.frame.width * scaleX;
                    const renderH = node.frame.height * scaleY;
                    
                    // Helper to intersect rects to ensure we only render reliable nodes within window
                    // (Though if they are transparent, it matters less, but good for layout)
                    const intersect = (r1, r2) => {
                        const x = Math.max(r1.x, r2.x);
                        const y = Math.max(r1.y, r2.y);
                        const w = Math.min(r1.x + r1.w, r2.x + r2.w) - x;
                        const h = Math.min(r1.y + r1.h, r2.y + r2.h) - y;
                        if (w <= 0 || h <= 0) return null;
                        return {x, y, w, h};
                    };
                    
                    const elRect = {x: relX, y: relY, w: node.frame.width, h: node.frame.height};
                    const winRect = {x: 0, y: 0, w: windowFrame.Size.Width, h: windowFrame.Size.Height};
                    const visible = intersect(elRect, winRect);
                    
                    // Render if visible-ish
                    if (visible && visible.w > 5 && visible.h > 5) {
                        const el = document.createElement('div');
                        el.className = 'ax-node';
                        
                        const visScaleX = rect.width / windowFrame.Size.Width; 
                        const visScaleY = rect.height / windowFrame.Size.Height;
                        
                        el.style.left = (visible.x * visScaleX) + 'px';
                        el.style.top = (visible.y * visScaleY) + 'px';
                        el.style.width = (visible.w * visScaleX) + 'px';
                        el.style.height = (visible.h * visScaleY) + 'px';
                        
                        // No background image request!
                        
                        el.title = node.role;
                        el.dataset.role = node.role;
                        if (node.title) el.dataset.title = node.title;
                        if (node.subrole) el.dataset.subrole = node.subrole;
                        
                        el.setAttribute('role', 'button');
                        let ariaLabel = node.role;
                        if (node.title) ariaLabel += ": " + node.title;
                        el.setAttribute('aria-label', ariaLabel);
                        
                        el.onmousedown = (e) => {
                             e.stopPropagation(); 
                             handleGenericClick(e);
                        };
                        
                        overlay.appendChild(el);
                    }
                }
            }
            
            if (node.children) {
                node.children.forEach(renderTree);
            }
        }
        
        function handleGenericClick(e) {
             const rect = img.getBoundingClientRect();
             const clickX = e.clientX - rect.left;
             const clickY = e.clientY - rect.top;
             
             showClick(e.pageX, e.pageY);
             
             const scaleX = img.naturalWidth / rect.width;
             const scaleY = img.naturalHeight / rect.height;
             
             fetch('/click', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({
                    x: Math.round(clickX * scaleX),
                    y: Math.round(clickY * scaleY),
                    width: img.naturalWidth,
                    height: img.naturalHeight
                })
            });
        }
        
        function resizeOverlay() {
            const rect = img.getBoundingClientRect();
            overlay.style.width = rect.width + 'px';
            overlay.style.height = rect.height + 'px';
            overlay.style.left = rect.left + 'px'; 
            overlay.style.top = rect.top + 'px'; 
        }
        
        function setupAccessibilityStream() {
            const source = new EventSource('/accessibility');
            source.onmessage = function(event) {
                try {
                    const data = JSON.parse(event.data);
                    windowFrame = data.windowFrame;
                    overlay.innerHTML = '';
                    renderTree(data.tree);
                    resizeOverlay(); 
                } catch(e) {
                    console.error("AX Parse Error:", e);
                }
            };
        }
        
        window.onresize = () => { resizeOverlay(); }; 
        img.onload = () => { resizeOverlay(); };
        img.onmousedown = (e) => { handleGenericClick(e); }; // Fallback click
        
        setupAccessibilityStream();
    </script>
</body>
</html>
    `
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}
