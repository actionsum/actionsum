# IntegrationsV2 - Universal Application Detection

This is the V2 implementation of application detection for actionsum. It provides universal detection that works across all display servers, compositors, and desktop environments.

## Architecture

The V2 system uses a hybrid approach that combines multiple detection methods:

### 1. Window Detection (Primary)
- Uses compositor-specific APIs when available
- Works for X11 and Wayland (Sway, Hyprland, KDE, GNOME with XWayland)
- Provides the most accurate information (app name, window title, process)
- **Priority: 100** (highest)

### 2. Process Monitoring (Fallback)
- Monitors `/proc` filesystem for running GUI applications
- Works universally on all Linux systems
- Detects applications even when window detection fails
- **Priority: 50**

### 3. Input Activity Monitoring (Enhancement)
- Tracks which processes are actively using CPU
- Helps identify the most recently active application
- Enhances process-based detection accuracy

## How It Works

```
┌─────────────────────────────────────────────────┐
│           Hybrid Detector                       │
│                                                 │
│  ┌──────────────────┐   ┌──────────────────┐  │
│  │ Window Detector  │   │ Process Detector │  │
│  │ (X11/Wayland)    │   │ (/proc + input)  │  │
│  └──────────────────┘   └──────────────────┘  │
│          │                      │              │
│          │ Try first            │ Fallback     │
│          ├──────────────────────┤              │
│          │                      │              │
│          ▼                      ▼              │
│    ┌─────────────────────────────────┐        │
│    │   Best Available Detection      │        │
│    └─────────────────────────────────┘        │
└─────────────────────────────────────────────────┘
```

## Key Features

### Universal Compatibility
- ✅ Works on X11
- ✅ Works on Wayland (all compositors)
- ✅ Works on GNOME (both X11 and Wayland sessions)
- ✅ Detects both XWayland and native Wayland apps
- ✅ No compositor-specific limitations

### Intelligent Fallback
1. First attempts window detection via compositor APIs
2. If that fails, falls back to process monitoring
3. Combines both methods when possible for enhanced accuracy

### Process-Based Detection
The process detector:
- Scans `/proc` for running GUI applications
- Identifies GUI apps by checking for `DISPLAY` or `WAYLAND_DISPLAY` env vars
- Maintains a list of known GUI applications
- Scores processes based on recent activity
- Returns the most likely active application

### Confidence Scoring
Each detection includes a confidence score (0.0-1.0):
- **1.0**: Direct window detection (most accurate)
- **0.9**: Hybrid detection (process + window title match)
- **0.5-0.8**: Process-based with recent input activity
- **0.3-0.5**: Process-based without recent activity

## Directory Structure

```
pkg/integrationsV2/
├── common/          # Shared types and interfaces
│   └── types.go     # AppInfo, Detector interface
├── process/         # Process-based detection
│   └── detector.go  # Process monitor + input tracking
├── hybrid/          # Hybrid detector (combines methods)
│   └── detector.go  # Main hybrid implementation
└── README.md        # This file
```

## Usage

### Creating a Detector

```go
import "actionsum/pkg/detector"

// Use the V2 factory to create a hybrid detector
det, err := detector.NewV2()
if err != nil {
    log.Fatal(err)
}
defer det.Close()
```

### Getting Active Application

```go
// The detector automatically chooses the best method
windowInfo, err := det.GetFocusedWindow()
if err != nil {
    log.Printf("Detection failed: %v", err)
}

fmt.Printf("App: %s\n", windowInfo.AppName)
fmt.Printf("Title: %s\n", windowInfo.WindowTitle)
fmt.Printf("Display: %s\n", windowInfo.DisplayServer)
```

### Checking Detection Method

The hybrid detector tracks which method succeeded:

```go
appInfo, err := hybridDet.GetActiveApp()
fmt.Printf("Detection method: %s\n", appInfo.DetectionMethod)
fmt.Printf("Confidence: %.2f\n", appInfo.Confidence)
```

## Advantages Over V1

### V1 Limitations:
- ❌ Failed on GNOME Wayland for native apps
- ❌ Returned "Unknown" for apps it couldn't detect
- ❌ Single detection method (window-only)
- ❌ No fallback mechanism

### V2 Improvements:
- ✅ Universal detection across all systems
- ✅ Multiple detection methods with intelligent fallback
- ✅ Works for native Wayland apps on GNOME
- ✅ Confidence scoring for detection quality
- ✅ Enhanced with input activity monitoring
- ✅ Better error handling and diagnostics

## Migration from V1

The V2 detector implements the same `window.Detector` interface as V1, so migration is straightforward:

```go
// Old V1 code:
det, err := detector.New()

// New V2 code:
det, err := detector.NewV2()

// All other code remains the same!
```

## Testing

A test program is provided to verify detection:

```bash
# Build test program
go build -o test-v2 ./cmd/test-v2/

# Run test (monitors for 30 seconds)
./test-v2
```

The test program displays:
- Which detection methods are available
- Currently active application
- Window title
- Detection method used
- Confidence score

## Known Limitations

1. **Process detection accuracy**: Process-based detection is less accurate than window detection. It uses heuristics (CPU usage, known GUI apps list) to determine the active application.

2. **Window title for native Wayland apps**: When falling back to process detection, window titles may not be available for native Wayland apps.

3. **Input monitoring**: Requires access to `/proc/bus/input/devices`. Input monitoring is optional and detection works without it.

4. **Performance**: Process scanning adds minimal overhead (~1-5ms per detection). Window detection is still preferred when available.

## Future Enhancements

- [ ] True input device monitoring via `/dev/input/event*`
- [ ] Machine learning for better active app prediction
- [ ] Support for remote desktop sessions
- [ ] Wakatime-style editor plugin integration
- [ ] Browser tab tracking (via extensions)

## Contributing

When adding new detection methods:

1. Implement the `common.Detector` interface
2. Add priority level (higher = preferred)
3. Implement `IsAvailable()` check
4. Add to hybrid detector with appropriate fallback logic

## License

MIT License - See LICENSE file for details
