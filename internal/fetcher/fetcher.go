package fetcher

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Quote struct {
	Symbol string
	Price  float64
	Change float64
}

func GetQuote(symbol string) (*Quote, error) {
	if strings.Contains(symbol, "USD") { // Cryptos: CoinGecko
		return getCrypto(symbol)
	}
	return getYahoo(symbol) // Stocks: Yahoo
}

func getYahoo(symbol string) (*Quote, error) {
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s", symbol)
	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d while fetching %s", resp.StatusCode, symbol)
	}

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("error parsing Yahoo JSON: %v", err)
	}

	chart, ok := data["chart"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid JSON structure (chart) for %s", symbol)
	}
	results, ok := chart["result"].([]interface{})
	if !ok || len(results) == 0 {
		return nil, fmt.Errorf("no results for %s", symbol)
	}

	res0, ok := results[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid result structure for %s", symbol)
	}

	meta, _ := res0["meta"].(map[string]interface{})

	// Try to get price from meta, fallback to indicators.quote.close last value
	var price float64
	if meta != nil {
		if p, ok := meta["regularMarketPrice"].(float64); ok {
			price = p
		}
	}
	if price == 0 {
		if indicators, ok := res0["indicators"].(map[string]interface{}); ok {
			if quoteArr, ok := indicators["quote"].([]interface{}); ok && len(quoteArr) > 0 {
				if quote, ok := quoteArr[0].(map[string]interface{}); ok {
					if closes, ok := quote["close"].([]interface{}); ok {
						for i := len(closes) - 1; i >= 0; i-- {
							if closes[i] != nil {
								if p, ok := closes[i].(float64); ok {
									price = p
									break
								}
							}
						}
					}
				}
			}
		}
	}
	if price == 0 {
		return nil, fmt.Errorf("could not determine price for %s", symbol)
	}

	// Try to get percent change from meta, otherwise compute from previous close (several fallbacks)
	var change float64
	if meta != nil {
		if v, ok := meta["regularMarketChangePercent"].(float64); ok {
			change = v
		} else if prev, ok := meta["previousClose"].(float64); ok && prev != 0 {
			change = (price - prev) / prev * 100
		} else if prev, ok := meta["chartPreviousClose"].(float64); ok && prev != 0 {
			change = (price - prev) / prev * 100
		}
	}

	// If still zero/unknown, try to compute from last two closes in indicators
	if change == 0 {
		if indicators, ok := res0["indicators"].(map[string]interface{}); ok {
			if quoteArr, ok := indicators["quote"].([]interface{}); ok && len(quoteArr) > 0 {
				if quote, ok := quoteArr[0].(map[string]interface{}); ok {
					if closes, ok := quote["close"].([]interface{}); ok {
						var last, prevVal float64
						found := 0
						for i := len(closes) - 1; i >= 0 && found < 2; i-- {
							if closes[i] != nil {
								if v, ok := closes[i].(float64); ok {
									if found == 0 {
										last = v
									} else {
										prevVal = v
									}
									found++
								}
							}
						}
						if found >= 2 && prevVal != 0 {
							change = (last - prevVal) / prevVal * 100
						}
					}
				}
			}
		}
	}

	return &Quote{Symbol: symbol, Price: price, Change: change}, nil
}

func getCrypto(symbol string) (*Quote, error) {
	id := map[string]string{
		"BTC-USD": "bitcoin",
		"ETH-USD": "ethereum",
		"SOL-USD": "solana",
	}[symbol]
	if id == "" {
		return nil, fmt.Errorf("crypto %s not supported", symbol)
	}

	url := fmt.Sprintf("https://api.coingecko.com/api/v3/coins/markets?vs_currency=usd&ids=%s", id)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d fetching %s from CoinGecko", resp.StatusCode, symbol)
	}

	var data []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("no data for %s", symbol)
	}

	price := data[0]["current_price"].(float64)
	change := data[0]["price_change_percentage_24h"].(float64)

	return &Quote{Symbol: symbol, Price: price, Change: change}, nil
}
