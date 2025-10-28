package fetcher

import (
	"encoding/json"
	"fmt"
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
	b, err := os.ReadFile(cacheFile)
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
	return os.WriteFile(cacheFile, b, 0o644)
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

func GetQuote(symbol, timeframe string) (*Quote, error) {
	// Dólar API special symbols (e.g. "dolar-oficial", "dolar-blue", "dolar-ccl", "dolar-cripto")
	if strings.HasPrefix(strings.ToLower(symbol), "dolar-") {
		return getDolarAPI(symbol)
	}

	if strings.Contains(symbol, "USD") { // Cryptos: CoinGecko
		return getCrypto(symbol, timeframe)
	}
	return getYahoo(symbol, timeframe) // Stocks: Yahoo
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

// getYahoo fetches quote and computes change for the requested timeframe.
func getYahoo(symbol, timeframe string) (*Quote, error) {
	// default URL (we may add range/interval query params later)
	baseURL := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s", symbol)
	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", baseURL, nil)
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

	// If timeframe is empty or daily, prefer meta change percent or previousClose
	tf := strings.TrimSpace(strings.ToUpper(timeframe))
	if tf == "" || tf == "D" || tf == "1D" {
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
		// fallback: compute from last two closes
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

	// For other timeframes, request chart with a range/interval likely to include the timeframe
	dur, err := parseTimeframeToDuration(tf)
	if err != nil {
		// unknown timeframe: fallback to daily
		return &Quote{Symbol: symbol, Price: price, Change: 0}, nil
	}

	yarange, interval := mapDurationToYahooRangeInterval(dur)
	url := fmt.Sprintf("%s?range=%s&interval=%s", baseURL, yarange, interval)

	client2 := &http.Client{Timeout: 10 * time.Second}
	req2, _ := http.NewRequest("GET", url, nil)
	req2.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36")
	resp2, err := client2.Do(req2)
	if err != nil {
		return nil, err
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d while fetching chart %s", resp2.StatusCode, symbol)
	}
	var d2 map[string]interface{}
	if err := json.NewDecoder(resp2.Body).Decode(&d2); err != nil {
		return nil, err
	}
	chart2, _ := d2["chart"].(map[string]interface{})
	results2, _ := chart2["result"].([]interface{})
	if len(results2) == 0 {
		return nil, fmt.Errorf("no results for %s (chart)", symbol)
	}
	r0, _ := results2[0].(map[string]interface{})
	timestamps, _ := r0["timestamp"].([]interface{})
	indicators, _ := r0["indicators"].(map[string]interface{})
	var closes []interface{}
	if quoteArr, ok := indicators["quote"].([]interface{}); ok && len(quoteArr) > 0 {
		if quote, ok := quoteArr[0].(map[string]interface{}); ok {
			closes, _ = quote["close"].([]interface{})
		}
	}
	if len(timestamps) == 0 || len(closes) == 0 {
		return &Quote{Symbol: symbol, Price: price, Change: 0}, nil
	}
	// find last non-nil close as current
	var lastIdx int = -1
	for i := len(closes) - 1; i >= 0; i-- {
		if closes[i] != nil {
			lastIdx = i
			break
		}
	}
	if lastIdx == -1 {
		return &Quote{Symbol: symbol, Price: price, Change: 0}, nil
	}
	lastTsF := timestamps[lastIdx].(float64)
	lastTs := int64(lastTsF)
	currClose, _ := closes[lastIdx].(float64)

	// target timestamp
	targetTs := lastTs - int64(dur.Seconds())
	// find index with timestamp <= targetTs
	var targetIdx int = -1
	for i := lastIdx; i >= 0; i-- {
		if timestamps[i] == nil {
			continue
		}
		tsf, ok := timestamps[i].(float64)
		if !ok {
			continue
		}
		if int64(tsf) <= targetTs {
			targetIdx = i
			break
		}
	}
	if targetIdx == -1 {
		// not found earlier; use first value
		targetIdx = 0
	}
	prevClose, _ := closes[targetIdx].(float64)
	var change float64
	if prevClose != 0 {
		change = (currClose - prevClose) / prevClose * 100
	}
	return &Quote{Symbol: symbol, Price: currClose, Change: change}, nil
}

// parseTimeframeToDuration parses strings like "15m", "1H", "3D", "1W", "1M", "1Y".
// Rules (case-sensitive-ish):
// - suffix "MM" or "mm" or "min" or lowercase "m" -> minutes
// - uppercase "M" (single) -> months
// - H/h -> hours, D/d -> days, W/w -> weeks, Y/y -> years
func parseTimeframeToDuration(tf string) (time.Duration, error) {
	s := strings.TrimSpace(tf)
	if s == "" {
		return 24 * time.Hour, nil
	}
	// detect minutes formats
	if strings.HasSuffix(s, "MM") || strings.HasSuffix(s, "mm") || strings.HasSuffix(strings.ToLower(s), "min") || (strings.HasSuffix(s, "m") && !strings.HasSuffix(strings.ToLower(s), "mo")) {
		// minutes
		num := s
		num = strings.TrimSuffix(num, "MM")
		num = strings.TrimSuffix(num, "mm")
		num = strings.TrimSuffix(num, "min")
		num = strings.TrimSuffix(num, "m")
		if num == "" {
			num = "1"
		}
		n, err := strconv.Atoi(num)
		if err != nil {
			return 0, err
		}
		return time.Duration(n) * time.Minute, nil
	}
	// months (single uppercase M)
	if strings.HasSuffix(s, "M") && !strings.HasSuffix(s, "MM") {
		num := strings.TrimSuffix(s, "M")
		if num == "" {
			num = "1"
		}
		n, err := strconv.Atoi(num)
		if err != nil {
			return 0, err
		}
		return time.Duration(n*24*30) * time.Hour, nil // approx 30 days
	}
	// generic suffixes (case-insensitive)
	lower := strings.ToLower(s)
	switch {
	case strings.HasSuffix(lower, "h"):
		num := strings.TrimSuffix(lower, "h")
		if num == "" {
			num = "1"
		}
		n, err := strconv.Atoi(num)
		if err != nil {
			return 0, err
		}
		return time.Duration(n) * time.Hour, nil
	case strings.HasSuffix(lower, "d"):
		num := strings.TrimSuffix(lower, "d")
		if num == "" {
			num = "1"
		}
		n, err := strconv.Atoi(num)
		if err != nil {
			return 0, err
		}
		return time.Duration(n*24) * time.Hour, nil
	case strings.HasSuffix(lower, "w"):
		num := strings.TrimSuffix(lower, "w")
		if num == "" {
			num = "1"
		}
		n, err := strconv.Atoi(num)
		if err != nil {
			return 0, err
		}
		return time.Duration(n*7*24) * time.Hour, nil
	case strings.HasSuffix(lower, "y"):
		num := strings.TrimSuffix(lower, "y")
		if num == "" {
			num = "1"
		}
		n, err := strconv.Atoi(num)
		if err != nil {
			return 0, err
		}
		return time.Duration(n*365*24) * time.Hour, nil
	default:
		// try parse as days number
		if n, err := strconv.Atoi(s); err == nil {
			return time.Duration(n*24) * time.Hour, nil
		}
	}
	return 0, fmt.Errorf("unknown timeframe: %s", tf)
}

// mapDurationToYahooRangeInterval returns a reasonable range and interval for Yahoo chart API
func mapDurationToYahooRangeInterval(d time.Duration) (string, string) {
	if d <= time.Hour*24 {
		return "1d", "5m"
	}
	if d <= time.Hour*24*7 {
		return "7d", "60m"
	}
	if d <= time.Hour*24*30 {
		return "1mo", "1d"
	}
	if d <= time.Hour*24*365 {
		return "1y", "1d"
	}
	return "5y", "1d"
}

func getCrypto(symbol, timeframe string) (*Quote, error) {
	id := map[string]string{
		"BTC-USD": "bitcoin",
		"ETH-USD": "ethereum",
		"SOL-USD": "solana",
	}[symbol]
	if id == "" {
		return nil, fmt.Errorf("crypto %s not supported", symbol)
	}

	// For timeframe-aware crypto data we use CoinGecko market endpoints
	// First, get current market data
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

	// If timeframe is empty or 24h, use the provided 24h field
	tf := strings.TrimSpace(strings.ToUpper(timeframe))
	if tf == "" || tf == "24H" || tf == "1D" || tf == "D" {
		var change float64
		if v, ok := data[0]["price_change_percentage_24h"].(float64); ok {
			change = v
		}
		return &Quote{Symbol: symbol, Price: price, Change: change}, nil
	}

	// otherwise, try to compute from market_chart (days param)
	dur, err := parseTimeframeToDuration(tf)
	if err != nil {
		return &Quote{Symbol: symbol, Price: price, Change: 0}, nil
	}
	// CoinGecko market_chart accepts days as float; we pass at least 1
	days := int((dur + 23*time.Hour) / (24 * time.Hour))
	if days < 1 {
		days = 1
	}
	mcurl := fmt.Sprintf("https://api.coingecko.com/api/v3/coins/%s/market_chart?vs_currency=usd&days=%d", id, days)
	resp2, err := client.Get(mcurl)
	if err != nil {
		return nil, err
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d fetching market_chart for %s", resp2.StatusCode, id)
	}
	var chart struct {
		Prices [][]float64 `json:"prices"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&chart); err != nil {
		return nil, err
	}
	if len(chart.Prices) == 0 {
		return &Quote{Symbol: symbol, Price: price, Change: 0}, nil
	}
	// market_chart.Prices: [ [ts_ms, price], ... ]
	// find last price and target timestamp
	last := chart.Prices[len(chart.Prices)-1]
	lastTs := int64(last[0]) / 1000
	lastPrice := last[1]
	targetTs := lastTs - int64(dur.Seconds())
	// find nearest earlier price
	var prevPrice float64
	for i := len(chart.Prices) - 1; i >= 0; i-- {
		p := chart.Prices[i]
		ts := int64(p[0]) / 1000
		if ts <= targetTs {
			prevPrice = p[1]
			break
		}
	}
	if prevPrice == 0 {
		prevPrice = chart.Prices[0][1]
	}
	var change float64
	if prevPrice != 0 {
		change = (lastPrice - prevPrice) / prevPrice * 100
	}
	return &Quote{Symbol: symbol, Price: lastPrice, Change: change}, nil
}
