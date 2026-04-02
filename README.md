# ansipx

A Go library for converting images to ANSI art. Renders images as colored terminal text using ASCII characters, Unicode block characters, or custom character sets. Supports static images and animated GIFs.

## Demos

### Unicode Half Blocks (default)

<!-- TODO: Terminal screenshot showing a photo rendered with default Unicode half-block settings -->
![Unicode Half Blocks](https://placeholder.com/unicode-half.png)

### ASCII Characters

<!-- TODO: Terminal screenshot showing the same image rendered with ASCII characters -->
![ASCII Mode](https://placeholder.com/ascii.png)

### Limited Palette with Dithering

<!-- TODO: Terminal screenshot showing an image rendered with a limited color palette and dithering enabled -->
![Palette + Dithering](https://placeholder.com/palette-dither.png)

### Alpha Transparency

<!-- TODO: Terminal screenshot showing a transparent PNG rendered with partial-block alpha edges -->
![Alpha Transparency](https://placeholder.com/alpha.png)

## Install

```
go get github.com/Zebbeni/ansipx
```

## Usage

```go
package main

import (
	"fmt"

	"github.com/Zebbeni/ansipx"
)

func main() {
	// Render with default settings (Unicode half-blocks, true color, fit 50x40)
	result, err := ansipx.RenderFile("photo.png", ansipx.DefaultOptions())
	if err != nil {
		panic(err)
	}
	fmt.Print(result)
}
```

### With a palette

```go
opts := ansipx.DefaultOptions()
opts.TrueColor = false
opts.Palette = color.Palette{
	colorful.MustParseHex("#000000"),
	colorful.MustParseHex("#626262"),
	colorful.MustParseHex("#ffffff"),
}

result, _ := ansipx.RenderFile("photo.png", opts)
fmt.Print(result)
```

### ASCII mode

```go
opts := ansipx.DefaultOptions()
opts.CharacterMode = ansipx.Ascii
opts.AsciiCharSet = ansipx.AsciiAll

result, _ := ansipx.RenderFile("photo.png", opts)
fmt.Print(result)
```

### Custom characters

```go
opts := ansipx.DefaultOptions()
opts.CharacterMode = ansipx.Custom
opts.CustomChars = []rune(".:+#@")
opts.SelectionMode = ansipx.DarkVariance // map by brightness (dark to light)
// opts.SelectionMode = ansipx.LightVariance // map by brightness (light to dark)
// opts.SelectionMode = ansipx.Repeat        // cycle through chars
// opts.SelectionMode = ansipx.Random        // pick at random (use RandomSeed for determinism)

result, _ := ansipx.RenderFile("photo.png", opts)
fmt.Print(result)
```

### From an `image.Image`

```go
img, _, _ := image.Decode(reader)
result, err := ansipx.Render(img, ansipx.DefaultOptions())
```

### Animated GIFs

```go
frames, err := ansipx.RenderGIF("animation.gif", ansipx.DefaultOptions())
if err != nil {
	panic(err)
}
for _, frame := range frames {
	fmt.Print("\033[H") // move cursor to top-left
	fmt.Print(frame.Content)
	time.Sleep(frame.Delay)
}
```

### Alpha transparency

```go
opts := ansipx.DefaultOptions()
opts.OutputAlpha = true          // transparent pixels render as empty space
opts.AlphaThreshold = 0.5       // pixels with alpha below 0.5 are treated as transparent
opts.TrimAlpha = true            // crop transparent borders from output

result, _ := ansipx.RenderFile("sprite.png", opts)
fmt.Print(result)
```

When alpha is enabled with Unicode block characters, partially transparent cells automatically select the best-fit block character (from the full set of quarter, half, three-quarter, and full blocks) to render only the opaque sub-pixels with foreground color while leaving the transparent portion as background.

### Text style

```go
opts := ansipx.DefaultOptions()
opts.TextStyle = ansipx.TextStyle{
	Bold:      true,
	Italic:    true,
	Underline: true,
}

result, _ := ansipx.RenderFile("photo.png", opts)
fmt.Print(result)
```

### Solid background color

```go
bg := colorful.MustParseHex("#1a1a2e")
opts := ansipx.DefaultOptions()
opts.SolidBackgroundColor = &bg

result, _ := ansipx.RenderFile("photo.png", opts)
fmt.Print(result)
```

## Options

### Size

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `SizeMode` | `SizeMode` | `Fit` | `Fit` preserves aspect ratio, `Fill` crops to fill, `Stretch` fills exact dimensions |
| `Width` | `int` | `50` | Output width in characters |
| `Height` | `int` | `40` | Output height in characters |
| `CharRatio` | `float64` | `0.46` | Terminal character width-to-height ratio |

### Characters

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `CharacterMode` | `CharacterMode` | `Unicode` | `Ascii`, `Unicode`, or `Custom` |
| `AsciiCharSet` | `AsciiCharSet` | `AsciiAZ` | `AsciiAZ`, `AsciiNums`, `AsciiSpec`, `AsciiAll` |
| `UnicodeCharSet` | `UnicodeCharSet` | `UnicodeHalf` | `UnicodeFull`, `UnicodeHalf`, `UnicodeQuarter`, `UnicodeShadeLight/Med/Heavy` |
| `CustomChars` | `[]rune` | `nil` | Characters for custom mode |
| `SelectionMode` | `SelectionMode` | `DarkVariance` | How chars map to pixels: `DarkVariance`, `LightVariance`, `Repeat`, `Random` |
| `RandomSeed` | `int64` | `0` | Seed for deterministic Random mode (same seed = same pattern per frame) |
| `VarianceThreshold` | `float64` | `0` | 0-1: below this normalized variance, render a space instead of a character |
| `SolidBackgroundColor` | `*colorful.Color` | `nil` | If set, use this as bg for FG-only character modes |

### Color

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `TrueColor` | `bool` | `true` | Use 24-bit RGB. Set `false` for palette mode |
| `Palette` | `color.Palette` | `nil` | Color palette for palette mode |
| `AdaptToPalette` | `bool` | `false` | Remap image color range to palette range before matching |

### Adjustments

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Brightness` | `int` | `0` | Brightness adjustment (-100 to 100) |
| `Contrast` | `int` | `0` | Contrast adjustment (-100 to 100) |

### Advanced

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Sampling` | `SamplingFunction` | `NearestNeighbor` | `NearestNeighbor`, `Bicubic`, `Bilinear`, `Lanczos2`, `Lanczos3`, `MitchellNetravali` |
| `Dithering` | `bool` | `false` | Enable dithering (palette mode only) |
| `Serpentine` | `bool` | `false` | Use serpentine scanning for error diffusion dithering |
| `DitherMode` | `DitherMode` | `DitherModeMatrix` | `DitherModeMatrix`, `DitherModeBayer`, `DitherModeClusteredDot` |
| `DitherMatrix` | `DitherMatrix` | `FloydSteinberg` | Error diffusion matrix (14 options including `Atkinson`, `Burkes`, `Stucki`, etc.) |
| `BayerSize` | `uint` | `4` | Bayer matrix size (must be power of 2) |
| `DitherStrength` | `float32` | `1.0` | Dithering strength for Bayer and ClusteredDot modes |
| `ClusteredDotMatrix` | `ClusteredDotMatrix` | `ClusteredDot4x4` | Clustered dot matrix (13 options) |

### Text Style

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `TextStyle` | `TextStyle` | `{}` | ANSI text attributes: `Bold`, `Italic`, `Underline`, `Strikethrough` |

### Alpha

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `OutputAlpha` | `bool` | `true` | Preserve transparent pixels as spaces |
| `TrimAlpha` | `bool` | `false` | Crop transparent borders from output |
| `AlphaThreshold` | `float64` | `0` | 0-1: pixels with alpha below this are treated as transparent (0 = only fully transparent, 1 = all transparent) |

## Dependencies

- [go-colorful](https://github.com/lucasb-eyer/go-colorful) -- color space math
- [resize](https://github.com/nfnt/resize) -- image resampling
- [dither](https://github.com/makeworld-the-better-one/dither) -- error diffusion and ordered dithering

No terminal UI dependencies. Output is raw ANSI escape sequences.
