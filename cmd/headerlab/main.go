// Package main renders header-design experiments for the kan TUI.
package main

import (
	"flag"
	"fmt"
	"image/color"
	"math"
	"strings"

	"charm.land/lipgloss/v2"
)

// defaultPreviewWidth defines the default render width for header previews.
const defaultPreviewWidth = 108

// minPreviewWidth defines the minimum supported preview width.
const minPreviewWidth = 72

// maxPreviewWidth defines the maximum supported preview width.
const maxPreviewWidth = 140

// samplePath stores a representative path value for previews.
const samplePath = "pf-gk-a-agent-updated"

// previewPalette defines the color palette for header previews.
type previewPalette struct {
	fg       color.Color
	accent   color.Color
	muted    color.Color
	dim      color.Color
	surface  color.Color
	surface2 color.Color
}

// headerVariant describes one previewable header style.
type headerVariant struct {
	ID          int
	Name        string
	Inspiration string
	Render      func(previewPalette, int) string
}

// main runs the header preview playground.
func main() {
	width := flag.Int("width", defaultPreviewWidth, "preview width")
	flag.Parse()

	renderWidth := clamp(*width, minPreviewWidth, maxPreviewWidth)
	palette := buildPalette()
	variants := buildVariants()
	fmt.Println(renderSheet(palette, variants, renderWidth))
}

// buildPalette returns the default kan-like color palette for previews.
func buildPalette() previewPalette {
	return previewPalette{
		fg:       lipgloss.Color("252"),
		accent:   lipgloss.Color("62"),
		muted:    lipgloss.Color("241"),
		dim:      lipgloss.Color("239"),
		surface:  lipgloss.Color("236"),
		surface2: lipgloss.Color("235"),
	}
}

// buildVariants returns ten header design candidates with path-only context.
func buildVariants() []headerVariant {
	return []headerVariant{
		{
			ID:          1,
			Name:        "KAN Pixel Blocks",
			Inspiration: "box-cell logo mosaic",
			Render: func(p previewPalette, width int) string {
				k := []string{"10001", "10010", "10100", "11000", "10100", "10010", "10001"}
				a := []string{"01110", "10001", "10001", "11111", "10001", "10001", "10001"}
				n := []string{"10001", "11001", "10101", "10011", "10001", "10001", "10001"}
				pixelColors := []color.Color{
					lipgloss.Color("50"),
					lipgloss.Color("86"),
					lipgloss.Color("121"),
					lipgloss.Color("156"),
					lipgloss.Color("190"),
				}
				logoRows := make([]string, 0, len(k))
				for row := 0; row < len(k); row++ {
					bits := k[row] + "0" + a[row] + "0" + n[row]
					var b strings.Builder
					for col, bit := range bits {
						if bit != '1' {
							b.WriteString("  ")
							continue
						}
						cell := lipgloss.NewStyle().Foreground(pixelColors[(row+col)%len(pixelColors)]).Render("██")
						b.WriteString(cell)
					}
					logoRows = append(logoRows, b.String())
				}
				path := lipgloss.NewStyle().Foreground(p.muted).Render(truncate("path: "+samplePath, width-4))
				box := lipgloss.NewStyle().
					Border(lipgloss.NormalBorder()).
					BorderForeground(p.accent).
					Padding(0, 1).
					Width(width)
				return box.Render(strings.Join([]string{strings.Join(logoRows, "\n"), "", path}, "\n"))
			},
		},
		{
			ID:          2,
			Name:        "KAN Stair-Step Chips",
			Inspiration: "diagonal chip motif",
			Render: func(p previewPalette, width int) string {
				chipA := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Background(lipgloss.Color("171")).Padding(0, 1)
				chipB := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Background(lipgloss.Color("99")).Padding(0, 1)
				steps := []string{
					chipA.Render("KAN"),
					"  " + chipB.Render("KAN"),
					"    " + chipA.Render("KAN"),
					"      " + chipB.Render("KAN"),
					"        " + chipA.Render("KAN"),
				}
				path := lipgloss.NewStyle().Foreground(p.muted).Render(truncate("path: "+samplePath, width))
				return strings.Join([]string{strings.Join(steps, "\n"), "", path}, "\n")
			},
		},
		{
			ID:          3,
			Name:        "Boxed Minimal Wordmark",
			Inspiration: "left-anchored baseline + boxed mark",
			Render: func(p previewPalette, width int) string {
				title := lipgloss.NewStyle().
					Border(lipgloss.NormalBorder()).
					BorderForeground(p.dim).
					Padding(0, 1).
					Bold(true).
					Foreground(p.fg).
					Render("KAN")
				rule := lipgloss.NewStyle().Foreground(p.accent).Render(strings.Repeat("─", max(8, width)))
				path := lipgloss.NewStyle().Foreground(p.muted).Render(truncate("path: "+samplePath, width))
				return strings.Join([]string{title, rule, path}, "\n")
			},
		},
		{
			ID:          4,
			Name:        "Gradient Rail Box",
			Inspiration: "merge of boxed mark + left rail",
			Render: func(p previewPalette, width int) string {
				logoBox := lipgloss.NewStyle().
					Border(lipgloss.NormalBorder()).
					BorderForeground(p.dim).
					Padding(0, 2).
					Render(lipgloss.NewStyle().Bold(true).Foreground(p.fg).Render("KAN"))
				ramp := []string{
					"#5A56E0",
					"#4E68F2",
					"#417EF9",
					"#359BEF",
					"#35B9DD",
					"#45D8C0",
					"#66E99A",
					"#98F26D",
					"#C7F45A",
				}
				logoLines := strings.Split(logoBox, "\n")
				frameWidth := max(width, lipgloss.Width(logoBox)+4)
				innerWidth := max(1, frameWidth-2)
				top := gradientBar(frameWidth, ramp, "█")
				sideBar := lipgloss.NewStyle().Foreground(lipgloss.Color("#8D95B5")).Render("█")
				middle := make([]string, 0, len(logoLines))
				for _, line := range logoLines {
					centered := lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center).Render(line)
					middle = append(middle, sideBar+centered+sideBar)
				}
				block := strings.Join(append([]string{top}, append(middle, top)...), "\n")
				path := lipgloss.NewStyle().Foreground(p.muted).Render(truncate("path: "+samplePath, width))
				div := lipgloss.NewStyle().Foreground(p.dim).Render(strings.Repeat("─", max(8, width)))
				return strings.Join([]string{block, path, div}, "\n")
			},
		},
		{
			ID:          5,
			Name:        "Solid Brand Banner",
			Inspiration: "high-contrast title stripe",
			Render: func(p previewPalette, width int) string {
				banner := lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color("230")).
					Background(p.accent).
					Align(lipgloss.Center).
					Width(width).
					Render("KAN")
				path := lipgloss.NewStyle().Foreground(p.muted).Render(truncate("path: "+samplePath, width))
				return strings.Join([]string{banner, path}, "\n")
			},
		},
		{
			ID:          6,
			Name:        "Inset Label Plate",
			Inspiration: "minimal branded plate",
			Render: func(p previewPalette, width int) string {
				plate := lipgloss.NewStyle().
					Background(p.surface2).
					Foreground(p.fg).
					Bold(true).
					Padding(0, 2).
					Render("kan")
				path := lipgloss.NewStyle().Foreground(p.muted).Render(truncate("path: "+samplePath, width))
				return strings.Join([]string{plate, path}, "\n")
			},
		},
		{
			ID:          7,
			Name:        "ASCII Monogram",
			Inspiration: "terminal-native letter art",
			Render: func(p previewPalette, width int) string {
				logo := []string{
					"K   K   A   N   N",
					"K K K  A A  N N N",
					"K   K AAAAA N   N",
				}
				rendered := lipgloss.NewStyle().Bold(true).Foreground(p.fg).Render(strings.Join(logo, "\n"))
				path := lipgloss.NewStyle().Foreground(p.muted).Render(truncate("path: "+samplePath, width))
				return strings.Join([]string{rendered, path}, "\n")
			},
		},
		{
			ID:          8,
			Name:        "Top+Bottom Frame",
			Inspiration: "simple framed masthead",
			Render: func(p previewPalette, width int) string {
				rule := lipgloss.NewStyle().Foreground(p.accent).Render(strings.Repeat("=", max(8, width)))
				title := lipgloss.NewStyle().Bold(true).Foreground(p.fg).Align(lipgloss.Center).Width(width).Render("kan")
				path := lipgloss.NewStyle().Foreground(p.muted).Render(truncate("path: "+samplePath, width))
				return strings.Join([]string{rule, title, rule, path}, "\n")
			},
		},
		{
			ID:          9,
			Name:        "Split Color Bar",
			Inspiration: "two-tone logo plate",
			Render: func(p previewPalette, width int) string {
				leftWidth := width / 2
				rightWidth := width - leftWidth
				left := lipgloss.NewStyle().
					Background(p.surface2).
					Bold(true).
					Foreground(p.fg).
					Align(lipgloss.Center).
					Width(leftWidth).
					Render("kan")
				right := lipgloss.NewStyle().
					Background(p.surface).
					Width(rightWidth).
					Render("")
				path := lipgloss.NewStyle().Foreground(p.muted).Render(truncate("path: "+samplePath, width))
				return strings.Join([]string{lipgloss.JoinHorizontal(lipgloss.Top, left, right), path}, "\n")
			},
		},
		{
			ID:          10,
			Name:        "Shadow Wordmark",
			Inspiration: "subtle duplicate ghost text",
			Render: func(p previewPalette, width int) string {
				top := lipgloss.NewStyle().Bold(true).Foreground(p.fg).Render("kan")
				shadow := lipgloss.NewStyle().Foreground(p.dim).Render("  kan")
				path := lipgloss.NewStyle().Foreground(p.muted).Render(truncate("path: "+samplePath, width))
				return strings.Join([]string{top, shadow, path}, "\n")
			},
		},
	}
}

// renderSheet renders all variants into one terminal-friendly output.
func renderSheet(p previewPalette, variants []headerVariant, width int) string {
	title := lipgloss.NewStyle().Bold(true).Foreground(p.accent).Render("kan header design playground")
	subtitle := lipgloss.NewStyle().
		Foreground(p.dim).
		Render(fmt.Sprintf("width=%d  path-only header context", width))
	sections := []string{title, subtitle}
	for _, variant := range variants {
		sections = append(sections, renderVariantCard(p, variant, width))
	}
	return strings.Join(sections, "\n\n")
}

// renderVariantCard renders one bordered variant preview card.
func renderVariantCard(p previewPalette, variant headerVariant, width int) string {
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(p.dim).
		Padding(0, 1).
		Width(width)
	label := lipgloss.NewStyle().
		Bold(true).
		Foreground(p.accent).
		Render(fmt.Sprintf("%02d. %s", variant.ID, variant.Name))
	source := lipgloss.NewStyle().
		Foreground(p.dim).
		Render("inspiration: " + variant.Inspiration)
	body := variant.Render(p, max(24, width-4))
	return cardStyle.Render(strings.Join([]string{label, source, "", body}, "\n"))
}

// gradientBar renders one horizontal color bar with a left-to-right gradient.
func gradientBar(width int, ramp []string, glyph string) string {
	if width <= 0 || len(ramp) == 0 {
		return ""
	}
	if glyph == "" {
		glyph = " "
	}
	var b strings.Builder
	for col := 0; col < width; col++ {
		t := 0.0
		if width > 1 {
			t = float64(col) / float64(width-1)
		}
		b.WriteString(
			lipgloss.NewStyle().
				Foreground(lipgloss.Color(gradientColorAt(ramp, t))).
				Width(1).
				Render(glyph),
		)
	}
	return b.String()
}

// gradientColorAt returns one interpolated hex color at t in [0,1] across the given stops.
func gradientColorAt(stops []string, t float64) string {
	if len(stops) == 0 {
		return "#000000"
	}
	if len(stops) == 1 {
		return normalizeHex(stops[0])
	}
	if t <= 0 {
		return normalizeHex(stops[0])
	}
	if t >= 1 {
		return normalizeHex(stops[len(stops)-1])
	}
	segments := len(stops) - 1
	position := t * float64(segments)
	index := int(math.Floor(position))
	localT := position - float64(index)
	if index >= segments {
		return normalizeHex(stops[len(stops)-1])
	}
	r1, g1, b1 := parseHexColor(stops[index])
	r2, g2, b2 := parseHexColor(stops[index+1])
	r := int(math.Round(float64(r1) + (float64(r2-r1) * localT)))
	g := int(math.Round(float64(g1) + (float64(g2-g1) * localT)))
	b := int(math.Round(float64(b1) + (float64(b2-b1) * localT)))
	return fmt.Sprintf("#%02X%02X%02X", clamp(r, 0, 255), clamp(g, 0, 255), clamp(b, 0, 255))
}

// normalizeHex returns one normalized #RRGGBB value for the provided color string.
func normalizeHex(hex string) string {
	r, g, b := parseHexColor(hex)
	return fmt.Sprintf("#%02X%02X%02X", r, g, b)
}

// parseHexColor parses #RRGGBB (or RRGGBB) to RGB ints and falls back to gray on invalid input.
func parseHexColor(hex string) (int, int, int) {
	s := strings.TrimSpace(strings.TrimPrefix(hex, "#"))
	if len(s) != 6 {
		return 128, 128, 128
	}
	var r, g, b int
	if _, err := fmt.Sscanf(strings.ToUpper(s), "%02X%02X%02X", &r, &g, &b); err != nil {
		return 128, 128, 128
	}
	return clamp(r, 0, 255), clamp(g, 0, 255), clamp(b, 0, 255)
}

// truncate shortens text to fit one line at the requested width.
func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	if width == 1 {
		return string(runes[:1])
	}
	return string(runes[:width-1]) + "~"
}

// clamp constrains one integer between lower and upper bounds.
func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// max returns the larger of the two values.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
