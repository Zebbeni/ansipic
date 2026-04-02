package ansipx

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"testing"

	"github.com/lucasb-eyer/go-colorful"
)

func TestRenderSolidColorImage(t *testing.T) {
	img := solidImage(4, 4, color.RGBA{R: 128, G: 64, B: 32, A: 255})
	opts := DefaultOptions()
	opts.Width = 2
	opts.Height = 2
	opts.SizeMode = Stretch

	result, err := Render(img, opts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if len(result) == 0 {
		t.Error("Render returned empty string")
	}
	if !strings.Contains(result, "\x1b[") {
		t.Error("output does not contain ANSI escape sequences")
	}
}

func TestRenderPaletteColorPrecision(t *testing.T) {
	paletteHexes := []string{"#000000", "#626262", "#ffffff"}
	paletteColors := make(color.Palette, len(paletteHexes))
	for i, hex := range paletteHexes {
		c, _ := colorful.Hex(hex)
		paletteColors[i] = c
	}

	opts := DefaultOptions()
	opts.Width = 2
	opts.Height = 2
	opts.SizeMode = Stretch
	opts.TrueColor = false
	opts.Palette = paletteColors

	// Test that palette color #626262 (RGB 98,98,98) appears correctly in output
	img := solidImage(4, 4, color.RGBA{R: 98, G: 98, B: 98, A: 255})
	result, err := Render(img, opts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	wantRGB := "98;98;98"
	if !strings.Contains(result, wantRGB) {
		t.Errorf("output does not contain expected RGB %s (#626262)", wantRGB)
		for delta := -3; delta <= 3; delta++ {
			nearby := fmt.Sprintf("%d;%d;%d", 98+delta, 98+delta, 98+delta)
			if strings.Contains(result, nearby) {
				t.Errorf("  found nearby RGB: %s (off by %d)", nearby, delta)
			}
		}
	}
}

func TestRenderAsciiMode(t *testing.T) {
	img := solidImage(4, 4, color.RGBA{R: 200, G: 200, B: 200, A: 255})
	opts := DefaultOptions()
	opts.Width = 2
	opts.Height = 2
	opts.SizeMode = Stretch
	opts.CharacterMode = Ascii
	opts.AsciiCharSet = AsciiAll

	result, err := Render(img, opts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if len(result) == 0 {
		t.Error("Render returned empty string")
	}
}

func TestRenderCustomMode(t *testing.T) {
	img := solidImage(4, 4, color.RGBA{R: 100, G: 100, B: 100, A: 255})
	opts := DefaultOptions()
	opts.Width = 2
	opts.Height = 2
	opts.SizeMode = Stretch
	opts.CharacterMode = Custom
	opts.CustomChars = []rune("ABC")
	opts.SelectionMode = DarkToLight

	result, err := Render(img, opts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if len(result) == 0 {
		t.Error("Render returned empty string")
	}
}

func TestRenderCustomRepeatMode(t *testing.T) {
	img := solidImage(4, 4, color.RGBA{R: 100, G: 100, B: 100, A: 255})
	opts := DefaultOptions()
	opts.Width = 4
	opts.Height = 2
	opts.SizeMode = Stretch
	opts.CharacterMode = Custom
	opts.CustomChars = []rune("AB")
	opts.SelectionMode = Repeat

	result, err := Render(img, opts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if !strings.Contains(result, "A") || !strings.Contains(result, "B") {
		t.Errorf("repeat mode should produce both A and B, got: %s", result)
	}
}

func TestRenderEmptyPaletteError(t *testing.T) {
	img := solidImage(2, 2, color.RGBA{R: 128, G: 128, B: 128, A: 255})
	opts := DefaultOptions()
	opts.TrueColor = false
	opts.Palette = nil

	_, err := Render(img, opts)
	if err == nil {
		t.Error("expected error for palette mode with nil palette")
	}
}

func TestRenderEmptyCustomChars(t *testing.T) {
	img := solidImage(4, 4, color.RGBA{R: 128, G: 128, B: 128, A: 255})
	opts := DefaultOptions()
	opts.Width = 2
	opts.Height = 2
	opts.SizeMode = Stretch
	opts.CharacterMode = Custom
	opts.CustomChars = nil

	result, err := Render(img, opts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if !strings.Contains(result, "Enter at least one custom character") {
		t.Errorf("expected error message for empty custom chars, got: %s", result)
	}
}

func TestAdjust16BitPrecision(t *testing.T) {
	img := image.NewNRGBA64(image.Rect(0, 0, 1, 1))
	img.SetNRGBA64(0, 0, color.NRGBA64{R: 0x62FF, G: 0x62FF, B: 0x62FF, A: 0xFFFF})

	result := adjustBrightness(img, 1)
	col, _ := colorful.MakeColor(result.At(0, 0))
	if col.Hex() != "#646464" {
		t.Errorf("brightness=1 on 16-bit 0x62FF: got %s, want #646464", col.Hex())
	}

	result = adjustContrast(img, 1)
	col, _ = colorful.MakeColor(result.At(0, 0))
	if col.Hex() != "#626262" {
		t.Errorf("contrast=1 on 16-bit 0x62FF: got %s, want #626262", col.Hex())
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()
	if opts.SizeMode != Fit {
		t.Error("default SizeMode should be Fit")
	}
	if opts.CharacterMode != Unicode {
		t.Error("default CharacterMode should be Unicode")
	}
	if opts.UnicodeCharSet != UnicodeHalf {
		t.Error("default UnicodeCharSet should be UnicodeHalf")
	}
	if !opts.TrueColor {
		t.Error("default TrueColor should be true")
	}
}

func solidImage(w, h int, c color.Color) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}
