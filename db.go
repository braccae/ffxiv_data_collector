package main

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

func initDB(dbFile string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return nil, err
	}



	createEncountersTableSQL := `
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

	createTravelLogTableSQL := `
	CREATE TABLE IF NOT EXISTS travel_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		zone_id INTEGER,
		zone_name TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`


	if _, err := db.Exec(createEncountersTableSQL); err != nil {
		return nil, err
	}
	if _, err := db.Exec(createTravelLogTableSQL); err != nil {
		return nil, err
	}
	return db, nil
}



func insertEncounter(db *sql.DB, encounterID, title, duration, playerName, job, dps, healing, damageTaken, deaths string) error {
	insertSQL := `INSERT INTO encounters (encounter_id, encounter_title, duration, player_name, job, dps, healing, damage_taken, deaths) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := db.Exec(insertSQL, encounterID, title, duration, playerName, job, dps, healing, damageTaken, deaths)
	return err
}

func insertTravelLog(db *sql.DB, zoneID int, zoneName string) error {
	insertSQL := `INSERT INTO travel_log (zone_id, zone_name) VALUES (?, ?)`
	_, err := db.Exec(insertSQL, zoneID, zoneName)
	return err
}
