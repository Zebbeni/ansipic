package ansipic

import (
	"image"
	"image/color"
	"math"
	"math/rand"

	"github.com/lucasb-eyer/go-colorful"
)

var black = colorful.Color{}

type renderer struct {
	opts                 Options
	rng                  *rand.Rand
	shadeLightBlockFuncs map[rune]blockFunc
	shadeMedBlockFuncs   map[rune]blockFunc
	shadeHeavyBlockFuncs map[rune]blockFunc
	quarterBlockFuncs    map[rune]blockFunc
	halfBlockFuncs       map[rune]blockFunc
	fullBlockFuncs       map[rune]blockFunc
}

func newRenderer(opts Options) *renderer {
	m := &renderer{
		opts: opts,
		rng:  rand.New(rand.NewSource(opts.RandomSeed)),
	}
	m.fullBlockFuncs = m.createFullBlockFuncs()
	m.halfBlockFuncs = m.createHalfBlockFuncs()
	m.quarterBlockFuncs = m.createQuarterBlockFuncs()
	m.shadeLightBlockFuncs = m.createShadeLightFuncs()
	m.shadeMedBlockFuncs = m.createShadeMedFuncs()
	m.shadeHeavyBlockFuncs = m.createShadeHeavyFuncs()
	return m
}

// resetRNG re-seeds the random number generator so each frame
// in a GIF animation produces the same character pattern.
func (m *renderer) resetRNG() {
	m.rng = rand.New(rand.NewSource(m.opts.RandomSeed))
}

// isTransparent checks if a pixel's alpha is below the threshold.
// With threshold 0 (default), only fully transparent pixels are transparent.
func (m *renderer) isTransparent(c color.Color) bool {
	_, _, _, a := c.RGBA()
	alpha := float64(a) / 65535.0
	if m.opts.AlphaThreshold > 0 {
		return alpha < m.opts.AlphaThreshold
	}
	return a == 0
}

// clearTransparentRGB zeros out RGB for pixels whose alpha is at or below the
// threshold. This prevents resampling algorithms (e.g. Lanczos) from bleeding
// color into fully transparent areas.
func (m *renderer) clearTransparentRGB(img image.Image) image.Image {
	bounds := img.Bounds()
	dst := image.NewNRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.At(x, y)
			if m.isTransparent(c) {
				dst.SetNRGBA(x, y, color.NRGBA{A: 0})
			} else {
				dst.Set(x, y, c)
			}
		}
	}
	return dst
}

// alphaBlockChars maps a 4-bit opaque mask to the block character whose
// foreground region covers exactly the opaque sub-pixels.
// Bit layout: bit3=r1(TL), bit2=r2(TR), bit1=r3(BL), bit0=r4(BR).
var alphaBlockChars = [16]rune{
	0b0000: ' ', // all transparent
	0b0001: '▗', // BR
	0b0010: '▖', // BL
	0b0011: '▄', // bottom
	0b0100: '▝', // TR
	0b0101: '▐', // right
	0b0110: '▞', // anti-diagonal
	0b0111: '▟', // BR 3/4
	0b1000: '▘', // TL
	0b1001: '▚', // diagonal
	0b1010: '▌', // left
	0b1011: '▙', // BL 3/4
	0b1100: '▀', // top
	0b1101: '▜', // TR 3/4
	0b1110: '▛', // TL 3/4
	0b1111: '█', // full
}

// getAlphaBlock returns a block character and fg color for a cell with mixed
// transparency. The block's fg region covers the opaque sub-pixels; bg is left
// transparent (hasBg=false).
func (m *renderer) getAlphaBlock(t1, t2, t3, t4 bool, r1, r2, r3, r4 colorful.Color) (rune, colorful.Color) {
	mask := 0
	if !t1 {
		mask |= 0b1000
	}
	if !t2 {
		mask |= 0b0100
	}
	if !t3 {
		mask |= 0b0010
	}
	if !t4 {
		mask |= 0b0001
	}

	var colors []colorful.Color
	if !t1 {
		colors = append(colors, r1)
	}
	if !t2 {
		colors = append(colors, r2)
	}
	if !t3 {
		colors = append(colors, r3)
	}
	if !t4 {
		colors = append(colors, r4)
	}

	fg := m.avgColTrue(colors...)
	if !m.opts.TrueColor && len(m.opts.Palette) > 0 {
		paletteAvg, _ := colorful.MakeColor(m.opts.Palette.Convert(fg))
		fg = paletteAvg
	}

	return alphaBlockChars[mask], fg
}

type blockFunc func(r1, r2, r3, r4 colorful.Color) (colorful.Color, colorful.Color, float64)

func (m *renderer) createQuarterBlockFuncs() map[rune]blockFunc {
	return map[rune]blockFunc{
		'▀': m.calcTop,
		'▐': m.calcRight,
		'▞': m.calcDiagonal,
		'▖': m.calcBotLeft,
		'▘': m.calcTopLeft,
		'▝': m.calcTopRight,
		'▗': m.calcBotRight,
	}
}

func (m *renderer) createHalfBlockFuncs() map[rune]blockFunc {
	return map[rune]blockFunc{
		'▀': m.calcTop,
	}
}

func (m *renderer) createFullBlockFuncs() map[rune]blockFunc {
	return map[rune]blockFunc{
		'█': m.calcFull,
	}
}

func (m *renderer) createShadeLightFuncs() map[rune]blockFunc {
	return map[rune]blockFunc{
		'░': m.calcHeavy,
	}
}

func (m *renderer) createShadeMedFuncs() map[rune]blockFunc {
	return map[rune]blockFunc{
		'▒': m.calcHeavy,
	}
}

func (m *renderer) createShadeHeavyFuncs() map[rune]blockFunc {
	return map[rune]blockFunc{
		'▓': m.calcHeavy,
	}
}

func (m *renderer) getLightDarkPaletted(light, dark colorful.Color) (colorful.Color, colorful.Color) {
	// Work on a copy to avoid mutating m.opts.Palette via append
	colors := make(color.Palette, len(m.opts.Palette))
	copy(colors, m.opts.Palette)

	index := colors.Index(dark)
	paletteDark := colors.Convert(dark)

	paletteMinusDarkest := make(color.Palette, 0, len(colors)-1)
	paletteMinusDarkest = append(paletteMinusDarkest, colors[:index]...)
	paletteMinusDarkest = append(paletteMinusDarkest, colors[index+1:]...)
	paletteLight := paletteMinusDarkest.Convert(light)

	light, _ = colorful.MakeColor(paletteLight)
	dark, _ = colorful.MakeColor(paletteDark)

	lightBlackDist := light.DistanceLuv(black)
	darkBlackDist := dark.DistanceLuv(black)
	if darkBlackDist > lightBlackDist {
		light, dark = dark, light
	}

	return light, dark
}

func (m *renderer) getDarkestPaletted() colorful.Color {
	if m.opts.TrueColor {
		return black
	}
	colors := m.opts.Palette
	darkest := colors.Convert(black)
	darkestConverted, _ := colorful.MakeColor(darkest)
	return darkestConverted
}

func (m *renderer) calcLight(r1, r2, r3, r4 colorful.Color) (colorful.Color, colorful.Color, float64) {
	_, dark := lightDark(r1, r2, r3, r4)
	avg := m.avgColTrue(r1, r2, r3, r4)
	if !m.opts.TrueColor {
		avg, dark = m.getLightDarkPaletted(avg, dark)
	}
	dist := avg.DistanceLuv(black)
	return avg, dark, math.Min(1.0, math.Abs(dist))
}

func (m *renderer) calcMed(r1, r2, r3, r4 colorful.Color) (colorful.Color, colorful.Color, float64) {
	_, dark := lightDark(r1, r2, r3, r4)
	avg := m.avgColTrue(r1, r2, r3, r4)
	if !m.opts.TrueColor {
		avg, dark = m.getLightDarkPaletted(avg, dark)
	}
	dist := avg.DistanceLuv(black)
	return avg, dark, math.Min(1.0, math.Abs(dist-0.5))
}

func (m *renderer) calcHeavy(r1, r2, r3, r4 colorful.Color) (colorful.Color, colorful.Color, float64) {
	_, dark := lightDark(r1, r2, r3, r4)
	avg := m.avgColTrue(r1, r2, r3, r4)
	if !m.opts.TrueColor {
		avg, dark = m.getLightDarkPaletted(avg, dark)
	}
	dist := avg.DistanceLuv(black)
	return avg, dark, math.Min(1.0, math.Abs(dist-1))
}

func (m *renderer) calcFull(r1, r2, r3, r4 colorful.Color) (colorful.Color, colorful.Color, float64) {
	// Full block uses only fg — match avg against the full palette, not the
	// reduced palette that getLightDarkPaletted would produce.
	avg := m.avgColTrue(r1, r2, r3, r4)
	if !m.opts.TrueColor {
		paletteAvg, _ := colorful.MakeColor(m.opts.Palette.Convert(avg))
		avg = paletteAvg
	}
	dist := avg.DistanceLuv(black)
	return avg, avg, math.Min(1.0, math.Abs(dist-1))
}

func (m *renderer) calcTop(r1, r2, r3, r4 colorful.Color) (colorful.Color, colorful.Color, float64) {
	if r1.R == 0 && r1.G == 0 && r1.B == 0 && (r3.R != 0 || r3.G != 0 || r3.B != 0) {
		r1.R = r1.G
	}
	fg, fDist := m.avgCol(r1, r2)
	bg, bDist := m.avgCol(r3, r4)
	return fg, bg, fDist + bDist
}

func (m *renderer) calcRight(r1, r2, r3, r4 colorful.Color) (colorful.Color, colorful.Color, float64) {
	fg, fDist := m.avgCol(r2, r4)
	bg, bDist := m.avgCol(r1, r3)
	return fg, bg, fDist + bDist
}

func (m *renderer) calcDiagonal(r1, r2, r3, r4 colorful.Color) (colorful.Color, colorful.Color, float64) {
	fg, fDist := m.avgCol(r2, r3)
	bg, bDist := m.avgCol(r1, r4)
	return fg, bg, fDist + bDist
}

func (m *renderer) calcBotLeft(r1, r2, r3, r4 colorful.Color) (colorful.Color, colorful.Color, float64) {
	fg, fDist := m.avgCol(r3)
	bg, bDist := m.avgCol(r1, r2, r4)
	return fg, bg, fDist + bDist
}

func (m *renderer) calcTopLeft(r1, r2, r3, r4 colorful.Color) (colorful.Color, colorful.Color, float64) {
	fg, fDist := m.avgCol(r1)
	bg, bDist := m.avgCol(r2, r3, r4)
	return fg, bg, fDist + bDist
}

func (m *renderer) calcTopRight(r1, r2, r3, r4 colorful.Color) (colorful.Color, colorful.Color, float64) {
	fg, fDist := m.avgCol(r2)
	bg, bDist := m.avgCol(r1, r3, r4)
	return fg, bg, fDist + bDist
}

func (m *renderer) calcBotRight(r1, r2, r3, r4 colorful.Color) (colorful.Color, colorful.Color, float64) {
	fg, fDist := m.avgCol(r4)
	bg, bDist := m.avgCol(r1, r2, r3)
	return fg, bg, fDist + bDist
}
