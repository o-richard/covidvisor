package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	db *sql.DB
}

type Query struct {
	Intent   string            `json:"intent"`
	Entities map[string]string `json:"entities"`
}

func NewDB() (*DB, error) {
	var isCreated bool
	databaseName := "db.sqlite3"
	if _, err := os.Stat(databaseName); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("unable to check database existence: %w", err)
		}
		file, err := os.Create(databaseName)
		if err != nil {
			return nil, fmt.Errorf("unable to create database: %w", err)
		}
		_ = file.Close()
		isCreated = true
	}

	db, err := sql.Open("sqlite3", databaseName)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to the database: %w", err)
	}
	if err = db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("unable to verify database connection: %w", err)
	}
	if isCreated {
		migrations := `
			CREATE TABLE covid_data (
				id SERIAL PRIMARY KEY,
				location TEXT NOT NULL,
				date DATE NOT NULL,
				tcin INT NOT NULL, -- Total Confirmed cases of Indian Nationals
				tcfn INT NOT NULL, -- Total Confirmed cases of Foreign Nationals
				cured INT NOT NULL, -- Recoveries
				death INT NOT NULL -- Deaths
			);
		`
		if _, err := db.Exec(migrations); err != nil {
			db.Close()
			if errPath := os.Remove(databaseName); errPath != nil {
				return nil, fmt.Errorf("unable to delete newly created database, remove the file '%v' manually if it exists: %w", databaseName, errPath)
			}
			return nil, fmt.Errorf("unable to perform migrations on newly created database: %w", err)
		}
	}
	return &DB{db: db}, nil
}

func (db *DB) InsertCovidData(entries []CovidData) (dbErr error) {
	if len(entries) == 0 {
		return nil
	}

	var count int
	_ = db.db.QueryRow("SELECT COUNT(*) FROM covid_data").Scan(&count)
	if count > 0 {
		return nil
	}

	tx, err := db.db.Begin()
	if err != nil {
		return fmt.Errorf("unable to start transaction, %w", err)
	}
	defer func() {
		if dbErr != nil {
			_ = tx.Rollback()
		}
	}()

	stmt, err := tx.Prepare("INSERT INTO covid_data (location, date, tcin, tcfn, cured, death) VALUES (?,?,?,?,?,?)")
	if err != nil {
		return fmt.Errorf("unable to create prepared statement, %w", err)
	}
	defer stmt.Close()

	for _, entry := range entries {
		if _, err := stmt.Exec(entry.Location, entry.Date, entry.TCIN, entry.TCFN, entry.Cured, entry.Death); err != nil {
			return fmt.Errorf("unable to insert covid data into the database, %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("unable to commit transaction, %w", err)
	}
	return nil
}

func (db *DB) ProcessQuery(s string) (answer string, isCustom bool, queryErr error) {
	var data Query
	if err := json.Unmarshal([]byte(s), &data); err != nil {
		return "", false, fmt.Errorf("unable to process query, %w", err)
	}

	caseKind := "(tcin+tcfn)"
	switch data.Entities["case_type"] {
	case "recovery_cases":
		caseKind = "(cured)"
	case "death_cases":
		caseKind = "(death)"
	}

	var duration string
	if data.Entities["duration"] != "all_time" {
		duration = fmt.Sprintf(` AND date <= CURRENT_DATE AND date >= DATE(CURRENT_DATE, '%v') `, data.Entities["duration"])
	}

	var args []any
	var query string
	switch data.Intent {
	case "cases_date":
		query = fmt.Sprintf(`SELECT CAST(COALESCE(SUM(%v), 0) AS TEXT) FROM covid_data WHERE date=CURRENT_DATE AND location=?`, caseKind)
		args = append(args, data.Entities["location"])
	case "max_cases_duration":
		query = fmt.Sprintf(`SELECT CAST(COALESCE(MAX(%v), 0) AS TEXT) FROM covid_data WHERE location=? %v`, caseKind, duration)
		args = append(args, data.Entities["location"])
	case "average_cases_duration":
		query = fmt.Sprintf(`SELECT CAST(COALESCE(AVG(%v), 0) AS TEXT) FROM covid_data WHERE location=? %v`, caseKind, duration)
		args = append(args, data.Entities["location"])
	case "sum_cases_duration":
		query = fmt.Sprintf(`SELECT CAST(COALESCE(SUM(%v), 0) AS TEXT) FROM covid_data WHERE location=? %v`, caseKind, duration)
		args = append(args, data.Entities["location"])
	case "location_based":
		query = fmt.Sprintf(`
		WITH casescte AS (
			SELECT location, SUM(%v) AS sum FROM covid_data
		)
		SELECT location FROM casescte ORDER BY sum DESC LIMIT 1`, caseKind)
	case "date_based":
		lowerBoundNumber, _ := strconv.Atoi(data.Entities["lower_bound_number"])
		query = fmt.Sprintf(`
		WITH casescte AS (
			SELECT date, SUM(%v) AS sum FROM covid_data WHERE location = ? %v
		)
		SELECT CAST(date AS TEXT) FROM casescte WHERE sum >= ? ORDER BY sum DESC LIMIT 1`, caseKind, duration)
		args = append(args, data.Entities["location"], lowerBoundNumber)
	default:
		return "", true, nil
	}

	var result string
	err := db.db.QueryRow(query, args...).Scan(&result)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "No matching data available.", false, nil
		}
		return "", false, fmt.Errorf("unable to process database query, %w", err)
	}
	return result, false, nil
}

func (db *DB) Close() error {
	return db.db.Close()
}
