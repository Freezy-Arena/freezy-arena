// Copyright 20## Team ###. All Rights Reserved.
// Author: cpapplefamily@gmail.com (Corey Applegate)
//
// Alternate IO handlers for the ###.

package plc

import (
	//"github.com/Team254/cheesy-arena/game"
	//"github.com/Team254/cheesy-arena/model"
	//"encoding/json"
	//"net/http"
	"log"
	"net"
	"strings"
	"time"
)

type Esp32 interface {
	Run()
	IsScoreTableIOEnabled() bool
	IsRedEstopsEnabled() bool
	IsBlueEstopsEnabled() bool
	IsRedHubEnabled() bool
	IsBlueHubEnabled() bool
	IsScoreTableHealthy() bool
	IsRedEstopsHealthy() bool
	IsBlueEstopsHealthy() bool
	IsRedHubHealthy() bool
	IsBlueHubHealthy() bool
	IsScoreTableActive() bool
	IsRedEstopsActive() bool
	IsBlueEstopsActive() bool
	IsRedHubActive() bool
	IsBlueHubActive() bool
	UpdateScoreTableLastSeen()
	UpdateRedEstopsLastSeen()
	UpdateBlueEstopsLastSeen()
	UpdateRedHubLastSeen()
	UpdateBlueHubLastSeen()
	SetScoreTableAddress(string)
	SetRedAllianceStationEstopAddress(string)
	SetBlueAllianceStationEstopAddress(string)
	SetRedAllianceHubAddress(string)
	SetBlueAllianceHubAddress(string)
	// Hub battery status
	GetRedHubBatteryVoltage() float64
	GetRedHubBatteryPercent() float64
	GetBlueHubBatteryVoltage() float64
	GetBlueHubBatteryPercent() float64
	SetRedHubBattery(voltage, percent float64)
	SetBlueHubBattery(voltage, percent float64)
}

type Esp32IO struct {
	ScoreTableIP         string
	RedAllianceEstopsIP  string
	BlueAllianceEstopsIP string
	RedAllianceHubIP     string
	BlueAllianceHubIP    string
	scoreTableHealthy    bool
	RedEstopsHealthy     bool
	BlueEstopsHealthy    bool
	RedHubHealthy        bool
	BlueHubHealthy       bool
	// Timestamps for tracking when each module last called its API
	ScoreTableLastSeen   time.Time
	RedEstopsLastSeen    time.Time
	BlueEstopsLastSeen   time.Time
	RedHubLastSeen       time.Time
	BlueHubLastSeen      time.Time
	// Hub battery status
	RedHubBatteryVoltage  float64
	RedHubBatteryPercent  float64
	BlueHubBatteryVoltage float64
	BlueHubBatteryPercent float64
}
const LoopPeriodMs = 1000 // Define the loop period in milliseconds


// RequestPayload represents the structure of the incoming POST data.
type RequestPayload struct {
	Channel int  `json:"channel"`
	State   bool `json:"state"`
}

func (esp32 *Esp32IO) SetScoreTableAddress(address string) {
	address = strings.TrimSpace(address)
	if address == "" {
		esp32.ScoreTableIP = address
        return
    }
    if net.ParseIP(address) == nil {
        log.Printf("Invalid Score Table IP address: %s", address)
        return
    }
    esp32.ScoreTableIP = address
    log.Printf("Set Score Table IP to: %s", esp32.ScoreTableIP)
}
func (esp32 *Esp32IO) SetRedAllianceStationEstopAddress(address string) {
	address = strings.TrimSpace(address)
	if address == "" {
		esp32.RedAllianceEstopsIP = address
        return
    }
    if net.ParseIP(address) == nil {
        log.Printf("Invalid Red Alliance Estops IP address: %s", address)
        return
    }
    esp32.RedAllianceEstopsIP = address
	log.Printf("Red Alliance Estops IP to: %s", esp32.RedAllianceEstopsIP)
}
func (esp32 *Esp32IO) SetBlueAllianceStationEstopAddress(address string) {
	address = strings.TrimSpace(address)
	if address == "" {
		esp32.BlueAllianceEstopsIP = address
        return
    }
    if net.ParseIP(address) == nil {
        log.Printf("Invalid Blue Alliance Estops IP address: %s", address)
        return
    }
    esp32.BlueAllianceEstopsIP = address
	log.Printf("Blue Alliance Estops IP to: %s", esp32.BlueAllianceEstopsIP)
}
func (esp32 *Esp32IO) SetBlueAllianceHubAddress(address string) {
	address = strings.TrimSpace(address)
	if address == "" {
		esp32.BlueAllianceHubIP = address
        return
    }
    if net.ParseIP(address) == nil {
        log.Printf("Invalid Blue Alliance Hub IP address: %s", address)
        return
    }
    esp32.BlueAllianceHubIP = address
	log.Printf("Blue Alliance Hub IP to: %s", esp32.BlueAllianceHubIP)
}
func (esp32 *Esp32IO) SetRedAllianceHubAddress(address string) {
	address = strings.TrimSpace(address)
	if address == "" {
		esp32.RedAllianceHubIP = address
        return
    }
    if net.ParseIP(address) == nil {
        log.Printf("Invalid Red Alliance Hub IP address: %s", address)
        return
    }
    esp32.RedAllianceHubIP = address
	log.Printf("Red Alliance Hub IP to: %s", esp32.RedAllianceHubIP)
}
// Checks if an IP address is reachable by attempting a TCP connection.
func isDevicePresent(ip string, port string) error {
    address := net.JoinHostPort(ip, port)
    conn, err := net.DialTimeout("tcp", address, time.Second*2)
    if err != nil {
        //log.Printf("Device not reachable at %s: %v", address, err)
        return err
    } 
    conn.Close()
    return err
}

// Run starts the ESP32 IO monitoring loop.
func (esp32 *Esp32IO) Run() {
	for {
		// Check if the Score Table Estops are reachable.
		if !esp32.IsScoreTableIOEnabled() {
			// If the Score Table is not enabled, don't check it.
			esp32.scoreTableHealthy = false
		} else {
			//log.Println("ScoreTable Check")
			err := isDevicePresent(esp32.ScoreTableIP, "80")
			if err != nil {
				log.Printf("Score Table not reachable at %s: %v", esp32.ScoreTableIP, err)
				time.Sleep(time.Second * plcRetryIntevalSec)
				esp32.scoreTableHealthy = false
				continue
				}else{
					if (!esp32.scoreTableHealthy){
						log.Printf("Score Table Connected at: %s", esp32.ScoreTableIP)
					}
					esp32.scoreTableHealthy = true
				}
			}
			// Check if the Red Alliance Estops are healthy.
			if !esp32.IsRedEstopsEnabled() {
				// If the Red Alliance Estops are not enabled, don't check them.
				esp32.RedEstopsHealthy= false
				} else {
			//log.Println("Red Estops IO Check")
			err := isDevicePresent(esp32.RedAllianceEstopsIP, "80")
			if err != nil {
				log.Printf("Red Alliance Estops not reachable at %s: %v", esp32.RedAllianceEstopsIP, err)
				time.Sleep(time.Second * plcRetryIntevalSec)
				esp32.RedEstopsHealthy = false
				continue
				}else{
					if (!esp32.RedEstopsHealthy){
						log.Printf("Red Estops Connected at: %s ", esp32.RedAllianceEstopsIP)
					}
					esp32.RedEstopsHealthy = true
				}
			}
			// Check if the Blue Alliance Estops are healthy.
			if !esp32.IsBlueEstopsEnabled() {
				// If the Blue Alliance Estops are not enabled, don't check them.
				esp32.BlueEstopsHealthy = false
				} else {
			//log.Println("Blue Estops IO Check")
			err := isDevicePresent(esp32.BlueAllianceEstopsIP, "80")
			if err != nil {
				log.Printf("Blue Alliance Estops not reachable at %s: %v", esp32.BlueAllianceEstopsIP, err)
				time.Sleep(time.Second * plcRetryIntevalSec)
				esp32.BlueEstopsHealthy = false
				continue
			}else{
				if (!esp32.BlueEstopsHealthy){
					log.Printf("Blue Estops Connected at: %s ", esp32.BlueAllianceEstopsIP)
				}
				esp32.BlueEstopsHealthy = true
			}
		}
			// Check if the Red Alliance Hub is healthy.
			if !esp32.IsRedHubEnabled() {
				// If the Red Alliance Hub are not enabled, don't check them.
				esp32.RedHubHealthy= false
				} else {
			//log.Println("Red Hub IO Check")
			err := isDevicePresent(esp32.RedAllianceHubIP, "80")
			if err != nil {
				log.Printf("Red Alliance Hub not reachable at %s: %v", esp32.RedAllianceHubIP, err)
				time.Sleep(time.Second * plcRetryIntevalSec)
				esp32.RedHubHealthy = false
				continue
				}else{
					if (!esp32.RedHubHealthy){
						log.Printf("Red Hub Connected at: %s ", esp32.RedAllianceHubIP)
					}
					esp32.RedHubHealthy = true
				}
			}		
			// Check if the Blue Alliance Hub is healthy.
			if !esp32.IsBlueHubEnabled() {
				// If the Blue Alliance Hub are not enabled, don't check them.
				esp32.BlueHubHealthy= false
				} else {
			//log.Println("Blue Hub IO Check")
			err := isDevicePresent(esp32.BlueAllianceHubIP, "80")
			if err != nil {
				log.Printf("Blue Alliance Hub not reachable at %s: %v", esp32.BlueAllianceHubIP, err)
				time.Sleep(time.Second * plcRetryIntevalSec)
				esp32.BlueHubHealthy = false
				continue
				}else{
					if (!esp32.BlueHubHealthy){
						log.Printf("Blue Hub Connected at: %s ", esp32.BlueAllianceHubIP)
					}
					esp32.BlueHubHealthy = true
				}
			}		

		startTime := time.Now()
		time.Sleep(time.Until(startTime.Add(time.Millisecond * LoopPeriodMs)))
	}
}

// Returns whether the alternate IO is enabled.
func (esp32 *Esp32IO) IsScoreTableIOEnabled() bool {
	return esp32.ScoreTableIP != ""
}

// Returns whether the alternate IO is enabled.
func (esp32 *Esp32IO) IsRedEstopsEnabled() bool {
	return esp32.RedAllianceEstopsIP != ""
}

// Returns whether the alternate IO is enabled.
func (esp32 *Esp32IO) IsBlueEstopsEnabled() bool {
	return esp32.BlueAllianceEstopsIP != ""
}
// Returns whether the alternate IO is enabled.
func (esp32 *Esp32IO) IsBlueHubEnabled() bool {
	return esp32.BlueAllianceHubIP != ""
}
// Returns whether the alternate IO is enabled.
func (esp32 *Esp32IO) IsRedHubEnabled() bool {
	return esp32.RedAllianceHubIP != ""
}

// Returns the health status of the alternate IO.
func (esp32 *Esp32IO) IsScoreTableHealthy() bool {
	return esp32.scoreTableHealthy
}

// Returns the health status of the alternate IO.
func (esp32 *Esp32IO) IsRedEstopsHealthy() bool {
	return esp32.RedEstopsHealthy
}

// Returns the health status of the alternate IO.
func (esp32 *Esp32IO) IsBlueEstopsHealthy() bool {
	return esp32.BlueEstopsHealthy
}

// Returns the health status of the alternate IO.
func (esp32 *Esp32IO) IsRedHubHealthy() bool {
	return esp32.RedHubHealthy
}

// Returns the health status of the alternate IO.
func (esp32 *Esp32IO) IsBlueHubHealthy() bool {
	return esp32.BlueHubHealthy
}

// Activity timeout for determining if a module is still actively calling the API.
const ModuleActivityTimeoutSec = 2

// Updates the last seen timestamp for the Score Table module.
func (esp32 *Esp32IO) UpdateScoreTableLastSeen() {
	esp32.ScoreTableLastSeen = time.Now()
}

// Updates the last seen timestamp for the Red Estops module.
func (esp32 *Esp32IO) UpdateRedEstopsLastSeen() {
	esp32.RedEstopsLastSeen = time.Now()
}

// Updates the last seen timestamp for the Blue Estops module.
func (esp32 *Esp32IO) UpdateBlueEstopsLastSeen() {
	esp32.BlueEstopsLastSeen = time.Now()
}

// Updates the last seen timestamp for the Red Hub module.
func (esp32 *Esp32IO) UpdateRedHubLastSeen() {
	esp32.RedHubLastSeen = time.Now()
}

// Updates the last seen timestamp for the Blue Hub module.
func (esp32 *Esp32IO) UpdateBlueHubLastSeen() {
	esp32.BlueHubLastSeen = time.Now()
}

// Returns whether the Score Table module is actively calling the API.
func (esp32 *Esp32IO) IsScoreTableActive() bool {
	return time.Since(esp32.ScoreTableLastSeen).Seconds() < ModuleActivityTimeoutSec
}

// Returns whether the Red Estops module is actively calling the API.
func (esp32 *Esp32IO) IsRedEstopsActive() bool {
	return time.Since(esp32.RedEstopsLastSeen).Seconds() < ModuleActivityTimeoutSec
}

// Returns whether the Blue Estops module is actively calling the API.
func (esp32 *Esp32IO) IsBlueEstopsActive() bool {
	return time.Since(esp32.BlueEstopsLastSeen).Seconds() < ModuleActivityTimeoutSec
}

// Returns whether the Red Hub module is actively calling the API.
func (esp32 *Esp32IO) IsRedHubActive() bool {
	return time.Since(esp32.RedHubLastSeen).Seconds() < ModuleActivityTimeoutSec
}

// Returns whether the Blue Hub module is actively calling the API.
func (esp32 *Esp32IO) IsBlueHubActive() bool {
	return time.Since(esp32.BlueHubLastSeen).Seconds() < ModuleActivityTimeoutSec
}

// Returns the Red Hub battery voltage.
func (esp32 *Esp32IO) GetRedHubBatteryVoltage() float64 {
	return esp32.RedHubBatteryVoltage
}

// Returns the Red Hub battery percent.
func (esp32 *Esp32IO) GetRedHubBatteryPercent() float64 {
	return esp32.RedHubBatteryPercent
}

// Returns the Blue Hub battery voltage.
func (esp32 *Esp32IO) GetBlueHubBatteryVoltage() float64 {
	return esp32.BlueHubBatteryVoltage
}

// Returns the Blue Hub battery percent.
func (esp32 *Esp32IO) GetBlueHubBatteryPercent() float64 {
	return esp32.BlueHubBatteryPercent
}

// Sets the Red Hub battery status.
func (esp32 *Esp32IO) SetRedHubBattery(voltage, percent float64) {
	esp32.RedHubBatteryVoltage = voltage
	esp32.RedHubBatteryPercent = percent
}

// Sets the Blue Hub battery status.
func (esp32 *Esp32IO) SetBlueHubBattery(voltage, percent float64) {
	esp32.BlueHubBatteryVoltage = voltage
	esp32.BlueHubBatteryPercent = percent
}