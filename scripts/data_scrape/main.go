package main

import (
	"context"
	"encoding/json"
	"fmt"
	sqliteStore "hft/internal/storage/sqlite"
	"hft/pkg/types"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type ApiResponse struct {
	Status string `json:"status"`
	Data   struct {
		Candles [][]interface{} `json:"candles"`
	} `json:"data"`
}

func fetchCandles(ctx context.Context, store *sqliteStore.TickStore, dateGroup map[string]string, wg *sync.WaitGroup) {
	defer wg.Done()

	fmt.Printf("Fetching data: Start: %s, End: %s\n", dateGroup["start"], dateGroup["end"])

	url := fmt.Sprintf("https://kite.zerodha.com/oms/instruments/historical/256265/minute?user_id=CK8434&oi=1&from=%s&to=%s", dateGroup["start"], dateGroup["end"])

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		log.Printf("build request error: %v", err)
		return
	}

	// Set headers (copied from previous script)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,hi;q=0.8")
	req.Header.Set("Authorization", "enctoken DVvW65EOL3uXOyf8omISU4yj99b2Cs5/WqbTKPY8EASF6YH9Cn3iC3qQVurMPOhLjhRGp+NDx1PuOrdLz12qbVya1+rXxqmQrMPNJNfMhwKsOLKy2RzPhA==")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Priority", "u=1, i")
	req.Header.Set("Sec-Ch-Ua", `"Not/A)Brand";v="8", "Chromium";v="126", "Google Chrome";v="126"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"macOS"`)
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("HTTP error for range %s-%s: %v", dateGroup["start"], dateGroup["end"], err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("read body error for range %s-%s: %v", dateGroup["start"], dateGroup["end"], err)
		return
	}

	var apiResponse ApiResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		log.Printf("decode error for range %s-%s: %v", dateGroup["start"], dateGroup["end"], err)
		return
	}

	var candles []types.Tick
	for _, item := range apiResponse.Data.Candles {
		// expected: [timestamp, open, high, low, close, volume, ...]
		if len(item) < 6 {
			continue
		}

		tsRaw, _ := item[0].(string)
		open, _ := item[1].(float64)
		high, _ := item[2].(float64)
		low, _ := item[3].(float64)
		closeV, _ := item[4].(float64)
		vol, _ := item[5].(float64)

		parsedTS, err := time.Parse("2006-01-02T15:04:05-0700", tsRaw)
		if err != nil {
			log.Printf("timestamp parse error (%s): %v", tsRaw, err)
			continue
		}

		candles = append(candles, types.Tick{
			Timestamp: parsedTS,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     closeV,
			Volume:    vol,
			Time:      parsedTS.Unix(),
			Symbol:    "nifty",
			TF:        "1",
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		})
	}

	if len(candles) == 0 {
		log.Printf("no candles parsed for range %s-%s", dateGroup["start"], dateGroup["end"])
		return
	}

	inserted, err := store.InsertTicks(ctx, candles)
	if err != nil {
		log.Printf("failed to insert ticks for range %s-%s: %v", dateGroup["start"], dateGroup["end"], err)
		return
	}
	log.Printf("inserted %d ticks for %s - %s", inserted, dateGroup["start"], dateGroup["end"])
}

func printGroupedDates(startDate, endDate string) []map[string]string {
	var dateGroups []map[string]string
	layout := "2006-01-02"
	start, _ := time.Parse(layout, startDate)
	end, _ := time.Parse(layout, endDate)
	for current := start; current.Before(end) || current.Equal(end); current = current.AddDate(0, 0, 3) {
		group := make(map[string]string)
		group["start"] = current.Format(layout)
		group["end"] = current.AddDate(0, 0, 2).Format(layout)
		dateGroups = append(dateGroups, group)
	}
	return dateGroups
}

func main() {
	ctx := context.Background()

	// Open sqlite in project root
	root, err := os.Getwd()
	if err != nil {
		log.Fatalf("get wd: %v", err)
	}
	store, err := sqliteStore.NewTickStore(filepath.Join(root, "hft.db"))
	if err != nil {
		log.Fatalf("open sqlite: %v", err)
	}
	defer store.Close()

	var wg sync.WaitGroup
	startDate := "2015-01-01"
	endDate := "2026-01-25"
	dg := printGroupedDates(startDate, endDate)

	// Limit concurrent requests
	semaphore := make(chan struct{}, 1)

	startTime := time.Now()
	for _, group := range dg {
		semaphore <- struct{}{}
		wg.Add(1)

		go func(group map[string]string) {
			defer func() { <-semaphore }()
			fetchCandles(ctx, store, group, &wg)
		}(group)
	}

	wg.Wait()
	endTime := time.Now()

	duration := endTime.Sub(startTime)
	fmt.Printf("Total time taken: %v\n", duration)
}
