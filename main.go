package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	pattern    = "/index.php?id=1"
	concurrent = 50 // Количество одновременных запросов
)

var client = &http.Client{
	Timeout: 5 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        concurrent,
		MaxIdleConnsPerHost: concurrent,
		IdleConnTimeout:     30 * time.Second,
	},
}

func main() {
	file, err := os.Open("majestic_million.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	in := csv.NewReader(file)
	records, err := in.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	results := make(chan string, 100)
	sem := make(chan struct{}, concurrent)

	var wg sync.WaitGroup
	wg.Add(len(records))

	for i, record := range records {
		if i%100 == 0 {
			loading(i, len(records))
		}

		sem <- struct{}{}
		go func(i int, record []string) {
			defer func() {
				<-sem
				wg.Done()
			}()

			url := "http://" + record[2] + pattern
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				return
			}

			res, err := client.Do(req)
			if err != nil {
				return
			}
			defer res.Body.Close()

			if res.StatusCode == http.StatusOK {
				results <- url
			}
		}(i, record)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for url := range results {
		f, err := os.OpenFile("result.txt", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0777)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		io.WriteString(f, url)

	}

	loading(len(records), len(records))
	fmt.Println("Scan completed!")
}

func loading(curr, total int) {
	clearConsole()

	if total <= 0 || curr < 0 || curr > total {
		fmt.Println("Invalid progress values")
		return
	}

	percentage := (float64(curr) / float64(total)) * 100
	barWidth := 50
	progress := int(float64(barWidth) * percentage / 100)

	bar := strings.Repeat("=", progress) + strings.Repeat(" ", barWidth-progress)

	fmt.Printf("\r[%s] %.2f%% (%d/%d)", bar, percentage, curr, total)

	os.Stdout.Sync()
}

func clearConsole() {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	default:
		fmt.Print("\033[H\033[2J")
	}
}
