package main

import (
	"fmt"
	"log"
	"mqtt/data"
	"os"
)

var MQTT *MQTTClient

func main() {
	// Get database connection string from environment variable
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		dsn = "postgresql://mqtt_user:mqtt_password@localhost:5432/mqtt_db?sslmode=disable"
	}

	database, err := data.NewDatabase(dsn)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize models
	models := data.NewModels(database.DB)

	// Initialize MQTT client with models
	mqttClient, err := NewMQTTClient(models)
	if err != nil {
		fmt.Printf("Warning: Failed to connect to MQTT broker: %v", err)
		fmt.Println("Continuing without MQTT functionality...")
	} else {
		MQTT = mqttClient
		fmt.Println("Connected to MQTT broker successfully")

		// Start the MQTT device data listener
		if err := mqttClient.StartDeviceDataListener(); err != nil {
			fmt.Printf("Failed to start MQTT device data listener: %v", err)
		} else {
			fmt.Println("MQTT device data listener started successfully")
		}

		// Defer closing the MQTT connection
		defer mqttClient.CloseConnection()
	}

	// Keep the application running
	select {}
}
