package ansipx

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"os"
	"time"
)

const maxGIFFrames = 100

// Frame holds one rendered animation frame.
type Frame struct {
	Content string
	Delay   time.Duration
}

// RenderGIF decodes an animated GIF, composites frames, and renders each
// to an ANSI art string. Returns up to 100 frames. Single-frame GIFs
// return a 1-element slice.
func RenderGIF(path string, opts Options) ([]Frame, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open GIF %s: %w", path, err)
	}
	defer f.Close()

	g, err := gif.DecodeAll(f)
	if err != nil {
		return nil, fmt.Errorf("could not decode GIF %s: %w", path, err)
	}

	composited := compositeGIFFrames(g, maxGIFFrames)
	r := newRenderer(opts)

	frames := make([]Frame, len(composited))
	for i, img := range composited {
		r.resetRNG()
		content, err := r.process(img)
		if err != nil {
			return nil, fmt.Errorf("error rendering frame %d: %w", i, err)
		}

		delay := time.Duration(100) * time.Millisecond // default for 0-delay frames
		if i < len(g.Delay) && g.Delay[i] > 0 {
			delay = time.Duration(g.Delay[i]) * 10 * time.Millisecond
		}

		frames[i] = Frame{
			Content: content,
			Delay:   delay,
		}
	}

	return frames, nil
}

// compositeGIFFrames builds full-frame images from a GIF's potentially
// partial frames, handling disposal methods correctly.
func compositeGIFFrames(g *gif.GIF, maxFrames int) []image.Image {
	frameCount := len(g.Image)
	if frameCount > maxFrames {
		frameCount = maxFrames
	}

	width, height := g.Config.Width, g.Config.Height
	if width == 0 || height == 0 {
		// Fallback to first frame bounds if config is missing
		b := g.Image[0].Bounds()
		width, height = b.Dx(), b.Dy()
	}

	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	var savedCanvas *image.RGBA // for DisposalPrevious

	result := make([]image.Image, frameCount)

	for i := 0; i < frameCount; i++ {
		// Handle disposal of the previous frame
		if i > 0 {
			prevBounds := g.Image[i-1].Bounds()
			disposal := byte(0)
			if i-1 < len(g.Disposal) {
				disposal = g.Disposal[i-1]
			}

			switch disposal {
			case gif.DisposalBackground:
				// Clear the previous frame's rectangle
				draw.Draw(canvas, prevBounds, image.NewUniform(color.Transparent), image.Point{}, draw.Src)
			case gif.DisposalPrevious:
				// Restore canvas to saved state
				if savedCanvas != nil {
					draw.Draw(canvas, canvas.Bounds(), savedCanvas, image.Point{}, draw.Src)
				}
			// DisposalNone (0) or unspecified: leave canvas as-is
			}
		}

		// Save canvas state before drawing (needed if current frame uses DisposalPrevious)
		if i < len(g.Disposal) && g.Disposal[i] == gif.DisposalPrevious {
			savedCanvas = image.NewRGBA(canvas.Bounds())
			draw.Draw(savedCanvas, canvas.Bounds(), canvas, image.Point{}, draw.Src)
		}

		// Draw the current frame onto the canvas
		draw.Draw(canvas, g.Image[i].Bounds(), g.Image[i], g.Image[i].Bounds().Min, draw.Over)

		// Snapshot the canvas for this frame
		snapshot := image.NewRGBA(canvas.Bounds())
		draw.Draw(snapshot, canvas.Bounds(), canvas, image.Point{}, draw.Src)
		result[i] = snapshot
	}

	return result
}
