package ansipic

import (
	"image"
	"math"

	"github.com/lucasb-eyer/go-colorful"
	"github.com/makeworld-the-better-one/dither/v2"
	"github.com/nfnt/resize"
)

var asciiChars = []rune(" `.-':_,^=;><+!rc*/z?sLTv)J7(|Fi{C}fI31tlu[neoZ5Yxjya]2ESwqkP6h9d4VpOGbUAKXHm8RD#$Bg0MNWQ%&@")
var asciiAZChars = []rune(" rczsLTvJFiCfItluneoZYxjyaESwqkPhdVpOGbUAKXHmRDBgMNWQ")
var asciiNumChars = []rune(" 7315269480")
var asciiSpecChars = []rune(" `.-':_,^=;><+!*/?)(|{}[]#$%&@")

func (m *renderer) processAscii(input image.Image) string {
	imgW, imgH := float32(input.Bounds().Dx()), float32(input.Bounds().Dy())

	width, height := m.opts.Width, m.opts.Height
	charRatio := m.opts.CharRatio
	if m.opts.SizeMode == Fit {
		fitHeight := float32(width) * (imgH / imgW) * float32(charRatio)
		fitWidth := (float32(height) * (imgW / imgH)) / float32(charRatio)
		if fitHeight > float32(height) {
			width = int(fitWidth)
		} else {
			height = int(fitHeight)
		}
	} else if m.opts.SizeMode == Fill {
		fillHeight := float32(width) * (imgH / imgW) * float32(charRatio)
		fillWidth := (float32(height) * (imgW / imgH)) / float32(charRatio)
		if fillHeight < float32(height) {
			width = int(fillWidth)
		} else {
			height = int(fillHeight)
		}
	}

	resizeFunc := m.opts.resizeFunc()
	refImg := resize.Resize(uint(width)*2, uint(height)*2, input, resizeFunc)
	refImg = m.clearTransparentRGB(refImg)
	refImg = adjustBrightness(refImg, m.opts.Brightness)
	refImg = adjustContrast(refImg, m.opts.Contrast)

	isPaletted := !m.opts.TrueColor

	if m.opts.AdaptToPalette && isPaletted {
		refImg = adaptImageToPalette(refImg, m.opts.Palette)
	}

	if m.opts.Dithering && isPaletted {
		ditherer := dither.NewDitherer(m.opts.Palette)
		m.opts.applyDither(ditherer)
		refImg = ditherer.Dither(refImg)
	}

	var chars []rune
	switch m.opts.AsciiCharSet {
	case AsciiAZ:
		chars = asciiAZChars
	case AsciiNums:
		chars = asciiNumChars
	case AsciiSpec:
		chars = asciiSpecChars
	case AsciiAll:
		chars = asciiChars
	}

	// Precompute variance range for DarkVariance/LightVariance modes
	vr := m.computeVarianceRange(refImg, width, height, isPaletted)

	pixelCount := 0
	rows := make([]string, height)
	rowChars := make([]styledChar, width)

	for y := 0; y < height*2; y += 2 {
		for x := 0; x < width*2; x += 2 {
			r1, _ := colorful.MakeColor(refImg.At(x, y))
			r2, _ := colorful.MakeColor(refImg.At(x+1, y))
			r3, _ := colorful.MakeColor(refImg.At(x, y+1))
			r4, _ := colorful.MakeColor(refImg.At(x+1, y+1))

			if m.opts.OutputAlpha && (m.isTransparent(refImg.At(x, y)) || m.isTransparent(refImg.At(x+1, y)) || m.isTransparent(refImg.At(x, y+1)) || m.isTransparent(refImg.At(x+1, y+1))) {
				rowChars[x/2] = styledChar{char: AlphaPlaceholder}
				continue
			}

			ts := m.opts.TextStyle
			ts.Bold = true // ASCII mode always uses bold characters
			// ASCII is always FG-only
			fg := m.avgColTrue(r1, r2, r3, r4)
			if isPaletted {
				fg, _ = colorful.MakeColor(m.opts.Palette.Convert(fg))
			}
			brightness := vr.normalize(fg)
			thresh := m.opts.VarianceThreshold
			isVarianceMode := m.opts.SelectionMode == DarkVariance || m.opts.SelectionMode == LightVariance || m.opts.SelectionMode == DarkToLight
			// If variance is below threshold, render a space
			if thresh > 0 && isVarianceMode && brightness < thresh {
				rowChars[x/2] = makeStyledChar(" ", fg, false, colorful.Color{}, ts, m.opts.SolidBackgroundColor)
			} else {
				// Remap brightness from [threshold, 1] → [0, 1] so full char set is used
				mapped := brightness
				if thresh > 0 && isVarianceMode && thresh < 1 {
					mapped = (brightness - thresh) / (1 - thresh)
				}
				index := m.charIndex(mapped, pixelCount, len(chars))
				rowChars[x/2] = makeStyledChar(string(chars[index]), fg, false, colorful.Color{}, ts, m.opts.SolidBackgroundColor)
			}
			pixelCount++
		}
		if m.opts.SolidBackgroundColor != nil {
			rows[y/2] = renderRowSolidBg(rowChars, *m.opts.SolidBackgroundColor)
		} else {
			rows[y/2] = renderRow(rowChars)
		}
	}
	return m.outputStrings(rows...)
}

// varianceRange holds the reference color and min/max distances found in the image
// for normalizing per-cell variance to the full character set range.
type varianceRange struct {
	ref      colorful.Color
	minDist  float64
	maxDist  float64
}

// computeVarianceRange scans the resized image to find the min and max distance
// from the reference color (darkest or lightest palette color), so per-cell
// variance can be normalized to use the full character set.
func (m *renderer) computeVarianceRange(img image.Image, width, height int, isPaletted bool) varianceRange {
	ref := m.varianceRef()
	minDist := math.MaxFloat64
	maxDist := 0.0

	for y := 0; y < height*2; y++ {
		for x := 0; x < width*2; x++ {
			c, ok := colorful.MakeColor(img.At(x, y))
			if !ok {
				continue
			}
			if isPaletted {
				c, _ = colorful.MakeColor(m.opts.Palette.Convert(c))
			}
			d := c.DistanceLuv(ref)
			if d < minDist {
				minDist = d
			}
			if d > maxDist {
				maxDist = d
			}
		}
	}

	if minDist >= maxDist {
		minDist = 0
		maxDist = 1
	}

	return varianceRange{ref: ref, minDist: minDist, maxDist: maxDist}
}

// varianceRef returns the reference color for the current selection mode.
func (m *renderer) varianceRef() colorful.Color {
	switch m.opts.SelectionMode {
	case LightVariance:
		if m.opts.TrueColor || len(m.opts.Palette) == 0 {
			return colorful.Color{R: 1, G: 1, B: 1}
		}
		white := colorful.Color{R: 1, G: 1, B: 1}
		lightest, _ := colorful.MakeColor(m.opts.Palette.Convert(white))
		return lightest
	default: // DarkVariance, DarkToLight
		return m.getDarkestPaletted()
	}
}

// normalizedVariance returns 0-1 for a color based on the precomputed range.
func (vr varianceRange) normalize(c colorful.Color) float64 {
	d := c.DistanceLuv(vr.ref)
	return math.Min(1.0, math.Max(0.0, (d-vr.minDist)/(vr.maxDist-vr.minDist)))
}

func (m *renderer) avgColTrue(colors ...colorful.Color) colorful.Color {
	rSum, gSum, bSum := 0.0, 0.0, 0.0
	for _, col := range colors {
		rSum += col.R
		gSum += col.G
		bSum += col.B
	}
	count := float64(len(colors))
	return colorful.Color{R: rSum / count, G: gSum / count, B: bSum / count}
}

func lightDark(c ...colorful.Color) (light, dark colorful.Color) {
	mostLight, mostDark := 0.0, 1.0
	for _, col := range c {
		_, _, l := col.Hsl()
		if l < mostDark {
			mostDark = l
			dark = col
		}
		if l > mostLight {
			mostLight = l
			light = col
		}
	}
	return
}
