package ansipx

import (
	"image"
	"math"

	"github.com/lucasb-eyer/go-colorful"
	"github.com/makeworld-the-better-one/dither/v2"
	"github.com/nfnt/resize"
)

var unicodeShadeChars = []rune{' ', '░', '▒', '▓'}

func (m *renderer) processUnicode(input image.Image) string {
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

	rows := make([]string, height)
	rowChars := make([]styledChar, width)
	for y := 0; y < height*2; y += 2 {
		for x := 0; x < width*2; x += 2 {
			r1, _ := colorful.MakeColor(refImg.At(x, y))
			r2, _ := colorful.MakeColor(refImg.At(x+1, y))
			r3, _ := colorful.MakeColor(refImg.At(x, y+1))
			r4, _ := colorful.MakeColor(refImg.At(x+1, y+1))

			if m.opts.OutputAlpha {
				t1 := m.isTransparent(refImg.At(x, y))
				t2 := m.isTransparent(refImg.At(x+1, y))
				t3 := m.isTransparent(refImg.At(x, y+1))
				t4 := m.isTransparent(refImg.At(x+1, y+1))

				if t1 && t2 && t3 && t4 {
					rowChars[x/2] = styledChar{char: AlphaPlaceholder}
					continue
				}
				if t1 || t2 || t3 || t4 {
					char, fg := m.getAlphaBlock(t1, t2, t3, t4, r1, r2, r3, r4)
					rowChars[x/2] = makeStyledChar(string(char), fg, false, colorful.Color{}, m.opts.TextStyle, nil)
					continue
				}
			}

			r, fg, bg := m.getBlock(r1, r2, r3, r4)
			// Unicode always uses FG+BG
			rowChars[x/2] = makeStyledChar(string(r), fg, true, bg, m.opts.TextStyle, m.opts.SolidBackgroundColor)
		}
		rows[y/2] = renderRow(rowChars)
	}
	return m.outputStrings(rows...)
}

func (m *renderer) getBlock(r1, r2, r3, r4 colorful.Color) (r rune, fg, bg colorful.Color) {
	var blockFuncs map[rune]blockFunc
	switch m.opts.UnicodeCharSet {
	case UnicodeFull:
		blockFuncs = m.fullBlockFuncs
	case UnicodeHalf:
		blockFuncs = m.halfBlockFuncs
	case UnicodeQuarter:
		blockFuncs = m.quarterBlockFuncs
	case UnicodeShadeLight:
		blockFuncs = m.shadeLightBlockFuncs
	case UnicodeShadeMed:
		blockFuncs = m.shadeMedBlockFuncs
	case UnicodeShadeHeavy:
		blockFuncs = m.shadeHeavyBlockFuncs
	}

	minDist := 100.0
	for bRune, bFunc := range blockFuncs {
		f, b, dist := bFunc(r1, r2, r3, r4)
		if dist < minDist {
			minDist = dist
			r, fg, bg = bRune, f, b
		}
	}
	return
}

func (m *renderer) avgCol(colors ...colorful.Color) (colorful.Color, float64) {
	rSum, gSum, bSum := 0.0, 0.0, 0.0
	for _, col := range colors {
		rSum += col.R
		gSum += col.G
		bSum += col.B
	}
	count := float64(len(colors))
	avg := colorful.Color{R: rSum / count, G: gSum / count, B: bSum / count}

	if !m.opts.TrueColor {
		paletteAvg := m.opts.Palette.Convert(avg)
		avg, _ = colorful.MakeColor(paletteAvg)
	}

	totalDist := 0.0
	for _, col := range colors {
		totalDist += math.Pow(col.DistanceCIEDE2000(avg), 2)
	}
	return avg, totalDist
}
