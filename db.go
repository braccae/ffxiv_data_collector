package main

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/microsoft/go-mssqldb"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

var dbDriver string

func initDB(cfg *DatabaseConfig) (*sql.DB, error) {
	var driverName string
	var dsn string

	switch cfg.Type {
	case "sqlite":
		driverName = "libsql"
		dsn = cfg.DSN
		if !strings.HasPrefix(dsn, "file:") {
			dsn = "file:" + dsn
		}
	case "postgres", "postgresql":
		driverName = "postgres"
		dsn = cfg.DSN
	case "mysql":
		driverName = "mysql"
		dsn = cfg.DSN
	case "mssql", "sqlserver":
		driverName = "sqlserver"
		dsn = cfg.DSN
	case "libsql", "turso":
		driverName = "libsql"
		dsn = cfg.DSN
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Type)
	}

	dbDriver = driverName

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	var createEncountersTableSQL, createTravelLogTableSQL string

	switch driverName {
	case "postgres":
		createEncountersTableSQL = `
		CREATE TABLE IF NOT EXISTS encounters (
			id SERIAL PRIMARY KEY,
			encounter_id TEXT,
			encounter_title TEXT,
			duration TEXT,
			player_name TEXT,
			job TEXT,
			dps TEXT,
			healing TEXT,
			damage_taken TEXT,
			deaths TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`
		createTravelLogTableSQL = `
		CREATE TABLE IF NOT EXISTS travel_log (
			id SERIAL PRIMARY KEY,
			zone_id INTEGER,
			zone_name TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`
	case "mysql":
		createEncountersTableSQL = `
		CREATE TABLE IF NOT EXISTS encounters (
			id INT AUTO_INCREMENT PRIMARY KEY,
			encounter_id TEXT,
			encounter_title TEXT,
			duration TEXT,
			player_name TEXT,
			job TEXT,
			dps TEXT,
			healing TEXT,
			damage_taken TEXT,
			deaths TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`
		createTravelLogTableSQL = `
		CREATE TABLE IF NOT EXISTS travel_log (
			id INT AUTO_INCREMENT PRIMARY KEY,
			zone_id INT,
			zone_name TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`
	case "sqlserver":
		createEncountersTableSQL = `
		IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='encounters' and xtype='U')
		CREATE TABLE encounters (
			id INT IDENTITY(1,1) PRIMARY KEY,
			encounter_id NVARCHAR(MAX),
			encounter_title NVARCHAR(MAX),
			duration NVARCHAR(MAX),
			player_name NVARCHAR(MAX),
			job NVARCHAR(MAX),
			dps NVARCHAR(MAX),
			healing NVARCHAR(MAX),
			damage_taken NVARCHAR(MAX),
			deaths NVARCHAR(MAX),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`
		createTravelLogTableSQL = `
		IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='travel_log' and xtype='U')
		CREATE TABLE travel_log (
			id INT IDENTITY(1,1) PRIMARY KEY,
			zone_id INT,
			zone_name NVARCHAR(MAX),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`
	default:
		createEncountersTableSQL = `
		CREATE TABLE IF NOT EXISTS encounters (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			encounter_id TEXT,
			encounter_title TEXT,
			duration TEXT,
			player_name TEXT,
			job TEXT,
			dps TEXT,
			healing TEXT,
			damage_taken TEXT,
			deaths TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`
		createTravelLogTableSQL = `
		CREATE TABLE IF NOT EXISTS travel_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			zone_id INTEGER,
			zone_name TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`
	}

	if _, err := db.Exec(createEncountersTableSQL); err != nil {
		return nil, fmt.Errorf("failed to create encounters table: %v", err)
	}
	if _, err := db.Exec(createTravelLogTableSQL); err != nil {
		return nil, fmt.Errorf("failed to create travel log table: %v", err)
	}
	return db, nil
}

func prepareQuery(query string) string {
	if dbDriver == "postgres" {
		for i := 1; strings.Contains(query, "?"); i++ {
			query = strings.Replace(query, "?", fmt.Sprintf("$%d", i), 1)
		}
	}
	return query
}

func insertEncounter(db *sql.DB, encounterID, title, duration, playerName, job, dps, healing, damageTaken, deaths string) error {
	insertSQL := `INSERT INTO encounters (encounter_id, encounter_title, duration, player_name, job, dps, healing, damage_taken, deaths) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	insertSQL = prepareQuery(insertSQL)
	_, err := db.Exec(insertSQL, encounterID, title, duration, playerName, job, dps, healing, damageTaken, deaths)
	return err
}

func insertTravelLog(db *sql.DB, zoneID int, zoneName string) error {
	insertSQL := `INSERT INTO travel_log (zone_id, zone_name) VALUES (?, ?)`
	insertSQL = prepareQuery(insertSQL)
	_, err := db.Exec(insertSQL, zoneID, zoneName)
	return err
}
