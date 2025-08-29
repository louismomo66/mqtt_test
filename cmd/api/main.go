package main

import (
	"fmt"
	"log"
	"mqtt/data"
	"net/http"
)

var MQTT *MQTTClient

func main() {
	// Direct database connection string - use Render's managed PostgreSQL
	dsn := "postgresql://mqtt_example_database_user:IBgqXOjxSq9IO8FJMmihrWQBqh2gIX3U@dpg-d2ooacidbo4c73brk330-a/mqtt_example_database"
	fmt.Printf("Using database connection: %s\n", dsn)

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

	// Setup HTTP server with routes
	apiHandler := NewAPIHandler(models)
	router := apiHandler.SetupRoutes()

	// Start HTTP server
	server := &http.Server{
		Addr:    ":9005",
		Handler: router,
	}

	fmt.Printf("Starting HTTP server on port 9005...\n")
	fmt.Printf("Health check: http://localhost:9005/health\n")
	fmt.Printf("API documentation:\n")
	fmt.Printf("  GET  /api/v1/devices/                    - Get all devices\n")
	fmt.Printf("  POST /api/v1/devices/                    - Create a device\n")
	fmt.Printf("  GET  /api/v1/devices/{id}                - Get device by ID\n")
	fmt.Printf("  PUT  /api/v1/devices/{id}                - Update device\n")
	fmt.Printf("  DELETE /api/v1/devices/{id}              - Delete device\n")
	fmt.Printf("  GET  /api/v1/devices/{id}/logs           - Get device logs\n")
	fmt.Printf("  GET  /api/v1/devices/{id}/logs/latest    - Get latest device log\n")
	fmt.Printf("  GET  /api/v1/devices/serial/{serial}     - Get device by serial number\n")
	fmt.Printf("  GET  /api/v1/devices/serial/{serial}/logs - Get device logs by serial\n")
	fmt.Printf("  GET  /api/v1/logs/imei/{imei}            - Get logs by IMEI\n")
	fmt.Printf("  GET  /api/v1/logs/serial/{serial}        - Get logs by serial number\n")

	// Start server in a goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Keep the application running
	select {}
}
