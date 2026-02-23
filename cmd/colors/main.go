// Package main provides a tool to display ANSI 256 colors and various theme palettes.
package main

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

func main() {
	// Display 256 colors first
	fmt.Println("=== ANSI 256 COLORS ===")
	display256Colors()

	// Display Charm theme colors
	fmt.Println("\n\n=== CHARM THEME COLORS ===")
	displayCharmTheme()

	// Display color profile capabilities
	fmt.Println("\n\n=== COLOR PROFILE SUPPORT ===")
	displayColorProfiles()
}

func display256Colors() {
	// Display standard 16 colors (0-15)
	fmt.Println("Standard 16 Colors:")
	displayColorBlock(0, 15, 8)

	// Display 216 color cube (16-231)
	fmt.Println("\n\n216 Color Cube (16-231):")
	for i := 0; i < 6; i++ {
		displayColorBlock(16+i*36, 16+(i+1)*36-1, 6)
		fmt.Println()
	}

	// Display grayscale (232-255)
	fmt.Println("\nGrayscale (232-255):")
	displayColorBlock(232, 255, 12)
}

func displayColorBlock(start, end, perRow int) {
	count := 0
	for i := start; i <= end; i++ {
		// Create a style with the background color
		style := lipgloss.NewStyle().
			Background(lipgloss.Color(strconv.Itoa(i))).
			Foreground(getContrastColor(i)).
			Width(6).
			Align(lipgloss.Center)

		// Render the color number with the background
		fmt.Print(style.Render(fmt.Sprintf("%3d", i)))

		count++
		if count%perRow == 0 {
			fmt.Println()
		} else {
			fmt.Print(" ")
		}
	}
	if count%perRow != 0 {
		fmt.Println()
	}
}

// Helper function to determine contrast color for text
func getContrastColor(colorIndex int) lipgloss.Color {
	// Use white text for dark colors, black for light colors
	switch {
	case colorIndex < 16:
		// For standard colors, use white for dark colors
		if colorIndex == 0 || colorIndex == 1 || colorIndex == 4 || colorIndex == 5 || colorIndex == 8 {
			return lipgloss.Color("15") // white
		}
		return lipgloss.Color("0") // black
	case colorIndex >= 232:
		// For grayscale
		if colorIndex < 244 {
			return lipgloss.Color("15") // white
		}
		return lipgloss.Color("0") // black
	default:
		// For 216 color cube, use a simple heuristic
		return lipgloss.Color("15") // white (can be improved)
	}
}

func displayCharmTheme() {
	// Create a table for Charm colors
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("62"))).
		Headers("Color", "Name", "Light", "Dark", "Sample").
		StyleFunc(func(row, _ int) lipgloss.Style {
			if row == 0 {
				return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230"))
			}
			return lipgloss.NewStyle()
		})

	// Charm color palette based on huh/theme.go
	charmColors := []struct {
		name      string
		lightHex  string
		darkHex   string
		lightAnsi string
		darkAnsi  string
	}{
		{"Normal FG", "", "", "235", "252"},
		{"Indigo", "#5A56E0", "#7571F9", "", ""},
		{"Cream", "#FFFDF5", "#FFFDF5", "", ""},
		{"Fuchsia", "#F780E2", "#F780E2", "", ""},
		{"Green", "#02BA84", "#02BF87", "", ""},
		{"Red", "#FF4672", "#ED567A", "", ""},
		{"Border", "", "", "238", "238"},
		{"Placeholder", "", "", "248", "238"},
		{"Comment", "", "", "243", "243"},
	}

	for _, c := range charmColors {
		var lightSample, darkSample string

		// Create light sample
		if c.lightHex != "" {
			lightSample = lipgloss.NewStyle().
				Background(lipgloss.Color(c.lightHex)).
				Foreground(lipgloss.Color("0")).
				Width(10).
				Align(lipgloss.Center).
				Render(c.lightHex)
		} else if c.lightAnsi != "" {
			lightSample = lipgloss.NewStyle().
				Background(lipgloss.Color(c.lightAnsi)).
				Foreground(getContrastColor(mustAtoi(c.lightAnsi))).
				Width(10).
				Align(lipgloss.Center).
				Render(c.lightAnsi)
		}

		// Create dark sample
		if c.darkHex != "" {
			darkSample = lipgloss.NewStyle().
				Background(lipgloss.Color(c.darkHex)).
				Foreground(lipgloss.Color("15")).
				Width(10).
				Align(lipgloss.Center).
				Render(c.darkHex)
		} else if c.darkAnsi != "" {
			darkSample = lipgloss.NewStyle().
				Background(lipgloss.Color(c.darkAnsi)).
				Foreground(getContrastColor(mustAtoi(c.darkAnsi))).
				Width(10).
				Align(lipgloss.Center).
				Render(c.darkAnsi)
		}

		// Combine samples
		sample := lightSample + " " + darkSample

		t.Row(
			c.name,
			c.name,
			coalesce(c.lightHex, c.lightAnsi),
			coalesce(c.darkHex, c.darkAnsi),
			sample,
		)
	}

	fmt.Println(t.Render())

	// Display Dracula theme colors too
	fmt.Println("\n\nDracula Theme Colors:")

	draculaTable := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("62"))).
		Headers("Color", "Name", "Hex", "Sample").
		StyleFunc(func(row, _ int) lipgloss.Style {
			if row == 0 {
				return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230"))
			}
			return lipgloss.NewStyle()
		})

	draculaColors := []struct {
		name string
		hex  string
	}{
		{"Background", "#282a36"},
		{"Selection", "#44475a"},
		{"Foreground", "#f8f8f2"},
		{"Comment", "#6272a4"},
		{"Green", "#50fa7b"},
		{"Purple", "#bd93f9"},
		{"Red", "#ff5555"},
		{"Yellow", "#f1fa8c"},
	}

	for _, c := range draculaColors {
		sample := lipgloss.NewStyle().
			Background(lipgloss.Color(c.hex)).
			Foreground(lipgloss.Color("#f8f8f2")).
			Width(20).
			Align(lipgloss.Center).
			Render(c.hex)

		draculaTable.Row(c.name, c.name, c.hex, sample)
	}

	fmt.Println(draculaTable.Render())
}

func displayColorProfiles() {
	// Create a table showing color profile support
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("62"))).
		Headers("Profile", "Colors", "Bits", "Description").
		StyleFunc(func(row, _ int) lipgloss.Style {
			if row == 0 {
				return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230"))
			}
			return lipgloss.NewStyle()
		})

	t.Row("ASCII", "2", "1-bit", "Black and white only")
	t.Row("ANSI", "16", "4-bit", "Standard 16 terminal colors")
	t.Row("ANSI256", "256", "8-bit", "Extended 256 color palette")
	t.Row("TrueColor", "16,777,216", "24-bit", "Full RGB color support")

	fmt.Println(t.Render())

	fmt.Println("\nLipgloss automatically detects your terminal's color profile and")
	fmt.Println("degrades colors gracefully to the best available approximation.")

	// Show color specification examples
	fmt.Println("\n\nColor Specification Examples:")
	examples := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("62"))).
		Headers("Method", "Example", "Description").
		StyleFunc(func(row, _ int) lipgloss.Style {
			if row == 0 {
				return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230"))
			}
			return lipgloss.NewStyle()
		})

	examples.Row("Hex", `lipgloss.Color("#FF6B6B")`, "Hex color notation")
	examples.Row("ANSI256", `lipgloss.Color("201")`, "256 color by number")
	examples.Row("ANSI", `lipgloss.Color("5")`, "Standard 16 color")
	examples.Row("Adaptive", `lipgloss.AdaptiveColor{Light: "235", Dark: "252"}`, "Auto light/dark")
	examples.Row("Complete", `lipgloss.CompleteColor{TrueColor: "#FF0000", ANSI256: "196", ANSI: "9"}`, "Exact per profile")

	fmt.Println(examples.Render())
}

func mustAtoi(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

func coalesce(strs ...string) string {
	for _, s := range strs {
		if s != "" {
			return s
		}
	}
	return ""
}
