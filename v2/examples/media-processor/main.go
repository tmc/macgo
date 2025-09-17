// Package main demonstrates media processing using the v2 macgo API.
// This example shows the simplified permission handling for hardware access.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	macgo "github.com/tmc/misc/macgo/v2"
)

// MediaInfo holds information about a media file
type MediaInfo struct {
	Filename    string        `json:"filename"`
	Format      string        `json:"format"`
	Duration    time.Duration `json:"duration"`
	VideoCodec  string        `json:"video_codec,omitempty"`
	AudioCodec  string        `json:"audio_codec,omitempty"`
	Resolution  string        `json:"resolution,omitempty"`
	Bitrate     string        `json:"bitrate,omitempty"`
	FrameRate   float64       `json:"frame_rate,omitempty"`
	FileSize    int64         `json:"file_size"`
	HasVideo    bool          `json:"has_video"`
	HasAudio    bool          `json:"has_audio"`
}

// ProcessOptions holds media processing options
type ProcessOptions struct {
	OutputFormat   string
	VideoCodec     string
	AudioCodec     string
	Resolution     string
	Bitrate        string
	FrameRate      int
	AudioChannels  int
	HardwareAccel  bool
	Preset         string
	Quality        int
	StartTime      time.Duration
	Duration       time.Duration
	RemoveAudio    bool
	ExtractAudio   bool
	GenerateThumbs bool
	ThumbCount     int
}

func main() {
	// Parse command-line flags
	var (
		input         = flag.String("input", "", "Input media file")
		output        = flag.String("output", "", "Output file path")
		format        = flag.String("format", "", "Output format (mp4, webm, mov, mp3, etc.)")
		preset        = flag.String("preset", "medium", "Encoding preset (fast, medium, slow, best)")
		resolution    = flag.String("res", "", "Output resolution (e.g., 1920x1080, 720p)")
		bitrate       = flag.String("bitrate", "", "Target bitrate (e.g., 2M, 128k)")
		fps           = flag.Int("fps", 0, "Target frame rate")
		quality       = flag.Int("quality", 23, "Quality (0-51, lower is better)")
		hwaccel       = flag.Bool("hwaccel", true, "Use hardware acceleration if available")
		extractAudio  = flag.Bool("extract-audio", false, "Extract audio only")
		removeAudio   = flag.Bool("remove-audio", false, "Remove audio from video")
		thumbnail     = flag.Bool("thumbnail", false, "Generate thumbnails")
		thumbCount    = flag.Int("thumb-count", 10, "Number of thumbnails to generate")
		trim          = flag.String("trim", "", "Trim video (format: start,duration in seconds)")
		info          = flag.Bool("info", false, "Show media information only")
		batch         = flag.String("batch", "", "Batch process directory")
		liveCapture   = flag.Bool("live", false, "Capture from camera/microphone")
		deviceList    = flag.Bool("list-devices", false, "List available capture devices")
	)
	flag.Parse()

	// Build permissions based on what's needed - much cleaner in v2!
	permissions := []macgo.Permission{
		macgo.Files,   // Read/write media files
		macgo.Network, // Optional: for streaming or remote processing
	}

	// Add media permissions for live capture if needed
	if *liveCapture || *deviceList {
		permissions = append(permissions,
			macgo.Camera,     // Camera access
			macgo.Microphone, // Microphone access
		)
	}

	// Configure macgo v2
	cfg := &macgo.Config{
		AppName:     "MediaProcessor",
		BundleID:    "com.example.mediaprocessor",
		Permissions: permissions,
		LSUIElement: true, // Hide from dock for CLI usage
		Debug:       os.Getenv("MACGO_DEBUG") == "1",

		// Custom privacy descriptions - much cleaner than v1!
		Custom: []string{
			"NSCameraUsageDescription:This app needs camera access to capture video.",
			"NSMicrophoneUsageDescription:This app needs microphone access to capture audio.",
			// Note: v2 automatically handles hardware acceleration entitlements
		},
	}

	// Start macgo
	if err := macgo.Start(cfg); err != nil {
		log.Fatalf("Failed to start macgo: %v", err)
	}

	fmt.Printf("🎬 Media Processor (v2 API)\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	// Execute requested operation
	switch {
	case *deviceList:
		listCaptureDevices()

	case *liveCapture:
		if *output == "" {
			log.Fatal("Output file required for live capture")
		}
		captureLive(*output, parseTrimDuration(*trim))

	case *info && *input != "":
		info := getMediaInfo(*input)
		showMediaInfo(info)

	case *batch != "":
		processBatch(*batch, *output, buildOptions(flag.CommandLine))

	case *input != "" && *output != "":
		opts := buildOptions(flag.CommandLine)
		processMedia(*input, *output, opts)

	default:
		flag.Usage()
		fmt.Println("\nExamples (v2 API):")
		fmt.Println("  # Convert video to MP4")
		fmt.Println("  media-processor -input video.mov -output video.mp4")
		fmt.Println()
		fmt.Println("  # Extract audio from video")
		fmt.Println("  media-processor -input video.mp4 -output audio.mp3 -extract-audio")
		fmt.Println()
		fmt.Println("  # Resize video to 720p with hardware acceleration")
		fmt.Println("  media-processor -input video.mp4 -output video_720p.mp4 -res 720p -hwaccel")
		fmt.Println()
		fmt.Println("  # Generate thumbnails")
		fmt.Println("  media-processor -input video.mp4 -output thumbs/ -thumbnail")
		fmt.Println()
		fmt.Println("  # Live capture from camera (v2 simplified permissions)")
		fmt.Println("  media-processor -live -output recording.mp4 -trim 0,30")
		fmt.Println()
		fmt.Println("v2 Benefits:")
		fmt.Println("  • Automatic hardware acceleration support")
		fmt.Println("  • Simplified permission configuration")
		fmt.Println("  • Cross-platform safe operation")
	}
}

func processMedia(input, output string, opts *ProcessOptions) {
	fmt.Printf("🎬 Processing media file (v2 API)...\n")
	fmt.Printf("   Input: %s\n", input)
	fmt.Printf("   Output: %s\n", output)

	// Get media info
	info := getMediaInfo(input)
	fmt.Printf("   Format: %s", info.Format)
	if info.HasVideo {
		fmt.Printf(" (Video: %s @ %s)", info.VideoCodec, info.Resolution)
	}
	if info.HasAudio {
		fmt.Printf(" (Audio: %s)", info.AudioCodec)
	}
	fmt.Println()

	// Build ffmpeg command
	args := buildFFmpegArgs(input, output, info, opts)

	// Execute ffmpeg
	start := time.Now()
	cmd := exec.Command("ffmpeg", args...)
	cmd.Stderr = os.Stderr

	if opts.HardwareAccel {
		fmt.Println("   Hardware acceleration: enabled (v2 auto-configured)")
	}

	fmt.Printf("\n⏳ Processing...")
	if err := cmd.Run(); err != nil {
		log.Fatalf("\n❌ Processing failed: %v", err)
	}

	duration := time.Since(start)
	outputInfo, _ := os.Stat(output)

	fmt.Printf("\n✅ Processing completed in %.2fs (v2 API)\n", duration.Seconds())
	if outputInfo != nil {
		fmt.Printf("   Output size: %.2f MB\n", float64(outputInfo.Size())/(1024*1024))
	}
}

func buildFFmpegArgs(input, output string, info *MediaInfo, opts *ProcessOptions) []string {
	args := []string{"-i", input}

	// Hardware acceleration - v2 automatically configures this
	if opts.HardwareAccel && info.HasVideo {
		// Use VideoToolbox on macOS - v2 handles entitlements automatically
		args = append([]string{"-hwaccel", "videotoolbox"}, args...)
	}

	// Trim options
	if opts.StartTime > 0 {
		args = append(args, "-ss", fmt.Sprintf("%.2f", opts.StartTime.Seconds()))
	}
	if opts.Duration > 0 {
		args = append(args, "-t", fmt.Sprintf("%.2f", opts.Duration.Seconds()))
	}

	// Video options
	if info.HasVideo && !opts.ExtractAudio {
		if opts.VideoCodec != "" {
			args = append(args, "-c:v", opts.VideoCodec)
		} else if opts.HardwareAccel {
			args = append(args, "-c:v", "h264_videotoolbox")
		}

		if opts.Resolution != "" {
			scale := parseResolution(opts.Resolution)
			args = append(args, "-vf", fmt.Sprintf("scale=%s", scale))
		}

		if opts.Bitrate != "" {
			args = append(args, "-b:v", opts.Bitrate)
		}

		if opts.FrameRate > 0 {
			args = append(args, "-r", fmt.Sprintf("%d", opts.FrameRate))
		}

		args = append(args, "-crf", fmt.Sprintf("%d", opts.Quality))
		args = append(args, "-preset", opts.Preset)
	}

	// Audio options
	if opts.RemoveAudio {
		args = append(args, "-an")
	} else if opts.ExtractAudio {
		args = append(args, "-vn")
		if opts.AudioCodec != "" {
			args = append(args, "-c:a", opts.AudioCodec)
		}
	} else if info.HasAudio {
		if opts.AudioCodec != "" {
			args = append(args, "-c:a", opts.AudioCodec)
		}
		if opts.AudioChannels > 0 {
			args = append(args, "-ac", fmt.Sprintf("%d", opts.AudioChannels))
		}
	}

	// Output file
	args = append(args, "-y", output)

	return args
}

func getMediaInfo(path string) *MediaInfo {
	info := &MediaInfo{
		Filename: filepath.Base(path),
	}

	// Get file info
	if stat, err := os.Stat(path); err == nil {
		info.FileSize = stat.Size()
	}

	// Use ffprobe to get media information
	cmd := exec.Command("ffprobe", "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", path)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Warning: Could not probe media file: %v", err)
		return info
	}

	// Parse ffprobe output
	var probe struct {
		Format struct {
			FormatName string `json:"format_name"`
			Duration   string `json:"duration"`
			BitRate    string `json:"bit_rate"`
		} `json:"format"`
		Streams []struct {
			CodecType string  `json:"codec_type"`
			CodecName string  `json:"codec_name"`
			Width     int     `json:"width"`
			Height    int     `json:"height"`
			FrameRate string  `json:"r_frame_rate"`
		} `json:"streams"`
	}

	if err := json.Unmarshal(output, &probe); err == nil {
		info.Format = probe.Format.FormatName
		info.Bitrate = probe.Format.BitRate

		if d, err := time.ParseDuration(probe.Format.Duration + "s"); err == nil {
			info.Duration = d
		}

		for _, stream := range probe.Streams {
			if stream.CodecType == "video" {
				info.HasVideo = true
				info.VideoCodec = stream.CodecName
				info.Resolution = fmt.Sprintf("%dx%d", stream.Width, stream.Height)
			} else if stream.CodecType == "audio" {
				info.HasAudio = true
				info.AudioCodec = stream.CodecName
			}
		}
	}

	return info
}

func showMediaInfo(info *MediaInfo) {
	fmt.Println("📊 Media Information (v2 API)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("📄 File: %s\n", info.Filename)
	fmt.Printf("📦 Format: %s\n", info.Format)
	fmt.Printf("📏 Size: %.2f MB\n", float64(info.FileSize)/(1024*1024))
	fmt.Printf("⏱ Duration: %s\n", info.Duration)

	if info.HasVideo {
		fmt.Println("\n🎥 Video:")
		fmt.Printf("   Codec: %s\n", info.VideoCodec)
		fmt.Printf("   Resolution: %s\n", info.Resolution)
		if info.Bitrate != "" {
			fmt.Printf("   Bitrate: %s\n", info.Bitrate)
		}
	}

	if info.HasAudio {
		fmt.Println("\n🔊 Audio:")
		fmt.Printf("   Codec: %s\n", info.AudioCodec)
	}

	fmt.Println("\n✨ v2 API Features:")
	fmt.Println("   • Automatic hardware acceleration entitlements")
	fmt.Println("   • Simplified permission configuration")
	fmt.Println("   • Cross-platform operation")
}

func processBatch(inputDir, outputDir string, opts *ProcessOptions) {
	fmt.Printf("📁 Batch processing directory (v2 API): %s\n", inputDir)

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Cannot create output directory: %v", err)
	}

	// Find media files
	patterns := []string{"*.mp4", "*.mov", "*.avi", "*.mkv", "*.mp3", "*.wav"}
	var files []string

	for _, pattern := range patterns {
		matches, _ := filepath.Glob(filepath.Join(inputDir, pattern))
		files = append(files, matches...)
	}

	if len(files) == 0 {
		fmt.Println("No media files found")
		return
	}

	fmt.Printf("Found %d media file(s)\n\n", len(files))

	// Process each file
	for i, file := range files {
		fmt.Printf("[%d/%d] Processing %s\n", i+1, len(files), filepath.Base(file))

		outputFile := filepath.Join(outputDir, filepath.Base(file))
		if opts.OutputFormat != "" {
			ext := filepath.Ext(outputFile)
			outputFile = strings.TrimSuffix(outputFile, ext) + "." + opts.OutputFormat
		}

		processMedia(file, outputFile, opts)
		fmt.Println()
	}

	fmt.Printf("✅ Batch processing completed: %d file(s) (v2 API)\n", len(files))
}

func captureLive(output string, duration time.Duration) {
	fmt.Println("📹 Starting live capture (v2 API)...")
	fmt.Printf("   Output: %s\n", output)
	if duration > 0 {
		fmt.Printf("   Duration: %v\n", duration)
	} else {
		fmt.Println("   Press Ctrl+C to stop")
	}

	// Use ffmpeg to capture from default devices
	args := []string{
		"-f", "avfoundation",
		"-framerate", "30",
		"-i", "0:0", // Default video:audio devices
	}

	if duration > 0 {
		args = append(args, "-t", fmt.Sprintf("%.2f", duration.Seconds()))
	}

	// Output settings - v2 automatically handles hardware encoding entitlements
	args = append(args,
		"-c:v", "h264_videotoolbox", // Hardware encoding (v2 auto-configured)
		"-c:a", "aac",
		"-b:a", "128k",
		"-preset", "fast",
		output,
	)

	cmd := exec.Command("ffmpeg", args...)
	cmd.Stderr = os.Stderr

	fmt.Println("\n🔴 Recording... (hardware acceleration auto-enabled)")
	if err := cmd.Run(); err != nil {
		// Ignore error if interrupted
		if !strings.Contains(err.Error(), "signal") {
			log.Fatalf("Capture failed: %v", err)
		}
	}

	fmt.Println("✅ Recording saved to", output)
}

func listCaptureDevices() {
	fmt.Println("📹 Available Capture Devices (v2 API)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// List AVFoundation devices
	cmd := exec.Command("ffmpeg", "-f", "avfoundation", "-list_devices", "true", "-i", "")
	output, _ := cmd.CombinedOutput()

	// Parse and display devices
	lines := strings.Split(string(output), "\n")
	videoDevices := false
	audioDevices := false

	fmt.Println("\n🎥 Video Devices:")
	for _, line := range lines {
		if strings.Contains(line, "AVFoundation video devices") {
			videoDevices = true
			audioDevices = false
			continue
		}
		if strings.Contains(line, "AVFoundation audio devices") {
			videoDevices = false
			audioDevices = true
			fmt.Println("\n🎤 Audio Devices:")
			continue
		}

		if (videoDevices || audioDevices) && strings.Contains(line, "]") {
			parts := strings.Split(line, "] ")
			if len(parts) > 1 {
				fmt.Printf("  %s\n", parts[1])
			}
		}
	}

	fmt.Println("\n💡 v2 API Benefits:")
	fmt.Println("  • Automatic permission handling for media devices")
	fmt.Println("  • Hardware acceleration auto-configured")
	fmt.Println("  • Cross-platform safe operation")
	fmt.Println()
	fmt.Println("Usage Examples:")
	fmt.Println("  media-processor -live -output recording.mp4")
	fmt.Println("  ffmpeg -f avfoundation -i \"0:0\" output.mp4  # Device 0 video, Device 0 audio")
}

func buildOptions(flags *flag.FlagSet) *ProcessOptions {
	opts := &ProcessOptions{
		Preset:        "medium",
		Quality:       23,
		HardwareAccel: true,
	}

	flags.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "format":
			opts.OutputFormat = f.Value.String()
		case "preset":
			opts.Preset = f.Value.String()
		case "res":
			opts.Resolution = f.Value.String()
		case "bitrate":
			opts.Bitrate = f.Value.String()
		case "fps":
			if v, _ := f.Value.(flag.Getter); v != nil {
				opts.FrameRate = v.Get().(int)
			}
		case "quality":
			if v, _ := f.Value.(flag.Getter); v != nil {
				opts.Quality = v.Get().(int)
			}
		case "hwaccel":
			if v, _ := f.Value.(flag.Getter); v != nil {
				opts.HardwareAccel = v.Get().(bool)
			}
		case "extract-audio":
			opts.ExtractAudio = true
		case "remove-audio":
			opts.RemoveAudio = true
		case "thumbnail":
			opts.GenerateThumbs = true
		case "thumb-count":
			if v, _ := f.Value.(flag.Getter); v != nil {
				opts.ThumbCount = v.Get().(int)
			}
		case "trim":
			start, duration := parseTrimDuration(f.Value.String())
			opts.StartTime = start
			opts.Duration = duration
		}
	})

	return opts
}

func parseResolution(res string) string {
	// Handle common resolution shortcuts
	switch res {
	case "1080p", "1080":
		return "1920:1080"
	case "720p", "720":
		return "1280:720"
	case "480p", "480":
		return "854:480"
	case "360p", "360":
		return "640:360"
	default:
		// Replace 'x' with ':' for ffmpeg
		return strings.Replace(res, "x", ":", 1)
	}
}

func parseTrimDuration(trim string) (start, duration time.Duration) {
	if trim == "" {
		return
	}

	parts := strings.Split(trim, ",")
	if len(parts) > 0 {
		if s, err := time.ParseDuration(parts[0] + "s"); err == nil {
			start = s
		}
	}
	if len(parts) > 1 {
		if d, err := time.ParseDuration(parts[1] + "s"); err == nil {
			duration = d
		}
	}

	return
}