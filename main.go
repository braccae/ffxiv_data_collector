package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

type PlayerStats struct {
	Job         string
	Damage      int64
	Healed      int64
	DamageTaken int64
	Deaths      int64
}

func parseInt(s string) int64 {
	s = strings.ReplaceAll(s, ",", "")
	f, _ := strconv.ParseFloat(s, 64)
	return int64(f)
}

type SubscribeMessage struct {
	Call   string   `json:"call"`
	Events []string `json:"events"`
	Key    string   `json:"key,omitempty"`
	Data   []string `json:"data,omitempty"`
}

func main() {
	debug := flag.Bool("debug", false, "Print raw events")
	portable := flag.Bool("portable", false, "Use config from local directory")
	flag.Parse()

	cfg, err := loadConfig(*portable)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	var db *sql.DB
	db, err = initDB(&cfg.Database)
	if err != nil {
		log.Printf("Warning: Failed to initialize database on startup: %v. Will retry when needed.", err)
		db = nil
	}
	defer func() {
		if db != nil {
			db.Close()
		}
	}()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	u := cfg.WebSocketURL
	done := make(chan struct{})

	go func() {
		defer close(done)
		wasActive := false
		lastZoneName := ""
		
		var zonePlayers map[string]*PlayerStats
		var zoneDuration int64
		var zoneEncounterID string

		for {
			log.Printf("Connecting to %s", u)
			c, _, err := websocket.DefaultDialer.Dial(u, nil)
			if err != nil {
				log.Printf("Dial error: %v", err)
				if wasActive {
					time.Sleep(10 * time.Second)
				} else {
					time.Sleep(1 * time.Minute)
				}
				continue
			}

			// Send subscribe message
			subMsg := SubscribeMessage{
				Call:   "subscribe",
				Events: []string{"CombatData", "ChangeZone", "PartyChanged"},
				Key:    "events",
				Data:   []string{"CombatData", "ChangeZone", "PartyChanged"},
			}

			err = c.WriteJSON(subMsg)
			if err != nil {
				log.Printf("WriteJSON error: %v", err)
				c.Close()
				if wasActive {
					time.Sleep(10 * time.Second)
				} else {
					time.Sleep(1 * time.Minute)
				}
				continue
			}
			log.Println("Subscribed to events.")

			// Inner loop to read messages
			for {
				_, message, err := c.ReadMessage()
				if err != nil {
					log.Println("Read error:", err)
					c.Close()
					break // Break inner loop to reconnect
				}
				
				var data map[string]interface{}
				if err := json.Unmarshal(message, &data); err != nil {
					log.Printf("Failed to unmarshal message: %v\nMessage: %s", err, string(message))
					continue
				}

				// Try to extract event type from common ACT websocket fields
				eventType := "Unknown"
				if t, ok := data["type"].(string); ok {
					eventType = t
				} else if t, ok := data["msgtype"].(string); ok {
					eventType = t
				}

				// Filter out CombatData with no Combatant before saving anything
				if eventType == "CombatData" {
					combatant, _ := data["Combatant"].(map[string]interface{})
					if len(combatant) == 0 {
						continue
					}
				}

				if *debug {
					log.Printf("Raw Event: %s", string(message))
				}

				if eventType == "ChangeZone" {
					var zoneID int
					if zoneIDFloat, ok := data["zoneID"].(float64); ok {
						zoneID = int(zoneIDFloat)
					}
					
					zoneName, _ := data["zoneName"].(string)
					if zoneName == "" {
						zoneName = ZoneMap[zoneID]
					}
					if zoneName == "" {
						zoneName = "Unknown Zone"
					}
					if zoneName != lastZoneName {
						// Save previous zone's accumulated encounter if we have any
						if len(zonePlayers) > 0 && lastZoneName != "" {
							durationStr := fmt.Sprintf("%02d:%02d", zoneDuration/60, zoneDuration%60)
							for playerName, stats := range zonePlayers {
								dps := "0.00"
								if zoneDuration > 0 {
									dps = fmt.Sprintf("%.2f", float64(stats.Damage)/float64(zoneDuration))
								}
								err = insertEncounter(&db, &cfg.Database, zoneEncounterID, lastZoneName, durationStr, playerName, stats.Job, dps, fmt.Sprintf("%d", stats.Healed), fmt.Sprintf("%d", stats.DamageTaken), fmt.Sprintf("%d", stats.Deaths))
								if err != nil {
									log.Printf("Failed to insert accumulated encounter for %s: %v", playerName, err)
								}
							}
						}

						err = insertTravelLog(&db, &cfg.Database, zoneID, zoneName)
						if err != nil {
							log.Printf("Failed to insert travel log: %v", err)
						}
						
						lastZoneName = zoneName
						zoneEncounterID = fmt.Sprintf("%d", time.Now().UnixNano())
						zonePlayers = make(map[string]*PlayerStats)
						zoneDuration = 0
					}
				} else if eventType == "CombatData" {
					isActiveRaw := data["isActive"]
					isActive := false
					if isActiveRaw == "true" || isActiveRaw == true {
						isActive = true
					}

					if !isActive && wasActive {
						// Combat just ended
						wasActive = false

						encounter, _ := data["Encounter"].(map[string]interface{})
						combatant, _ := data["Combatant"].(map[string]interface{})

						// add to zoneDuration
						durationRaw, _ := encounter["DURATION"].(string)
						durationSec, _ := strconv.ParseInt(durationRaw, 10, 64)
						if durationSec == 0 {
							// fallback: parse "duration" string like "01:23"
							durStr, _ := encounter["duration"].(string)
							parts := strings.Split(durStr, ":")
							if len(parts) == 2 {
								m, _ := strconv.ParseInt(parts[0], 10, 64)
								s, _ := strconv.ParseInt(parts[1], 10, 64)
								durationSec = m*60 + s
							}
						}
						zoneDuration += durationSec

						if zonePlayers == nil {
							zonePlayers = make(map[string]*PlayerStats)
						}

						for playerName, playerData := range combatant {
							pData, ok := playerData.(map[string]interface{})
							if !ok {
								continue
							}

							jobRaw, _ := pData["Job"].(string)
							jobName := JobMap[jobRaw]
							if jobName == "" {
								jobName = jobRaw
							}

							damageStr, _ := pData["damage"].(string)
							healedStr, _ := pData["healed"].(string)
							damageTakenStr, _ := pData["damagetaken"].(string)
							deathsStr, _ := pData["deaths"].(string)

							damage := parseInt(damageStr)
							healed := parseInt(healedStr)
							damageTaken := parseInt(damageTakenStr)
							deaths := parseInt(deathsStr)

							stats, exists := zonePlayers[playerName]
							if !exists {
								stats = &PlayerStats{Job: jobName}
								zonePlayers[playerName] = stats
							}
							
							if jobName != "" {
								stats.Job = jobName
							}

							stats.Damage += damage
							stats.Healed += healed
							stats.DamageTaken += damageTaken
							stats.Deaths += deaths
						}
					} else if isActive {
						// Combat is currently active
						wasActive = true
					}
				}
			}
			
			// Disconnected from websocket
			log.Println("Disconnected. Attempting to reconnect...")
			if wasActive {
				time.Sleep(10 * time.Second)
			} else {
				time.Sleep(1 * time.Minute)
			}
		}
	}()

	for {
		select {
		case <-done:
			return
		case <-interrupt:
			log.Println("Interrupt received, exiting...")
			return
		}
	}
}
