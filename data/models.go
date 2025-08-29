package data

import (
	"time"

	"gorm.io/gorm"
)

// DeviceData represents the complete device data structure
type DeviceData struct {
	ID           uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	DeviceID     uint      `json:"device_id" gorm:"index"`
	SerialNumber string    `json:"serial_number" gorm:"index;size:50"`
	Timestamp    time.Time `json:"timestamp" gorm:"index"`

	// Device identification
	IMEI  string `json:"imei" gorm:"size:20;index"`
	Token string `json:"token" gorm:"size:50"`

	// Power supply data
	SupplyVoltage  float64 `json:"supply_voltage" gorm:"type:decimal(10,2)"`
	SupplyCurrent  float64 `json:"supply_current" gorm:"type:decimal(10,2)"`
	BatteryVoltage float64 `json:"battery_voltage" gorm:"type:decimal(10,2)"`
	PanelVoltage   float64 `json:"panel_voltage" gorm:"type:decimal(10,2)"`
	PanelCurrent   float64 `json:"panel_current" gorm:"type:decimal(10,2)"`

	// Temperature data
	TempRoom    float64 `json:"temp_room" gorm:"type:decimal(5,2)"`
	TempBattery float64 `json:"temp_battery" gorm:"type:decimal(5,2)"`

	// Environmental data
	Humidity float64 `json:"humidity" gorm:"type:decimal(5,2)"`

	// Device status and configuration
	NetworkStrength string `json:"network_strength" gorm:"size:50"`
	SDLogStatus     int    `json:"sd_log_status"`
	FirmwareVersion string `json:"firmware_version" gorm:"size:20"`
	MainLoopCount   int    `json:"main_loop_count"`

	// Location data
	Latitude  float64 `json:"latitude" gorm:"type:decimal(10,6)"`
	Longitude float64 `json:"longitude" gorm:"type:decimal(10,6)"`

	// Door sensor data
	DoorOpenCounter int `json:"door_open_counter"`
	IsDoorSense     int `json:"is_door_sense"`
	IsDs8           int `json:"is_ds8"`
	IsDHT22         int `json:"is_dht22"`

	// Sensor arrays (S1, S2, S3) - stored as JSON
	Sensor1 string `json:"sensor1" gorm:"type:text"` // JSON string: [value1, value2]
	Sensor2 string `json:"sensor2" gorm:"type:text"` // JSON string: [value1, value2]
	Sensor3 string `json:"sensor3" gorm:"type:text"` // JSON string: [value1, value2]

	// Metadata
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Relationships
	Device Device `json:"device,omitempty" gorm:"foreignKey:DeviceID"`
}

// Device represents a device in the system
type Device struct {
	ID           uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	DeviceType   string         `json:"device_type" gorm:"size:50;index"`
	SerialNumber string         `json:"serial_number" gorm:"size:50;uniqueIndex"`
	Name         string         `json:"name" gorm:"size:100"`
	Description  string         `json:"description" gorm:"size:500"`
	Status       string         `json:"status" gorm:"size:20;default:'active'"`
	CreatedAt    time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt    gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Relationships
	DeviceData []DeviceData `json:"device_data,omitempty" gorm:"foreignKey:DeviceID"`
}

// DeviceDataModel interface for database operations
type DeviceDataModel interface {
	CreateLog(*DeviceData) error
	GetByDeviceID(deviceID uint) ([]*DeviceData, error)
	GetBySerialNumber(serialNumber string) ([]*DeviceData, error)
	GetByIMEI(imei string) ([]*DeviceData, error)
	GetLatestByDeviceID(deviceID uint) (*DeviceData, error)
}

// DeviceModel interface for device database operations
type DeviceModel interface {
	CreateDevice(*Device) error
	GetBySerialNumber(serialNumber string) (*Device, error)
	GetByID(id uint) (*Device, error)
	UpdateDevice(*Device) error
	DeleteDevice(id uint) error
}
