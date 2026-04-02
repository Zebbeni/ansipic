package ansipx

import (
	"image"
	"image/color"
	"math"
)

type rgbBounds struct {
	minR, maxR float64
	minG, maxG float64
	minB, maxB float64
}

// imageBounds scans the image and returns the min/max RGB values (0-1 range).
func imageBounds(img image.Image) rgbBounds {
	bounds := img.Bounds()
	b := rgbBounds{
		minR: 1, maxR: 0,
		minG: 1, maxG: 0,
		minB: 1, maxB: 0,
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			rf := float64(r) / 65535.0
			gf := float64(g) / 65535.0
			bf := float64(bl) / 65535.0

			b.minR = math.Min(b.minR, rf)
			b.maxR = math.Max(b.maxR, rf)
			b.minG = math.Min(b.minG, gf)
			b.maxG = math.Max(b.maxG, gf)
			b.minB = math.Min(b.minB, bf)
			b.maxB = math.Max(b.maxB, bf)
		}
	}
	return b
}

// paletteBounds returns the min/max RGB values (0-1 range) from a palette.
func paletteBounds(palette color.Palette) rgbBounds {
	b := rgbBounds{
		minR: 1, maxR: 0,
		minG: 1, maxG: 0,
		minB: 1, maxB: 0,
	}

	for _, c := range palette {
		r, g, bl, _ := c.RGBA()
		rf := float64(r) / 65535.0
		gf := float64(g) / 65535.0
		bf := float64(bl) / 65535.0

		b.minR = math.Min(b.minR, rf)
		b.maxR = math.Max(b.maxR, rf)
		b.minG = math.Min(b.minG, gf)
		b.maxG = math.Max(b.maxG, gf)
		b.minB = math.Min(b.minB, bf)
		b.maxB = math.Max(b.maxB, bf)
	}
	return b
}

// remapChannel linearly maps a value from [srcMin, srcMax] to [dstMin, dstMax].
func remapChannel(val, srcMin, srcMax, dstMin, dstMax float64) float64 {
	srcRange := srcMax - srcMin
	if srcRange == 0 {
		return (dstMin + dstMax) / 2
	}
	t := (val - srcMin) / srcRange
	return dstMin + t*(dstMax-dstMin)
}

// adaptImageToPalette remaps the image's color range to the palette's color range.
func adaptImageToPalette(img image.Image, palette color.Palette) *image.RGBA {
	imgB := imageBounds(img)
	palB := paletteBounds(palette)
	bounds := img.Bounds()

	out := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			rf := float64(r) / 65535.0
			gf := float64(g) / 65535.0
			bf := float64(b) / 65535.0

			nr := remapChannel(rf, imgB.minR, imgB.maxR, palB.minR, palB.maxR)
			ng := remapChannel(gf, imgB.minG, imgB.maxG, palB.minG, palB.maxG)
			nb := remapChannel(bf, imgB.minB, imgB.maxB, palB.minB, palB.maxB)

			// Clamp to [0, 1]
			nr = math.Max(0, math.Min(1, nr))
			ng = math.Max(0, math.Min(1, ng))
			nb = math.Max(0, math.Min(1, nb))

			out.Set(x, y, color.RGBA{
				R: uint8(nr * 255),
				G: uint8(ng * 255),
				B: uint8(nb * 255),
				A: uint8(a >> 8),
			})
		}
	}
	return out
}
