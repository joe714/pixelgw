package pixelclient

import (
	"fmt"
	"image"
	"strings"
	"time"

	"github.com/tidbyt/go-libwebp/webp"
)

const (
	// LED character with trailing space for ~square aspect ratio
	// Terminal chars are ~2:1 height:width, so "● " approximates a square pixel
	ledChar = "● " // BLACK CIRCLE U+25CF + space
	// Image dimensions (matching HUB75 matrix)
	imageWidth  = 64
	imageHeight = 32
)

// AnimationFrame holds a rendered frame and its display duration
type AnimationFrame struct {
	Grid  string
	Delay time.Duration
}

// Renderer converts WebP images to terminal output with colored LED circles.
type Renderer struct {
	termWidth  int
	termHeight int
}

// NewRenderer creates a new Renderer with the given terminal dimensions.
func NewRenderer(termWidth, termHeight int) *Renderer {
	return &Renderer{
		termWidth:  termWidth,
		termHeight: termHeight,
	}
}

// UpdateSize updates the terminal dimensions.
func (r *Renderer) UpdateSize(width, height int) {
	r.termWidth = width
	r.termHeight = height
}

// RenderWebP decodes a WebP image and returns a string with colored LED circles.
func (r *Renderer) RenderWebP(data []byte) (string, error) {
	frames, err := r.DecodeAnimation(data)
	if err != nil {
		return "", err
	}
	if len(frames) > 0 {
		return frames[0].Grid, nil
	}
	return "", fmt.Errorf("no frames decoded")
}

// DecodeAnimation decodes a WebP and returns all animation frames with timing.
func (r *Renderer) DecodeAnimation(data []byte) ([]AnimationFrame, error) {
	// Try animated WebP first
	decoder, err := webp.NewAnimationDecoder(data)
	if err == nil {
		defer decoder.Close()
		anim, err := decoder.Decode()
		if err == nil && len(anim.Image) > 0 {
			frames := make([]AnimationFrame, len(anim.Image))
			for i, img := range anim.Image {
				// Compute delay from timestamps (cumulative ms values)
				delay := time.Duration(50) * time.Millisecond // default
				if i < len(anim.Timestamp) {
					if i == 0 {
						delay = time.Duration(anim.Timestamp[0]) * time.Millisecond
					} else {
						delay = time.Duration(anim.Timestamp[i]-anim.Timestamp[i-1]) * time.Millisecond
					}
					if delay <= 0 {
						delay = 50 * time.Millisecond
					}
				}
				frames[i] = AnimationFrame{
					Grid:  r.RenderImage(img),
					Delay: delay,
				}
			}
			return frames, nil
		}
	}

	// Fall back to static WebP
	img, err := webp.DecodeRGBA(data, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	return []AnimationFrame{{
		Grid:  r.RenderImage(img),
		Delay: 0, // Static image, no animation
	}}, nil
}

// RenderImage converts an image to a string with colored LED circles.
// Each pixel is rendered as two half-circle characters to approximate a square aspect ratio.
func (r *Renderer) RenderImage(img image.Image) string {
	bounds := img.Bounds()
	width := bounds.Max.X - bounds.Min.X

	var sb strings.Builder

	// Calculate horizontal padding for centering (2 chars per pixel)
	displayWidth := width * 2
	leftPad := 0
	if r.termWidth > displayWidth {
		leftPad = (r.termWidth - displayWidth) / 2
	}
	padding := strings.Repeat(" ", leftPad)

	// Render each row
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		sb.WriteString(padding)
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			cr, g, b, _ := img.At(x, y).RGBA()
			// Convert from 16-bit to 8-bit color
			r8 := uint8(cr >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)

			// Use 24-bit true color ANSI escape code
			sb.WriteString(fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s\x1b[0m", r8, g8, b8, ledChar))
		}
		if y < bounds.Max.Y-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// RenderPlaceholder returns a placeholder grid when no image is available.
func (r *Renderer) RenderPlaceholder() string {
	var sb strings.Builder

	// Calculate horizontal padding for centering (2 chars per pixel)
	displayWidth := imageWidth * 2
	leftPad := 0
	if r.termWidth > displayWidth {
		leftPad = (r.termWidth - displayWidth) / 2
	}
	padding := strings.Repeat(" ", leftPad)

	// Dark gray color for placeholder
	grayColor := "\x1b[38;2;40;40;40m"
	reset := "\x1b[0m"

	for y := 0; y < imageHeight; y++ {
		sb.WriteString(padding)
		for x := 0; x < imageWidth; x++ {
			sb.WriteString(grayColor + ledChar + reset)
		}
		if y < imageHeight-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// CalculateVerticalPadding returns the number of blank lines needed above the image
// to vertically center it in the terminal.
func (r *Renderer) CalculateVerticalPadding(contentHeight int) int {
	// Reserve space for status bar (1 line)
	availableHeight := r.termHeight - 1
	if availableHeight > contentHeight {
		return (availableHeight - contentHeight) / 2
	}
	return 0
}
