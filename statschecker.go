package main

import (
	"bufio"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	serverURL        = "http://srv.msk01.gigacorp.local/_stats"
	loadThreshold    = 30.0
	memoryThreshold  = 0.8 // 80%
	diskThreshold    = 0.9 // 90%
	networkThreshold = 0.9 // 90%
	maxErrors        = 3
	checkInterval    = 1 * time.Second // Изменила значение счетчика с 0 до 1
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

func fetchStats() ([]float64, error) {
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
	stats := make([]float64, 7)
	for i, val := range values {
		parsed, err := strconv.ParseFloat(strings.TrimSpace(val), 64)
		if err != nil {
			return nil, fmt.Errorf("invalid number format: %v", err)
		}
		stats[i] = parsed
	}

	return stats, nil
}

func checkMetrics(stats []float64) {
	// 0: Load Average
	load := stats[0]
	if load > loadThreshold {
		fmt.Printf("Load Average is too high: %.0f\n", load)
	}

	// 1: Total RAM, 2: Used RAM
	totalRAM := stats[1]
	usedRAM := stats[2]
	if totalRAM > 0 {
		memoryUsage := usedRAM / totalRAM
		if memoryUsage > memoryThreshold {
			percentage := memoryUsage * 100
			percentage = math.Floor(percentage) // Округление вниз
			fmt.Printf("Memory usage too high: %.0f%%\n", percentage)
		}
	}

	// 3: Total Disk, 4: Used Disk
	totalDisk := stats[3]
	usedDisk := stats[4]
	if totalDisk > 0 {
		diskUsage := usedDisk / totalDisk
		if diskUsage > diskThreshold {
			freeMB := (totalDisk - usedDisk) / (1024 * 1024)
			freeMB = math.Floor(freeMB) // Округление вниз
			fmt.Printf("Free disk space is too low: %.0f Mb left\n", freeMB)
		}
	}

	// 5: Total Network, 6: Used Network
	totalNetwork := stats[5]
	usedNetwork := stats[6]
	if totalNetwork > 0 {
		networkUsage := usedNetwork / totalNetwork
		if networkUsage > networkThreshold {
			// Конвертируем из байт/сек в мегабит/сек
			// 1 байт/сек = 8 бит/сек, 1 мегабит = 1,000,000 бит
			freeMbits := (totalNetwork - usedNetwork) * 8 / 10000000
			freeMbits = math.Floor(freeMbits) // Округление вниз
			fmt.Printf("Network bandwidth usage high: %.0f Mbit/s available\n", freeMbits)
		}
	}
}
