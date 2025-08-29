package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"mqtt/data"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	clientID = "devices_api_render"
	topic    = "device/logs"
)

// MQTT Topics
const (
	mqttTopicLED  = "led_control"
	mqttTopicData = "sensor_data"
)

var brokerURL = "tcp://localhost:1883"
var mqttUser = "hassan"
var mqttPass = "ha55an"

type MQTTClient struct {
	client     mqtt.Client
	topicRoot  string
	bufferSize int
	models     *data.Models
}

// Message buffer for reassembling multi-part messages
type messageBuffer struct {
	Parts        map[int][]byte
	TotalParts   int
	ReceivedTime time.Time
	IsComplete   bool
}

// Map to store message buffers by device serial number
var messageBuffers = make(map[string]*messageBuffer)

func NewMQTTClient(models *data.Models) (*MQTTClient, error) {
	// Connect to external MQTT server
	mqttBroker := "tcp://157.230.113.253:1883"

	opts := mqtt.NewClientOptions()
	opts.AddBroker(mqttBroker)
	opts.SetClientID(clientID)
	opts.SetUsername("hassan")
	opts.SetPassword("ha55an")
	opts.SetKeepAlive(20 * time.Second)  // More frequent keep-alive
	opts.SetPingTimeout(5 * time.Second) // Shorter ping timeout
	opts.SetConnectTimeout(20 * time.Second)
	opts.SetCleanSession(false) // Keep session to avoid re-subscription issues
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(30 * time.Second) // Faster reconnection
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(3 * time.Second) // Faster retry
	opts.SetResumeSubs(true)                      // Resume subscriptions after reconnect
	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		fmt.Printf("MQTT connection lost: %v", err)
	})

	opts.SetOnConnectHandler(func(client mqtt.Client) {
		fmt.Printf("MQTT connection established successfully")
	})

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("failed to connect: %v", token.Error())
	}

	return &MQTTClient{
		client:     client,
		topicRoot:  topic,
		bufferSize: 4096,
		models:     models,
	}, nil
}

func (m *MQTTClient) Subscribe(topic string, handler mqtt.MessageHandler) error {
	if token := m.client.Subscribe(topic, 0, handler); token.Wait() && token.Error() != nil {
		return fmt.Errorf("subscribe error: %v", token.Error())
	}
	return nil
}

func (m *MQTTClient) Publish(topic string, payload interface{}) error {
	if token := m.client.Publish(topic, 0, false, payload); token.Wait() && token.Error() != nil {
		return fmt.Errorf("publish error: %v", token.Error())
	}
	return nil
}
func (m *MQTTClient) IsConnected() bool {
	return m.client != nil && m.client.IsConnected()
}

// CloseConnection gracefully closes the MQTT connection
func (m *MQTTClient) CloseConnection() {
	if m.client.IsConnected() {
		m.client.Disconnect(250) // Wait 250ms for graceful disconnect
	}
}

// parseDeviceData parses the URL-encoded device data format
func parseDeviceData(rawData string) (*data.DeviceData, error) {
	// Remove any leading/trailing whitespace
	rawData = strings.TrimSpace(rawData)

	// Parse the URL-encoded data
	values, err := url.ParseQuery(rawData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL data: %v", err)
	}

	deviceData := &data.DeviceData{
		Timestamp: time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Parse basic fields
	if imei := values.Get("imei"); imei != "" {
		deviceData.IMEI = imei
		deviceData.SerialNumber = imei // Use IMEI as serial number
	}

	if token := values.Get("tkn"); token != "" {
		deviceData.Token = token
	}

	// Parse numeric fields
	if sv := values.Get("sv"); sv != "" {
		if val, err := strconv.ParseFloat(sv, 64); err == nil {
			deviceData.SupplyVoltage = val
		}
	}

	if sc := values.Get("sc"); sc != "" {
		if val, err := strconv.ParseFloat(sc, 64); err == nil {
			deviceData.SupplyCurrent = val
		}
	}

	if bv := values.Get("bv"); bv != "" {
		if val, err := strconv.ParseFloat(bv, 64); err == nil {
			deviceData.BatteryVoltage = val
		}
	}

	if pv := values.Get("pv"); pv != "" {
		if val, err := strconv.ParseFloat(pv, 64); err == nil {
			deviceData.PanelVoltage = val
		}
	}

	if pc := values.Get("pc"); pc != "" {
		if val, err := strconv.ParseFloat(pc, 64); err == nil {
			deviceData.PanelCurrent = val
		}
	}

	if tempRoom := values.Get("temp_room"); tempRoom != "" {
		if val, err := strconv.ParseFloat(tempRoom, 64); err == nil {
			deviceData.TempRoom = val
		}
	}

	if tempBattery := values.Get("temp_battery"); tempBattery != "" {
		if val, err := strconv.ParseFloat(tempBattery, 64); err == nil {
			deviceData.TempBattery = val
		}
	}

	// Parse the embedded JSON data in the 'e' field
	if embeddedData := values.Get("e"); embeddedData != "" {
		// Extract the JSON part from the embedded data
		// Format: {NwS:value,SD:value,la:value,lo:value,D:value,Hs:value,Hm:value,Dc:value,DHc:value,DSc:value,Fv:'value',S1:[value,value],S2:[value,value],S3:[value,value]}value

		// Find the start and end of the JSON object
		start := strings.Index(embeddedData, "{")
		end := strings.LastIndex(embeddedData, "}")

		if start != -1 && end != -1 && end > start {
			jsonStr := embeddedData[start : end+1]

			// Parse the embedded JSON-like structure
			// Note: This is not standard JSON, so we need to parse it manually
			parseEmbeddedData(jsonStr, deviceData)

			// Parse the main loop count after the JSON
			if end+1 < len(embeddedData) {
				loopCountStr := embeddedData[end+1:]
				if val, err := strconv.Atoi(loopCountStr); err == nil {
					deviceData.MainLoopCount = val
				}
			}
		}
	}

	return deviceData, nil
}

// parseEmbeddedData parses the embedded JSON-like structure
func parseEmbeddedData(jsonStr string, deviceData *data.DeviceData) {
	// Remove the outer braces
	jsonStr = strings.Trim(jsonStr, "{}")

	// Split by comma, but be careful with nested arrays
	parts := splitEmbeddedData(jsonStr)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if !strings.Contains(part, ":") {
			continue
		}

		colonIndex := strings.Index(part, ":")
		key := strings.TrimSpace(part[:colonIndex])
		value := strings.TrimSpace(part[colonIndex+1:])

		switch key {
		case "NwS":
			deviceData.NetworkStrength = value
		case "SD":
			if val, err := strconv.Atoi(value); err == nil {
				deviceData.SDLogStatus = val
			}
		case "la":
			if val, err := strconv.ParseFloat(value, 64); err == nil {
				deviceData.Latitude = val
			}
		case "lo":
			if val, err := strconv.ParseFloat(value, 64); err == nil {
				deviceData.Longitude = val
			}
		case "D":
			if val, err := strconv.Atoi(value); err == nil {
				deviceData.DoorOpenCounter = val
			}
		case "Hs":
			if val, err := strconv.ParseFloat(value, 64); err == nil {
				deviceData.Humidity = val
			}
		case "Dc":
			if val, err := strconv.Atoi(value); err == nil {
				deviceData.IsDoorSense = val
			}
		case "DHc":
			if val, err := strconv.Atoi(value); err == nil {
				deviceData.IsDs8 = val
			}
		case "DSc":
			if val, err := strconv.Atoi(value); err == nil {
				deviceData.IsDHT22 = val
			}
		case "Fv":
			// Remove quotes from firmware version
			deviceData.FirmwareVersion = strings.Trim(value, "'")
		case "S1":
			sensorArray := parseSensorArray(value)
			if sensorJSON, err := json.Marshal(sensorArray); err == nil {
				deviceData.Sensor1 = string(sensorJSON)
			}
		case "S2":
			sensorArray := parseSensorArray(value)
			if sensorJSON, err := json.Marshal(sensorArray); err == nil {
				deviceData.Sensor2 = string(sensorJSON)
			}
		case "S3":
			sensorArray := parseSensorArray(value)
			if sensorJSON, err := json.Marshal(sensorArray); err == nil {
				deviceData.Sensor3 = string(sensorJSON)
			}
		}
	}
}

// splitEmbeddedData splits the embedded data by comma, respecting nested arrays
func splitEmbeddedData(data string) []string {
	var parts []string
	var currentPart strings.Builder
	bracketCount := 0

	for _, char := range data {
		if char == '[' {
			bracketCount++
		} else if char == ']' {
			bracketCount--
		} else if char == ',' && bracketCount == 0 {
			parts = append(parts, currentPart.String())
			currentPart.Reset()
			continue
		}
		currentPart.WriteRune(char)
	}

	if currentPart.Len() > 0 {
		parts = append(parts, currentPart.String())
	}

	return parts
}

// parseSensorArray parses sensor array values like [0.00,0.00]
func parseSensorArray(arrayStr string) [2]float64 {
	var result [2]float64

	// Remove brackets
	arrayStr = strings.Trim(arrayStr, "[]")

	// Split by comma
	parts := strings.Split(arrayStr, ",")

	for i, part := range parts {
		if i >= 2 {
			break
		}
		part = strings.TrimSpace(part)
		if val, err := strconv.ParseFloat(part, 64); err == nil {
			result[i] = val
		}
	}

	return result
}

// handleDeviceData processes incoming device data messages
func (m *MQTTClient) handleDeviceData(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("Received MQTT message on topic: %s\n", msg.Topic())
	fmt.Printf("Message payload: %s\n", string(msg.Payload()))

	// Try to parse as URL-encoded data first
	deviceData, err := parseDeviceData(string(msg.Payload()))
	if err != nil {
		fmt.Printf("URL parsing failed, trying JSON: %v\n", err)
		// If URL parsing fails, try JSON parsing
		if err := json.Unmarshal(msg.Payload(), &deviceData); err != nil {
			fmt.Printf("JSON parsing also failed: %v\n", err)
			fmt.Printf("Raw message: %s\n", string(msg.Payload()))
			return
		}
	}

	fmt.Printf("Successfully parsed device data from IMEI: %s\n", deviceData.IMEI)

	if err := m.processDeviceData(deviceData); err != nil {
		fmt.Printf("Error processing device data: %v", err)
	}
}

// handleLEDControl processes LED control messages
func (m *MQTTClient) handleLEDControl(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("Received MQTT message on topic: %s\n", msg.Topic())
	fmt.Printf("LED control message payload: %s\n", string(msg.Payload()))
	// TODO: Implement LED control processing
	// You can add logic here to control LEDs based on the message content
}

// processDeviceData processes device data and saves to database
func (m *MQTTClient) processDeviceData(logEntry *data.DeviceData) error {
	// Check if the device exists
	device, err := m.models.Device.GetBySerialNumber(logEntry.SerialNumber)
	if err != nil {
		// Auto-register the device
		device = &data.Device{
			DeviceType:   "auto_registered",
			SerialNumber: logEntry.SerialNumber,
		}
		if err := m.models.Device.CreateDevice(device); err != nil {
			return fmt.Errorf("failed to auto-register device: %v", err)
		}
		fmt.Printf("Auto-registered device: %s", logEntry.SerialNumber)
	}

	// Link the log entry to the device
	logEntry.DeviceID = device.ID

	// Save the log entry
	if err := m.models.DeviceData.CreateLog(logEntry); err != nil {
		return fmt.Errorf("failed to save device data: %v", err)
	}

	fmt.Printf("Successfully logged data for device: %s", logEntry.SerialNumber)
	return nil
}
func (m *MQTTClient) StartDeviceDataListener() error {
	// Subscribe to sensor data topic
	if err := m.Subscribe(mqttTopicData, m.handleDeviceData); err != nil {
		return fmt.Errorf("failed to subscribe to topic %s: %v", mqttTopicData, err)
	}
	fmt.Printf("MQTT client subscribed to topic: %s\n", mqttTopicData)

	// Add a small delay between subscriptions to avoid overwhelming the connection
	time.Sleep(1 * time.Second)

	// Subscribe to LED control topic
	if err := m.Subscribe(mqttTopicLED, m.handleLEDControl); err != nil {
		// Log the error but don't fail the entire process
		fmt.Printf("Warning: Failed to subscribe to topic %s: %v\n", mqttTopicLED, err)
		fmt.Printf("Continuing with sensor data subscription only...\n")
	} else {
		fmt.Printf("MQTT client subscribed to topic: %s\n", mqttTopicLED)
	}

	// Start a goroutine to clean up stale message buffers
	go m.cleanupStaleBuffers()

	// Start a goroutine to monitor MQTT connection health
	go m.monitorConnection()

	return nil
}

func (m *MQTTClient) cleanupStaleBuffers() {
	ticker := time.NewTicker(10 * time.Minute)
	for range ticker.C {
		now := time.Now()
		for serialNumber, buffer := range messageBuffers {
			// If buffer is older than 1 hour and not complete, remove it
			if now.Sub(buffer.ReceivedTime) > time.Hour && !buffer.IsComplete {
				fmt.Printf("Cleaning up stale message buffer for device %s\n", serialNumber)
				delete(messageBuffers, serialNumber)
			}

			// If buffer is complete and older than 5 minutes, remove it
			if buffer.IsComplete && now.Sub(buffer.ReceivedTime) > 5*time.Minute {
				delete(messageBuffers, serialNumber)
			}
		}
	}
}

func (m *MQTTClient) monitorConnection() {
	ticker := time.NewTicker(30 * time.Second)
	for range ticker.C {
		if m.client != nil {
			if m.client.IsConnected() {
				fmt.Printf("MQTT connection status: CONNECTED\n")
			} else {
				fmt.Printf("MQTT connection status: DISCONNECTED - attempting to reconnect...\n")
			}
		} else {
			fmt.Printf("MQTT client is nil\n")
		}
	}
}
