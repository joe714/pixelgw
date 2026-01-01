package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joe714/pixelgw/internal/pixelclient"
	"github.com/tidbyt/go-libwebp/webp"
)

func main() {
	// Define CLI flags
	deviceName := flag.String("d", "", "Device name from config (bypass menu)")
	deviceFlag := flag.String("device", "", "Device name from config (bypass menu)")
	newDevice := flag.Bool("n", false, "Create new device")
	newDeviceFlag := flag.Bool("new", false, "Create new device")
	server := flag.String("server", "", "Server address (used with --new)")
	deviceID := flag.String("device-id", "", "Custom device UUID (optional, used with --new)")
	configPath := flag.String("c", "", "Path to config file")
	configFlag := flag.String("config", "", "Path to config file")
	headless := flag.Bool("headless", false, "Run in headless mode (print frame info only)")
	verbose := flag.Bool("verbose", false, "Enable verbose/debug output (with --headless)")
	timeout := flag.Int("time", 0, "Auto-disconnect after N seconds (headless mode)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: pixelclient [flags]\n\n")
		fmt.Fprintf(os.Stderr, "A terminal UI client for PixelGateway that displays LED matrix content.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		fmt.Fprintf(os.Stderr, "  -c, --config <path>      Path to config file (default: ~/.pixelclient/config.json)\n")
		fmt.Fprintf(os.Stderr, "  -d, --device <name>      Use named device from config (bypass menu)\n")
		fmt.Fprintf(os.Stderr, "  -n, --new                Create new device (interactive if missing args)\n")
		fmt.Fprintf(os.Stderr, "      --server <ip:port>   Server address (used with --new)\n")
		fmt.Fprintf(os.Stderr, "      --device-id <uuid>   Custom device UUID (optional, used with --new)\n")
		fmt.Fprintf(os.Stderr, "      --headless           Run without TUI, print frame info to stdout\n")
		fmt.Fprintf(os.Stderr, "      --verbose            Enable verbose/debug output (with --headless)\n")
		fmt.Fprintf(os.Stderr, "      --time <seconds>     Auto-disconnect after N seconds (with --headless)\n")
		fmt.Fprintf(os.Stderr, "  -h, --help               Show help\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  pixelclient                                        # Auto-select \"default\" or show menu\n")
		fmt.Fprintf(os.Stderr, "  pixelclient -d living-room                         # Connect using \"living-room\" device\n")
		fmt.Fprintf(os.Stderr, "  pixelclient --new                                  # Create device interactively\n")
		fmt.Fprintf(os.Stderr, "  pixelclient --new -d bedroom --server 10.0.0.5:8080    # Create \"bedroom\" and connect\n")
		fmt.Fprintf(os.Stderr, "  pixelclient -c /path/to/config.json                # Use alternate config file\n")
		fmt.Fprintf(os.Stderr, "  pixelclient --headless -d Test1                    # Headless mode with existing device\n")
		fmt.Fprintf(os.Stderr, "  pixelclient --headless --time 30 -d Test1          # Headless, auto-stop after 30s\n")
	}

	flag.Parse()

	// Merge short and long flags
	if *deviceFlag != "" && *deviceName == "" {
		deviceName = deviceFlag
	}
	if *newDeviceFlag && !*newDevice {
		newDevice = newDeviceFlag
	}
	if *configFlag != "" && *configPath == "" {
		configPath = configFlag
	}

	// Load config
	config, err := pixelclient.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	var initialDevice *pixelclient.Device

	if *newDevice {
		// Create new device mode
		name := *deviceName
		serverAddr := *server

		// Prompt for missing values if not provided
		if name == "" || serverAddr == "" {
			reader := bufio.NewReader(os.Stdin)

			if name == "" {
				fmt.Print("Device name: ")
				name, _ = reader.ReadString('\n')
				name = strings.TrimSpace(name)
			}

			if serverAddr == "" {
				fmt.Print("Server (ip:port): ")
				serverAddr, _ = reader.ReadString('\n')
				serverAddr = strings.TrimSpace(serverAddr)
			}
		}

		if name == "" || serverAddr == "" {
			fmt.Fprintf(os.Stderr, "Error: name and server are required\n")
			os.Exit(1)
		}

		device, err := config.AddDevice(name, serverAddr, *deviceID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if err := config.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Created device \"%s\" with ID %s\n", device.Name, device.ID)
		initialDevice = device

	} else if *deviceName != "" {
		// Use specified device
		device := config.FindDeviceByName(*deviceName)
		if device == nil {
			fmt.Fprintf(os.Stderr, "Error: device \"%s\" not found\n", *deviceName)
			os.Exit(1)
		}
		initialDevice = device

	} else {
		// Check for default device
		if device := config.GetDefaultDevice(); device != nil {
			initialDevice = device
		}
		// Otherwise, show selection menu (initialDevice stays nil)
	}

	// Headless mode
	if *headless {
		if initialDevice == nil {
			fmt.Fprintf(os.Stderr, "Error: --headless requires a device (-d or --new)\n")
			os.Exit(1)
		}
		runHeadless(initialDevice, *timeout, *verbose)
		return
	}

	// Create and run the TUI
	model := pixelclient.NewModel(config, initialDevice)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runHeadless runs the client in headless mode, printing frame info to stdout
func runHeadless(device *pixelclient.Device, timeoutSec int, verbose bool) {
	// Build WebSocket URI
	u := url.URL{
		Scheme:   "ws",
		Host:     device.Server,
		Path:     "/ws",
		RawQuery: "device=" + url.QueryEscape(device.ID),
	}

	fmt.Printf("Device:    %s\n", device.Name)
	fmt.Printf("Device ID: %s\n", device.ID)
	fmt.Printf("URI:       %s\n", u.String())
	fmt.Println()

	// Set up context with optional timeout
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle Ctrl-C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nInterrupted, disconnecting...")
		cancel()
	}()

	// Set up timeout if specified
	if timeoutSec > 0 {
		go func() {
			time.Sleep(time.Duration(timeoutSec) * time.Second)
			fmt.Printf("\nTimeout after %d seconds, disconnecting...\n", timeoutSec)
			cancel()
		}()
	}

	// Connect
	client := pixelclient.NewClient(device)
	if verbose {
		fmt.Println("[verbose] Connecting...")
	} else {
		fmt.Println("Connecting...")
	}
	if err := client.Connect(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	if verbose {
		fmt.Println("[verbose] Connected, entering read loop...")
	} else {
		fmt.Println("Connected. Waiting for frames...")
	}
	fmt.Println()

	frameCount := 0
	for {
		select {
		case data := <-client.Frames():
			frameCount++
			animFrames := countWebPFrames(data)
			if animFrames > 1 {
				fmt.Printf("Frame %d: %s bytes (%d animation frames)\n", frameCount, formatNumber(len(data)), animFrames)
			} else {
				fmt.Printf("Frame %d: %s bytes\n", frameCount, formatNumber(len(data)))
			}
			if verbose {
				fmt.Printf("[verbose] WebP data: first 16 bytes: %x\n", data[:min(16, len(data))])
			}

		case err := <-client.Errors():
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return

		case <-client.Done():
			fmt.Println("Connection closed")
			return

		case <-ctx.Done():
			return
		}
	}
}

// formatNumber formats an integer with thousands separators
// TODO: Replace with golang.org/x/text/message for locale-aware formatting
func formatNumber(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}

	// Insert commas from right to left
	var result strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(c)
	}
	return result.String()
}

// countWebPFrames counts animation frames in a WebP file using libwebp
func countWebPFrames(data []byte) int {
	decoder, err := webp.NewAnimationDecoder(data)
	if err != nil {
		return 1 // Not animated or error
	}
	defer decoder.Close()

	info, err := decoder.GetInfo()
	if err != nil {
		return 1
	}

	if info.FrameCount == 0 {
		return 1
	}
	return info.FrameCount
}
