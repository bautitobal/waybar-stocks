package formatter

import (
	"fmt"
	"strings"
)

func escapeMarkup(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func FormatText(format, symbol, timeframe string, price, change float64, colorUp, colorDown, colorNeutral string) string {
	icon := "▲"
	color := colorUp

	if change < 0 {
		icon = "▼"
		color = colorDown
	} else if change == 0 {
		color = colorNeutral
	}

	// if the format does not include {timeframe}, append the timeframe to the symbol
	sym := escapeMarkup(symbol)
	tfEsc := escapeMarkup(timeframe)
	if tfEsc != "" && !strings.Contains(format, "{timeframe}") {
		sym = sym + " (" + tfEsc + ")"
	}

	out := strings.ReplaceAll(format, "{symbol}", sym)
	out = strings.ReplaceAll(out, "{price}", fmt.Sprintf("%.2f", price))
	out = strings.ReplaceAll(out, "{change}", fmt.Sprintf("%.2f", change))
	out = strings.ReplaceAll(out, "{timeframe}", tfEsc)
	out = strings.ReplaceAll(out, "{icon}", icon)

	return fmt.Sprintf("<span color='%s'>%s</span>", color, out)
}
