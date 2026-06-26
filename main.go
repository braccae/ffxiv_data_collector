package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

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

	db, err := initDB(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := cfg.WebSocketURL
	log.Printf("Connecting to %s", u)

	c, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		log.Fatalf("Dial error: %v", err)
	}
	defer c.Close()

	// Send subscribe message
	subMsg := SubscribeMessage{
		Call:   "subscribe",
		Events: []string{"CombatData", "ChangeZone", "PartyChanged"},
		Key:    "events",
		Data:   []string{"CombatData", "ChangeZone", "PartyChanged"},
	}

	err = c.WriteJSON(subMsg)
	if err != nil {
		log.Fatalf("WriteJSON error: %v", err)
	}
	log.Println("Subscribed to events.")

	done := make(chan struct{})

	go func() {
		defer close(done)
		wasActive := false
		lastZoneName := ""

		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("Read error:", err)
				return
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
					err = insertTravelLog(db, zoneID, zoneName)
					if err != nil {
						log.Printf("Failed to insert travel log: %v", err)
					} else {
						lastZoneName = zoneName
					}
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

					title, _ := encounter["title"].(string)
					duration, _ := encounter["duration"].(string)

					encounterID := fmt.Sprintf("%d", time.Now().UnixNano())

					for playerName, playerData := range combatant {
						pData, ok := playerData.(map[string]interface{})
						if !ok {
							continue
						}

						jobRaw, _ := pData["Job"].(string)
						jobName := JobMap[jobRaw]
						if jobName == "" {
							// Try to use the raw job if it's not empty, or leave as is
							jobName = jobRaw
						}

						dps, _ := pData["encdps"].(string)
						healing, _ := pData["healed"].(string)
						damageTaken, _ := pData["damagetaken"].(string)
						deaths, _ := pData["deaths"].(string)

						err = insertEncounter(db, encounterID, title, duration, playerName, jobName, dps, healing, damageTaken, deaths)
						if err != nil {
							log.Printf("Failed to insert encounter for %s: %v", playerName, err)
						}
					}
				} else if isActive {
					// Combat is currently active
					wasActive = true
				}
			}
		}
	}()

	for {
		select {
		case <-done:
			return
		case <-interrupt:
			log.Println("Interrupt received, closing connection...")
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("Write close error:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}
