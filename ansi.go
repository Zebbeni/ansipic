package ansipx

import (
	"bufio"
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/lucasb-eyer/go-colorful"
)

const (
	// AlphaPlaceholder marks transparent pixels in the output.
	AlphaPlaceholder string = " "
)

func ansiFg(c colorful.Color) string {
	r := uint8(c.R*255.0 + 0.5)
	g := uint8(c.G*255.0 + 0.5)
	b := uint8(c.B*255.0 + 0.5)
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b)
}

func ansiBg(c colorful.Color) string {
	r := uint8(c.R*255.0 + 0.5)
	g := uint8(c.G*255.0 + 0.5)
	b := uint8(c.B*255.0 + 0.5)
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b)
}

const ansiReset = "\x1b[0m"

// styledChar holds a character and its styling info for deferred rendering.
type styledChar struct {
	char          string
	fg            colorful.Color
	bg            colorful.Color
	hasBg         bool
	bold          bool
	italic        bool
	underline     bool
	strikethrough bool
}

// TextStyle groups the ANSI text attributes that can be applied to characters.
type TextStyle struct {
	Bold          bool
	Italic        bool
	Underline     bool
	Strikethrough bool
}

func makeStyledChar(char string, fg colorful.Color, useBg bool, bg colorful.Color, ts TextStyle, solidBg *colorful.Color) styledChar {
	sc := styledChar{
		char: char, fg: fg,
		bold: ts.Bold, italic: ts.Italic,
		underline: ts.Underline, strikethrough: ts.Strikethrough,
	}
	if useBg {
		sc.bg = bg
		sc.hasBg = true
	} else if solidBg != nil {
		sc.bg = *solidBg
		sc.hasBg = true
	}
	return sc
}

func (sc styledChar) sameStyle(other styledChar) bool {
	return sc.fg == other.fg && sc.bg == other.bg && sc.hasBg == other.hasBg &&
		sc.bold == other.bold && sc.italic == other.italic &&
		sc.underline == other.underline && sc.strikethrough == other.strikethrough
}

// renderRowSolidBg renders a row where all characters share the same background.
// The bg is emitted once at the start; only fg and text attribute changes are tracked.
func renderRowSolidBg(chars []styledChar, bg colorful.Color) string {
	if len(chars) == 0 {
		return ""
	}

	var b strings.Builder
	b.Grow(len(chars) * 4)

	// Set bg once for the entire row
	b.WriteString(ansiBg(bg))

	var curFg colorful.Color
	var cur styledChar // tracks current text attributes
	fgSet := false

	for _, sc := range chars {
		if sc.char == AlphaPlaceholder && !sc.hasBg {
			b.WriteString(ansiReset)
			b.WriteString(sc.char)
			b.WriteString(ansiBg(bg))
			fgSet = false
			cur = styledChar{}
			continue
		}

		if !fgSet || sc.fg != curFg {
			b.WriteString(ansiFg(sc.fg))
			curFg = sc.fg
			fgSet = true
		}
		if !sc.sameAttrs(cur) {
			// Any attribute turning off requires a reset
			if cur.hasAnyAttr() && !sc.sameAttrs(cur) {
				b.WriteString(ansiReset)
				b.WriteString(ansiBg(bg))
				b.WriteString(ansiFg(sc.fg))
				cur = styledChar{}
			}
			// Apply any attributes that need to turn on
			if sc.bold && !cur.bold {
				b.WriteString("\x1b[1m")
			}
			if sc.italic && !cur.italic {
				b.WriteString("\x1b[3m")
			}
			if sc.underline && !cur.underline {
				b.WriteString("\x1b[4m")
			}
			if sc.strikethrough && !cur.strikethrough {
				b.WriteString("\x1b[9m")
			}
			cur.bold = sc.bold
			cur.italic = sc.italic
			cur.underline = sc.underline
			cur.strikethrough = sc.strikethrough
		}
		b.WriteString(sc.char)
	}
	b.WriteString(ansiReset)
	return b.String()
}

// renderRow takes a slice of styledChars and produces an optimized ANSI string,
// tracking fg, bg, and text attributes so only changed attributes are emitted.
func renderRow(chars []styledChar) string {
	if len(chars) == 0 {
		return ""
	}

	var b strings.Builder
	b.Grow(len(chars) * 6)

	var cur styledChar
	active := false // whether any style is currently set

	for _, sc := range chars {
		if sc.char == AlphaPlaceholder && !sc.hasBg {
			if active {
				b.WriteString(ansiReset)
				active = false
				cur = styledChar{}
			}
			b.WriteString(sc.char)
			continue
		}

		if !active {
			b.WriteString(styleEscape(sc))
			cur = sc
			active = true
		} else {
			// Need reset when any attribute turns off or bg disappears
			needsReset := (cur.hasAnyAttr() && !cur.sameAttrs(sc)) || (cur.hasBg && !sc.hasBg)
			if needsReset {
				b.WriteString(ansiReset)
				b.WriteString(styleEscape(sc))
				cur = sc
			} else {
				if sc.fg != cur.fg {
					b.WriteString(ansiFg(sc.fg))
					cur.fg = sc.fg
				}
				if sc.hasBg && sc.bg != cur.bg {
					b.WriteString(ansiBg(sc.bg))
					cur.bg = sc.bg
					cur.hasBg = true
				}
				if sc.bold && !cur.bold {
					b.WriteString("\x1b[1m")
					cur.bold = true
				}
				if sc.italic && !cur.italic {
					b.WriteString("\x1b[3m")
					cur.italic = true
				}
				if sc.underline && !cur.underline {
					b.WriteString("\x1b[4m")
					cur.underline = true
				}
				if sc.strikethrough && !cur.strikethrough {
					b.WriteString("\x1b[9m")
					cur.strikethrough = true
				}
			}
		}
		b.WriteString(sc.char)
	}
	if active {
		b.WriteString(ansiReset)
	}
	return b.String()
}

func styleEscape(sc styledChar) string {
	s := ansiFg(sc.fg)
	if sc.hasBg {
		s += ansiBg(sc.bg)
	}
	if sc.bold {
		s += "\x1b[1m"
	}
	if sc.italic {
		s += "\x1b[3m"
	}
	if sc.underline {
		s += "\x1b[4m"
	}
	if sc.strikethrough {
		s += "\x1b[9m"
	}
	return s
}

// hasAnyAttr returns true if any text attribute (bold/italic/underline/strikethrough) is set.
func (sc styledChar) hasAnyAttr() bool {
	return sc.bold || sc.italic || sc.underline || sc.strikethrough
}

// sameAttrs returns true if the text attributes match.
func (sc styledChar) sameAttrs(other styledChar) bool {
	return sc.bold == other.bold && sc.italic == other.italic &&
		sc.underline == other.underline && sc.strikethrough == other.strikethrough
}

// renderChar is kept for backward compatibility but uses the new system.
func renderChar(char string, fg colorful.Color, useBg bool, bg colorful.Color, ts TextStyle, solidBg *colorful.Color) string {
	sc := makeStyledChar(char, fg, useBg, bg, ts, solidBg)
	return styleEscape(sc) + char + ansiReset
}

func (m *renderer) outputStrings(rows ...string) string {
	content := ""
	if m.opts.OutputAlpha && m.opts.TrimAlpha {
		leftWhitespaceRE := `(^\s+)`
		re := regexp.MustCompile(leftWhitespaceRE)
		leftTrimAmount := math.MaxInt
		contentAlpha := strings.Join(rows, "\n")
		reader := strings.NewReader(contentAlpha)
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			leftWhitespaceMatch := re.FindStringSubmatch(scanner.Text())
			if leftWhitespaceMatch != nil && leftTrimAmount > len(leftWhitespaceMatch[0]) {
				leftTrimAmount = len(leftWhitespaceMatch[0])
			}
			if leftTrimAmount == 0 {
				break
			}
		}
		if leftTrimAmount == math.MaxInt {
			leftTrimAmount = 0
		}
		blankLine := ([]string)(nil)
		blankLineRE := `(^\s+$)`
		re = regexp.MustCompile(blankLineRE)
		imageTop := true
		reader = strings.NewReader(contentAlpha)
		scanner = bufio.NewScanner(reader)
		for scanner.Scan() {
			thisLine := scanner.Text()
			if leftTrimAmount > 0 && len(thisLine) >= leftTrimAmount {
				// Only trim if the line actually starts with spaces (not ANSI escapes)
				allSpaces := true
				for i := 0; i < leftTrimAmount; i++ {
					if thisLine[i] != ' ' {
						allSpaces = false
						break
					}
				}
				if allSpaces {
					thisLine = thisLine[leftTrimAmount:]
				}
			}
			if imageTop {
				blankLine = re.FindStringSubmatch(thisLine)
			}
			if blankLine == nil {
				imageTop = false
			}
			if !imageTop {
				content += strings.TrimRight(thisLine, " ") + "\n"
			}
		}
		content = strings.TrimRight(content, "\n")
	} else {
		content += strings.Join(rows, "\n")
	}
	return content
}
