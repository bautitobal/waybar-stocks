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

## [v0.4.0] - 2025-10-24
### Added
- Integration with DolarApi (https://dolarapi.com) for `dolar-*` symbols:
  - Supported: `dolar-oficial`, `dolar-blue`, `dolar-bolsa` / `dolar-mep`, `dolar-ccl` / `dolar-contadoconliqui`, `dolar-tarjeta`, `dolar-mayorista`, `dolar-cripto`.
- Persistent cache of last prices to calculate percent changes between runs (cache file: $XDG_CACHE_HOME/waybar-stocks/dolar_cache.json).
- Robust number parsing for values from DolarApi (supports local thousands/decimal formats).
- Acceptance of `dolar-*` symbols in config assets (example: `- symbol: dolar-blue`).
- Added example entries for `dolar-oficial` and `dolar-cripto` in the default config file `config.yml`.

### Fixed
- fix(fetcher): calculate percent change for Dólar quotes using the last saved price (shows % relative to the previous saved value).
- fix(fetcher): improved fallbacks for stock percent change and price extraction from Yahoo Finance:
  - use `regularMarketChangePercent` when available;
  - fall back to `previousClose` or `chartPreviousClose`;
  - last-resort: compute from the last two non-nil `close` values in `indicators.quote`.

### Changed
- Updated `internal/fetcher/fetcher.go` to add DolarApi support, cache persistence, percent calculations, and improved parsing.
- Updated `config.yml` to include example Dólar assets.
- Updated `README.md` with DolarApi integration details and configuration instructions.

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

