package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"time"
)

type CovidData struct {
	TCIN     int // Total Confirmed cases of Indian Nationals
	TCFN     int // Total Confirmed cases of Foreign Nationals
	Cured    int // Recoveries
	Death    int // Deaths
	Location string
	Date     time.Time
}

func seedCovidData() ([]CovidData, error) {
	file, err := os.Open("datasets/covid.csv")
	if err != nil {
		return nil, fmt.Errorf("unable to open csv file, %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("unable to read all csv rows, %w", err)
	}
	// 1st row; contains dates
	// 2nd row; contains labels
	// 3rd+ row; contains data - last row contains the totals
	if len(rows) < 3 {
		return nil, fmt.Errorf("expected at least three rows in the csv file")
	}

	entries := map[string]CovidData{}
	for _, row := range rows[3 : len(rows)-1] {
		location := row[0]

		for colIndex, col := range row[1:] {
			date := rows[0][colIndex]
			dataType := rows[1][colIndex]

			data, _ := strconv.Atoi(col)
			key := fmt.Sprintf("%v-%v", date, location)
			entry, ok := entries[key]
			if !ok {
				entry.Date, _ = time.Parse("02/01/06", date)
				entry.Location = location
			}

			switch dataType {
			case "TCIN":
				entry.TCIN = data
			case "TCFN":
				entry.TCFN = data
			case "Cured":
				entry.Cured = data
			case "Death":
				entry.Death = data
			}
			entries[key] = entry
		}
	}

	result := make([]CovidData, 0, len(entries))
	for _, entry := range entries {
		result = append(result, entry)
	}
	return result, nil
}
