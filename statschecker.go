package main

import (
	"bufio"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	serverURL        = "http://srv.msk01.gigacorp.local/_stats"
	loadThreshold    = 30
	memoryThreshold  = 80 // 80%
	diskThreshold    = 90 // 90%
	networkThreshold = 90 // 90%
	maxErrors        = 3
	checkInterval    = 0 * time.Second
)

func main() {
	errorCount := 0

	for {
		stats, err := fetchStats()
		if err != nil {
			errorCount++

			if errorCount >= maxErrors {
				fmt.Println("Unable to fetch server statistic")
				errorCount = 0
			}

			time.Sleep(checkInterval)
			continue
		}

		// Сбрасываем счетчик ошибок при успешном запросе
		errorCount = 0

		// Проверяем метрики
		checkMetrics(stats)

		time.Sleep(checkInterval)
	}
}

func fetchStats() ([]int, error) {
	resp, err := http.Get(serverURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP status: %s", resp.Status)
	}

	// Читаем тело ответа
	scanner := bufio.NewScanner(resp.Body)
	if !scanner.Scan() {
		return nil, fmt.Errorf("empty response")
	}

	line := scanner.Text()
	values := strings.Split(line, ",")

	if len(values) != 7 {
		return nil, fmt.Errorf("invalid data format: expected 7 values, got %d", len(values))
	}

	// Парсим числовые значения
	stats := make([]int, 7)
	for i, val := range values {
		parsed, err := strconv.Atoi(strings.TrimSpace(val))
		if err != nil {
			return nil, fmt.Errorf("invalid number format: %v", err)
		}
		stats[i] = parsed
	}

	return stats, nil
}

func checkMetrics(stats []int) {
	// 0: Load Average
	load := stats[0]
	if load > loadThreshold {
		fmt.Printf("Load Average is too high: %d\n", load)
	}

	// 1: Total RAM, 2: Used RAM
	totalRAM := stats[1]
	usedRAM := stats[2]
	if totalRAM > 0 {
		memoryUsage := usedRAM * 100 / totalRAM
		if memoryUsage > memoryThreshold {
			fmt.Printf("Memory usage too high: %d%%\n", memoryUsage)
		}
	}

	// 3: Total Disk, 4: Used Disk
	totalDisk := stats[3]
	usedDisk := stats[4]
	if totalDisk > 0 {
		diskUsage := usedDisk * 100 / totalDisk
		if diskUsage > diskThreshold {
			freeMB := (totalDisk - usedDisk) / (1024 * 1024)
			fmt.Printf("Free disk space is too low: %d Mb left\n", freeMB)
		}
	}

	// 5: Total Network, 6: Used Network
	totalNetwork := stats[5]
	usedNetwork := stats[6]
	if totalNetwork > 0 {
		networkUsage := usedNetwork * 100 / totalNetwork
		if networkUsage > networkThreshold {
			freeMbits := (totalNetwork - usedNetwork) / 1000000
			fmt.Printf("Network bandwidth usage high: %d Mbit/s available\n", freeMbits)
		}
	}
}
