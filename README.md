# waybar-stocks

A customizable **Waybar module** written in **Go** that displays real-time prices for **stocks, cryptocurrencies and indices**.  
Supports price rotation, percent change indicators, custom colors, and YAML configuration — no external Python scripts or dependencies required.

## Features

- Display stock, crypto, or index prices (e.g. `BTC-USD` (or `bitcoin` like CoinGecko's naming), `AAPL`, `SPY`, `^GSPC`), and Argentinian Dólar prices (`dolar-oficial`, `dolar-blue`, `dolar-cripto`, etc.).
- Shows **price + percent change + up/down icons** for each asset.
- Configurable via [`config.yml`](/config.yml).
  - Supports **color customization**
  - Assets
  - Refresh & rotation interval!
  - Timeframe for each stock! (see [Timeframe (per-asset)](README.md#timeframe-per-asset))
- Fast & lightweight (native Go binary, no runtime needed)
- Includes CLI flags like `--help` and `--config`
- Works perfectly with **Waybar (Hyprland / Sway / niri / dwl)**

## Requirements

- Go 1.20+ (or the version configured in [`go.mod`](/go.mod))
- Internet access to query Yahoo Finance and CoinGecko

## Installation

1. Clone the repository:

```bash
git clone https://github.com/bautitobal/waybar-stocks.git
cd waybar-stocks
```

2. Build:

```bash
go build -o waybar-stocks
```
(Opcional) Move it to PATH:

```bash
sudo mv waybar-stocks ~/.local/bin/
```

## Configuration
Create or edit the file [`config.yml`](/config.yml) in the project folder or any custom path.

```yaml
refresh_interval: 60        # seconds between API updates
rotation_interval: 5        # seconds to display each asset
format: "{symbol} {price} ({change}%{icon})"

colors:
  up: "#00FF00"        # price increasing
  down: "#FF5555"      # price decreasing
  neutral: "#FFFFFF"   # no change

assets:
  - symbol: BTC-USD
    name: BTC
    timeframe: 1D
  - symbol: AAPL
    name: AAPL
    timeframe: 1W
  - symbol: SPY
    name: S&P500
    timeframe: 1M
  - symbol: dolar-oficial
    name: DÓLAR OFICIAL
    timeframe: 1D
  - symbol: dolar-cripto
    name: DÓLAR CRIPTO
    timeframe: 1D
```

### Timeframe (per-asset)

You can optionally set a `timeframe` per asset to control which period the percent change is computed for. If omitted, the default is daily (`1D`). Examples:

- `15m` — 15 minutes
- `1H` or `H` — 1 hour
- `1D` or `D` — 1 day (default)
- `3D` — 3 days
- `1W` or `W` — 1 week
- `1M` — 1 month (approximated as 30 days)
- `1Y` or `Y` — 1 year (approximated as 365 days)

Notes:
- For stocks the fetcher will prefer Yahoo's session metadata for daily change, or request Yahoo chart data for custom timeframes and compute the percent between "now" and "timeframe ago".
- For cryptocurrencies the fetcher uses CoinGecko's 24h percent by default; for custom timeframes it queries CoinGecko's market_chart and computes the percent accordingly.
- For `dolar-*` symbols the app fetches the latest `venta` (or `compra`) from DolarApi and computes change relative to the last saved value (persisted in the cache). If you need percent vs a fixed period (e.g. 24h), the app can be extended to keep a longer history or use historical endpoints.

You can include the `{timeframe}` token in your `format` string to show the timeframe explicitly. If `{timeframe}` is not present, the timeframe will be appended to the symbol automatically when set.


## Add to Waybar
In your `~/.config/waybar/config.jsonc`, add:

```jsonc
"custom/stocks": {
  "exec": "~/waybar-stocks --config ~/waybar-stocks/config.yml", // or ~/.local/bin/waybar-stocks --config ~/.local/bin/waybar-stocks/config.yml; could be ANY config PATH tbh.
  "return-type": "json",
  "interval": 1
}
```

Then reload waybar.

## 🛠 Command Line Usage

```bash
waybar-stocks --help
```

## Contributing

Pull requests are welcome!
Feel free to open an issue for feature requests or bugs.

## License

This project is available under the MIT License — see the [`LICENSE`](/LICENSE) file for details.

## Support
If you have any questions or need help, feel free to open an issue or contact the maintainers.

### Socials
[![LinkedIn](https://img.shields.io/badge/LinkedIn-%230077B5.svg?logo=linkedin&logoColor=white)](https://linkedin.com/in/bautistatobal) [![Roadmap](https://img.shields.io/badge/Roadmap-000000?style=flat&logo=roadmap.sh&logoColor=white)](https://roadmap.sh/u/bautitobal) [![Instagram](https://img.shields.io/badge/Instagram-%23E4405F.svg?logo=Instagram&logoColor=white)](https://instagram.com/bautitobal) [![Behance](https://img.shields.io/badge/Behance-1769ff?logo=behance&logoColor=white)](https://behance.net/bautitobal) [![X](https://img.shields.io/badge/X-black.svg?logo=X&logoColor=white)](https://x.com/bautitobal) [![Medium](https://img.shields.io/badge/Medium-12100E?logo=medium&logoColor=white)](https://medium.com/@bautitobal) [![Goodreads](https://img.shields.io/badge/Goodreads-F3F1EA?style=for-the-badge&logo=goodreads&logoColor=372213)](https://www.goodreads.com/bautitobal)

#### Donate
Feel free to donate! (If you can afford and you want of course).

[![PayPal](https://img.shields.io/badge/PayPal-00457C?style=for-the-badge&logo=paypal&logoColor=white)](https://paypal.me/bautitobal) [![Ko-Fi](https://img.shields.io/badge/Ko--fi-F16061?style=for-the-badge&logo=ko-fi&logoColor=white)](https://ko-fi.com/bautitobal) 
