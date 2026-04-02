package ansipx

import (
	"image"
	"image/color"
	"image/gif"
	"os"
	"testing"
	"time"
)

func TestRenderGIFSingleFrame(t *testing.T) {
	path := createTestGIF(t, 1, 10)
	defer os.Remove(path)

	opts := DefaultOptions()
	opts.Width = 2
	opts.Height = 2
	opts.SizeMode = Stretch

	frames, err := RenderGIF(path, opts)
	if err != nil {
		t.Fatalf("RenderGIF failed: %v", err)
	}
	if len(frames) != 1 {
		t.Errorf("expected 1 frame, got %d", len(frames))
	}
	if len(frames[0].Content) == 0 {
		t.Error("frame content is empty")
	}
}

func TestRenderGIFMultiFrame(t *testing.T) {
	path := createTestGIF(t, 5, 20)
	defer os.Remove(path)

	opts := DefaultOptions()
	opts.Width = 2
	opts.Height = 2
	opts.SizeMode = Stretch

	frames, err := RenderGIF(path, opts)
	if err != nil {
		t.Fatalf("RenderGIF failed: %v", err)
	}
	if len(frames) != 5 {
		t.Errorf("expected 5 frames, got %d", len(frames))
	}

	// Check delays (20 centiseconds = 200ms)
	for i, f := range frames {
		if f.Delay != 200*time.Millisecond {
			t.Errorf("frame %d: expected delay 200ms, got %v", i, f.Delay)
		}
	}
}

func TestRenderGIFMaxFrames(t *testing.T) {
	path := createTestGIF(t, 150, 5)
	defer os.Remove(path)

	opts := DefaultOptions()
	opts.Width = 2
	opts.Height = 2
	opts.SizeMode = Stretch

	frames, err := RenderGIF(path, opts)
	if err != nil {
		t.Fatalf("RenderGIF failed: %v", err)
	}
	if len(frames) != 100 {
		t.Errorf("expected 100 frames (capped), got %d", len(frames))
	}
}

func TestRenderGIFZeroDelay(t *testing.T) {
	path := createTestGIF(t, 3, 0) // 0 delay
	defer os.Remove(path)

	opts := DefaultOptions()
	opts.Width = 2
	opts.Height = 2
	opts.SizeMode = Stretch

	frames, err := RenderGIF(path, opts)
	if err != nil {
		t.Fatalf("RenderGIF failed: %v", err)
	}

	for i, f := range frames {
		if f.Delay != 100*time.Millisecond {
			t.Errorf("frame %d: zero delay should default to 100ms, got %v", i, f.Delay)
		}
	}
}

// createTestGIF builds a simple animated GIF with solid-color frames
// and writes it to a temp file. Returns the file path.
func createTestGIF(t *testing.T, numFrames, delayCentiseconds int) string {
	t.Helper()

	f, err := os.CreateTemp("", "test-*.gif")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	g := &gif.GIF{}
	colors := []color.Color{
		color.RGBA{R: 255, A: 255},
		color.RGBA{G: 255, A: 255},
		color.RGBA{B: 255, A: 255},
	}

	for i := 0; i < numFrames; i++ {
		img := image.NewPaletted(image.Rect(0, 0, 4, 4), color.Palette{
			color.Transparent,
			colors[i%len(colors)],
		})
		// Fill with the solid color
		for y := 0; y < 4; y++ {
			for x := 0; x < 4; x++ {
				img.SetColorIndex(x, y, 1)
			}
		}
		g.Image = append(g.Image, img)
		g.Delay = append(g.Delay, delayCentiseconds)
		g.Disposal = append(g.Disposal, gif.DisposalBackground)
	}

	g.Config = image.Config{
		ColorModel: color.Palette{color.Transparent, color.White},
		Width:      4,
		Height:     4,
	}

	if err := gif.EncodeAll(f, g); err != nil {
		t.Fatalf("failed to encode GIF: %v", err)
	}
	f.Close()

	return f.Name()
}
