package fetcher

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Quote struct {
	Symbol string
	Price  float64
	Change float64
}

var (
	dolarCache map[string]float64
	cacheMutex sync.Mutex
	cacheFile  string
)

// ensureCacheLoaded initializes cacheFile and loads cache from disk if needed
func ensureCacheLoaded() {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	if dolarCache != nil {
		return
	}
	dolarCache = make(map[string]float64)

	// choose user cache dir
	dir, err := os.UserCacheDir()
	if err != nil || dir == "" {
		// fallback to current working directory
		dir = "."
	}
	dir = filepath.Join(dir, "waybar-stocks")
	_ = os.MkdirAll(dir, 0o755)
	cacheFile = filepath.Join(dir, "dolar_cache.json")

	// load if exists
	b, err := ioutil.ReadFile(cacheFile)
	if err != nil {
		return
	}
	var m map[string]float64
	if err := json.Unmarshal(b, &m); err == nil {
		for k, v := range m {
			dolarCache[k] = v
		}
	}
}

func saveCache() error {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	if cacheFile == "" {
		// ensure path
		dir, err := os.UserCacheDir()
		if err != nil || dir == "" {
			dir = "."
		}
		dir = filepath.Join(dir, "waybar-stocks")
		_ = os.MkdirAll(dir, 0o755)
		cacheFile = filepath.Join(dir, "dolar_cache.json")
	}
	b, err := json.MarshalIndent(dolarCache, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(cacheFile, b, 0o644)
}

func getPrevPrice(symbol string) float64 {
	ensureCacheLoaded()
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	return dolarCache[symbol]
}

func setPrevPrice(symbol string, price float64) error {
	ensureCacheLoaded()
	cacheMutex.Lock()
	dolarCache[symbol] = price
	cacheMutex.Unlock()
	return saveCache()
}

func GetQuote(symbol string) (*Quote, error) {
	// Dólar API special symbols (e.g. "dolar-oficial", "dolar-blue", "dolar-ccl", "dolar-cripto")
	if strings.HasPrefix(strings.ToLower(symbol), "dolar-") {
		return getDolarAPI(symbol)
	}

	if strings.Contains(symbol, "USD") { // Cryptos: CoinGecko
		return getCrypto(symbol)
	}
	return getYahoo(symbol) // Stocks: Yahoo
}

// getDolarAPI fetches dollar quotations from https://dolarapi.com
func getDolarAPI(symbol string) (*Quote, error) {
	// map common symbol names to API endpoints
	m := map[string]string{
		"dolar-oficial":         "oficial",
		"dolar-blue":            "blue",
		"dolar-bolsa":           "bolsa",
		"dolar-mep":             "bolsa",
		"dolar-ccl":             "contadoconliqui",
		"dolar-contadoconliqui": "contadoconliqui",
		"dolar-tarjeta":         "tarjeta",
		"dolar-mayorista":       "mayorista",
		"dolar-cripto":          "cripto",
	}

	key := strings.ToLower(symbol)
	endpoint, ok := m[key]
	if !ok {
		// fallback: try to use the part after "dolar-" directly
		if strings.HasPrefix(key, "dolar-") {
			endpoint = strings.TrimPrefix(key, "dolar-")
		} else {
			return nil, fmt.Errorf("unknown dolar symbol: %s", symbol)
		}
	}

	url := fmt.Sprintf("https://dolarapi.com/v1/dolares/%s", endpoint)
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d fetching %s from DolarApi", resp.StatusCode, endpoint)
	}

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("error parsing DolarApi JSON: %v", err)
	}

	// Parse venta (prefer) or compra
	var price float64
	if v, ok := data["venta"]; ok {
		if f, err := parseNumber(v); err == nil {
			price = f
		}
	}
	if price == 0 {
		if v, ok := data["compra"]; ok {
			if f, err := parseNumber(v); err == nil {
				price = f
			}
		}
	}
	if price == 0 {
		return nil, fmt.Errorf("no price found in DolarApi response for %s", endpoint)
	}

	// compute change against last stored price (rueda anterior)
	prev := getPrevPrice(symbol)
	var change float64
	if prev > 0 {
		change = (price - prev) / prev * 100
	}

	// store current price for next run
	if err := setPrevPrice(symbol, price); err != nil {
		// non-fatal: log to stderr via fmt (don't fail the fetch)
		fmt.Fprintf(os.Stderr, "warning: could not save dolar cache: %v\n", err)
	}

	return &Quote{Symbol: symbol, Price: price, Change: change}, nil
}

// parseNumber accepts numbers or strings (with comma/dot) and returns float64
func parseNumber(v interface{}) (float64, error) {
	switch t := v.(type) {
	case float64:
		return t, nil
	case float32:
		return float64(t), nil
	case int:
		return float64(t), nil
	case int64:
		return float64(t), nil
	case string:
		s := strings.TrimSpace(t)
		// normalize 1.234,56 -> 1234.56
		// if contains comma and dot, assume '.' thousand sep and ',' decimal
		if strings.Contains(s, ",") && strings.Contains(s, ".") {
			s = strings.ReplaceAll(s, ".", "")
			s = strings.ReplaceAll(s, ",", ".")
		} else {
			// replace comma with dot
			s = strings.ReplaceAll(s, ",", ".")
		}
		return strconv.ParseFloat(s, 64)
	default:
		return 0, fmt.Errorf("unsupported number type: %T", v)
	}
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
