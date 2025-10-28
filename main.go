package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/bautitobal/waybar-stocks/internal/config"
	"github.com/bautitobal/waybar-stocks/internal/fetcher"
	"github.com/bautitobal/waybar-stocks/internal/formatter"
)

// CLI help / usage message
func printHelp() {
	fmt.Print(`waybar-stocks - Custom Waybar module for displaying stock and cryptocurrency prices.

USAGE:
  waybar-stocks [options]

OPTIONS:
  --config <path>    Path to the config.yml file (default: ./config.yml)
  --help             Show this help message and exit

EXAMPLE:
  waybar-stocks --config ~/.config/waybar/config.yml (if exists)

This program prints a JSON object to stdout that Waybar can render, for example:
  {"text": "<span color='#00FF00'>BTC (1D) 107000.00 (1.25%▲)</span>"}
`)
}

func main() {
	// Define flags
	configPath := flag.String("config", "config.yml", "Path to YAML config file")
	helpFlag := flag.Bool("help", false, "Show help and exit")

	flag.Parse()

	if *helpFlag {
		printHelp()
		return
	}

	if len(flag.Args()) > 0 {
		fmt.Fprintf(os.Stderr, "Unknown argument: %s\n\n", flag.Args()[0])
		printHelp()
		return
	}

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Rotate current asset based on time
	index := int(time.Now().Unix()/int64(cfg.RotationInterval)) % len(cfg.Assets)
	asset := cfg.Assets[index]

	// Fetch quote
	q, err := fetcher.GetQuote(asset.Symbol, asset.Timeframe)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching %s: %v\n", asset.Symbol, err)
		os.Exit(1)
	}

	// Format output with colors from config
	text := formatter.FormatText(
		cfg.Format,
		asset.Name,
		asset.Timeframe,
		q.Price,
		q.Change,
		cfg.Colors.Up,
		cfg.Colors.Down,
		cfg.Colors.Neutral,
	)

	// Print JSON for Waybar
	output := map[string]string{"text": text}
	json.NewEncoder(os.Stdout).Encode(output)
}
