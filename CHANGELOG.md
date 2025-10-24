# Changelog
All notable changes to this project will be documented in this file.

The format is based on **Keep a Changelog**  
and this project adheres to **Semantic Versioning**.

---

## [Unreleased]
### Added
- (Planned) Configurable decimal precision.
- (Planned) Cache system to reduce API calls.
- (Planned) Finnhub/AlphaVantage API support as alternative to Yahoo.

---

## [v0.2.0] - 2025-10-24
### Fixed
- fix(fetcher): correctly compute percent change for stocks
  - Use `regularMarketChangePercent` from Yahoo `meta` when available.
  - Fallbacks: compute from `previousClose` or `chartPreviousClose` if present.
  - Last-resort fallback: compute percent from the last two non-nil `close` values in `indicators.quote`.
  - Improved price extraction: prefer `regularMarketPrice`, otherwise use the last non-nil `close` from `indicators`.
  - Modified file: `internal/fetcher/fetcher.go`

## [v0.1.5] - 2025-10-18
### Added
- Initial release of **waybar-stocks**.
- Displays stock, crypto, and ETF prices in Waybar.
- Rotates assets using `rotation_interval`.
- `config.yml` with:
  - Custom `assets`
  - `refresh_interval`, `rotation_interval`
  - Custom `format` string
  - `colors` section (`up`, `down`, `neutral`)
- Support for command-line flags:
  - `--config` to specify a config file
  - `--help` to show usage info
- CoinGecko API support for cryptocurrencies.
- Yahoo Finance API support for stocks & indices.
- Markup-safe output for Waybar (fixed S&P500 `&` issue).
- Color logic:
  - Green if price increased
  - Red if price decreased
  - White if no change (0.00%)

---

## [v0.1.0] - Initial Development
### Added
- Prototype script working only with BTC from Yahoo.
- No rotation, no config file support yet.
- Hardcoded colors and values.

---

