package ansipic

import (
	"bufio"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
)

// Render converts an image.Image to an ANSI art string using the given options.
func Render(img image.Image, opts Options) (string, error) {
	r := newRenderer(opts)
	return r.process(img)
}

// RenderFile opens an image file, decodes it, and renders it to an ANSI art string.
func RenderFile(path string, opts Options) (string, error) {
	if path == "" {
		return "", fmt.Errorf("no image file path provided")
	}

	imgFile, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("could not open image %s: %w", path, err)
	}
	defer imgFile.Close()

	img, _, err := image.Decode(bufio.NewReader(imgFile))
	if err != nil {
		return "", fmt.Errorf("could not decode image %s: %w", path, err)
	}

	return Render(img, opts)
}

func (m *renderer) process(input image.Image) (string, error) {
	if !m.opts.TrueColor && len(m.opts.Palette) == 0 {
		return "", fmt.Errorf("palette mode requires a non-empty palette")
	}

	switch m.opts.CharacterMode {
	case Ascii:
		return m.processAscii(input), nil
	case Unicode:
		return m.processUnicode(input), nil
	case Custom:
		return m.processCustom(input), nil
	}
	return "", fmt.Errorf("unknown character mode: %d", m.opts.CharacterMode)
}
