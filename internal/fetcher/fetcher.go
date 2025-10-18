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

	chart := data["chart"].(map[string]interface{})
	results := chart["result"].([]interface{})
	if len(results) == 0 {
		return nil, fmt.Errorf("no results for %s", symbol)
	}
	meta := results[0].(map[string]interface{})["meta"].(map[string]interface{})
	price := meta["regularMarketPrice"].(float64)
	change, _ := meta["regularMarketChangePercent"].(float64)

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
