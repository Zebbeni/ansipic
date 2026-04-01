package ansipic

import (
	"image"

	"github.com/lucasb-eyer/go-colorful"
	"github.com/makeworld-the-better-one/dither/v2"
	"github.com/nfnt/resize"
)

func (m *renderer) processCustom(input image.Image) string {
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

	chars := m.opts.CustomChars
	if len(chars) == 0 {
		return "Enter at least one custom character"
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
			ts.Bold = true // Custom mode always uses bold characters
			// Custom is always FG-only
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

func (m *renderer) charIndex(brightness float64, pixelIndex, charCount int) int {
	switch m.opts.SelectionMode {
	case Repeat:
		return pixelIndex % charCount
	case Random:
		return m.rng.Intn(charCount)
	default:
		return min(int(brightness*float64(charCount)), charCount-1)
	}
}
