package ansipx

import (
	"image/color"

	"github.com/lucasb-eyer/go-colorful"
	"github.com/makeworld-the-better-one/dither/v2"
	"github.com/nfnt/resize"
)

// SizeMode controls how the image dimensions are calculated.
type SizeMode int

const (
	Fit     SizeMode = iota // Fit within Width x Height preserving aspect ratio
	Stretch                 // Stretch to exactly Width x Height
	Fill                    // Fill Width x Height preserving aspect ratio (may crop)
)

// CharacterMode selects the character set family.
type CharacterMode int

const (
	Ascii   CharacterMode = iota // ASCII characters mapped by brightness
	Unicode                      // Unicode block characters
	Custom                       // User-defined characters
)

// AsciiCharSet selects which ASCII characters to use.
type AsciiCharSet int

const (
	AsciiAZ   AsciiCharSet = iota // Letters only
	AsciiNums                     // Numbers only
	AsciiSpec                     // Special characters only
	AsciiAll                      // All ASCII characters
)

// UnicodeCharSet selects which Unicode block characters to use.
type UnicodeCharSet int

const (
	UnicodeFull       UnicodeCharSet = iota // Full block █
	UnicodeHalf                             // Half blocks ▀▄
	UnicodeQuarter                          // Quarter blocks ▞▟
	UnicodeShadeLight                       // Light shade ░
	UnicodeShadeMed                         // Medium shade ▒
	UnicodeShadeHeavy                       // Heavy shade ▓
)

// SelectionMode controls how custom characters are assigned to pixels.
type SelectionMode int

const (
	DarkVariance  SelectionMode = iota // Map by distance from darkest palette color (or black for true color)
	LightVariance                      // Map by distance from lightest palette color (or white for true color)
	DarkToLight                        // Legacy: same as DarkVariance
	Repeat                             // Cycle through chars sequentially
	Random                             // Pick chars at random
)

// SamplingFunction selects the image resize interpolation method.
type SamplingFunction int

const (
	NearestNeighbor SamplingFunction = iota
	Bicubic
	Bilinear
	Lanczos2
	Lanczos3
	MitchellNetravali
)

// DitherMode selects the dithering algorithm family.
type DitherMode int

const (
	DitherModeMatrix       DitherMode = iota // Error diffusion matrix
	DitherModeBayer                          // Bayer ordered dithering
	DitherModeClusteredDot                   // Clustered dot ordered dithering
)

// DitherMatrix selects the error diffusion matrix for dithering.
type DitherMatrix int

const (
	FloydSteinberg DitherMatrix = iota
	Atkinson
	Burkes
	FalseFloydSteinberg
	JarvisJudiceNinke
	Sierra
	Sierra2
	Sierra3
	SierraLite
	TwoRowSierra
	Sierra2_4A
	Simple2D
	Stucki
	StevenPigeon
)

// ClusteredDotMatrix selects the ordered dither matrix for clustered dot dithering.
type ClusteredDotMatrix int

const (
	ClusteredDot4x4 ClusteredDotMatrix = iota
	ClusteredDot6x6
	ClusteredDot6x6_2
	ClusteredDot6x6_3
	ClusteredDot8x8
	ClusteredDotDiagonal6x6
	ClusteredDotDiagonal8x8
	ClusteredDotDiagonal8x8_2
	ClusteredDotDiagonal8x8_3
	ClusteredDotDiagonal16x16
	ClusteredDotHorizontalLine
	ClusteredDotVerticalLine
	ClusteredDotSpiral5x5
)

// Options configures how an image is rendered to ANSI art.
type Options struct {
	// Size
	SizeMode  SizeMode
	Width     int
	Height    int
	CharRatio float64 // terminal character width-to-height ratio

	// Characters
	CharacterMode        CharacterMode
	AsciiCharSet         AsciiCharSet   // used when CharacterMode == Ascii
	UnicodeCharSet       UnicodeCharSet // used when CharacterMode == Unicode
	CustomChars          []rune         // used when CharacterMode == Custom
	SolidBackgroundColor *colorful.Color // if set, use this as bg for Ascii/Custom FG-only characters
	SelectionMode        SelectionMode  // used when CharacterMode == Custom or Ascii
	RandomSeed           int64          // seed for deterministic Random mode (same seed + position = same char)
	VarianceThreshold    float64        // 0-1: if normalized variance is below this, render a space instead of a character

	// Color
	TrueColor       bool          // true = 24-bit RGB; false = use Palette
	Palette         color.Palette // used when TrueColor == false
	AdaptToPalette  bool          // remap image color range to palette color range before matching

	// Adjustments
	Brightness int // -100..100
	Contrast   int // -100..100

	// Advanced
	Sampling           SamplingFunction
	Dithering          bool
	Serpentine         bool
	DitherMode         DitherMode
	DitherMatrix       DitherMatrix       // used when DitherMode == DitherModeMatrix
	BayerSize          uint               // used when DitherMode == DitherModeBayer (must be power of 2)
	DitherStrength     float32            // strength for Bayer/ClusteredDot (default 1.0)
	ClusteredDotMatrix ClusteredDotMatrix // used when DitherMode == DitherModeClusteredDot

	// Text Style
	TextStyle TextStyle

	// Alpha
	OutputAlpha    bool
	TrimAlpha      bool
	AlphaThreshold float64 // 0-1: pixels with alpha below this are treated as transparent (default 0 = only fully transparent)
}

// DefaultOptions returns sensible defaults matching the ansizalizer TUI defaults.
func DefaultOptions() Options {
	return Options{
		SizeMode:       Fit,
		Width:          50,
		Height:         40,
		CharRatio:      0.46,
		CharacterMode:  Unicode,
		AsciiCharSet:   AsciiAZ,
		UnicodeCharSet: UnicodeHalf,
		SelectionMode:  DarkVariance,
		TrueColor:      true,
		Brightness:     0,
		Contrast:       0,
		Sampling:       NearestNeighbor,
		DitherMode:     DitherModeMatrix,
		DitherMatrix:   FloydSteinberg,
		BayerSize:      4,
		DitherStrength: 1.0,
		OutputAlpha:    true,
	}
}

var samplingFuncMap = map[SamplingFunction]resize.InterpolationFunction{
	NearestNeighbor:   resize.NearestNeighbor,
	Bicubic:           resize.Bicubic,
	Bilinear:          resize.Bilinear,
	Lanczos2:          resize.Lanczos2,
	Lanczos3:          resize.Lanczos3,
	MitchellNetravali: resize.MitchellNetravali,
}

var ditherMatrixMap = map[DitherMatrix]dither.ErrorDiffusionMatrix{
	FloydSteinberg:      dither.FloydSteinberg,
	Atkinson:            dither.Atkinson,
	Burkes:              dither.Burkes,
	FalseFloydSteinberg: dither.FalseFloydSteinberg,
	JarvisJudiceNinke:   dither.JarvisJudiceNinke,
	Sierra:              dither.Sierra,
	Sierra2:             dither.Sierra2,
	Sierra3:             dither.Sierra3,
	SierraLite:          dither.SierraLite,
	TwoRowSierra:        dither.TwoRowSierra,
	Sierra2_4A:          dither.Sierra2_4A,
	Simple2D:            dither.Simple2D,
	Stucki:              dither.Stucki,
	StevenPigeon:        dither.StevenPigeon,
}

func (o Options) resizeFunc() resize.InterpolationFunction {
	if f, ok := samplingFuncMap[o.Sampling]; ok {
		return f
	}
	return resize.NearestNeighbor
}

func (o Options) ditherMatrix() dither.ErrorDiffusionMatrix {
	if m, ok := ditherMatrixMap[o.DitherMatrix]; ok {
		return m
	}
	return dither.FloydSteinberg
}

var clusteredDotMatrixMap = map[ClusteredDotMatrix]dither.OrderedDitherMatrix{
	ClusteredDot4x4:            dither.ClusteredDot4x4,
	ClusteredDot6x6:            dither.ClusteredDot6x6,
	ClusteredDot6x6_2:          dither.ClusteredDot6x6_2,
	ClusteredDot6x6_3:          dither.ClusteredDot6x6_3,
	ClusteredDot8x8:            dither.ClusteredDot8x8,
	ClusteredDotDiagonal6x6:    dither.ClusteredDotDiagonal6x6,
	ClusteredDotDiagonal8x8:    dither.ClusteredDotDiagonal8x8,
	ClusteredDotDiagonal8x8_2:  dither.ClusteredDotDiagonal8x8_2,
	ClusteredDotDiagonal8x8_3:  dither.ClusteredDotDiagonal8x8_3,
	ClusteredDotDiagonal16x16:  dither.ClusteredDotDiagonal16x16,
	ClusteredDotHorizontalLine: dither.ClusteredDotHorizontalLine,
	ClusteredDotVerticalLine:   dither.ClusteredDotVerticalLine,
	ClusteredDotSpiral5x5:      dither.ClusteredDotSpiral5x5,
}

func (o Options) clusteredDotMatrix() dither.OrderedDitherMatrix {
	if m, ok := clusteredDotMatrixMap[o.ClusteredDotMatrix]; ok {
		return m
	}
	return dither.ClusteredDot4x4
}

func isPowerOfTwo(n uint) bool {
	return n > 0 && (n&(n-1)) == 0
}

// applyDither configures and runs the ditherer on the given image.
func (o Options) applyDither(d *dither.Ditherer) {
	strength := o.DitherStrength
	if strength <= 0 {
		strength = 1.0
	}
	switch o.DitherMode {
	case DitherModeBayer:
		bayerSize := o.BayerSize
		if bayerSize == 0 || !isPowerOfTwo(bayerSize) {
			bayerSize = 4
		}
		d.Mapper = dither.Bayer(bayerSize, bayerSize, strength)
	case DitherModeClusteredDot:
		d.Mapper = dither.PixelMapperFromMatrix(o.clusteredDotMatrix(), strength)
	default:
		d.Matrix = dither.ErrorDiffusionStrength(o.ditherMatrix(), strength)
		if o.Serpentine {
			d.Serpentine = true
		}
	}
}
