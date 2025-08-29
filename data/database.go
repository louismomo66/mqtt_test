package data

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database represents the database connection and models
type Database struct {
	DB *gorm.DB
}

// NewDatabase creates a new database connection
func NewDatabase(dsn string) (*Database, error) {
	var db *gorm.DB
	var err error
	attempt := 0
	// Try to connect up to 5 times with exponential backoff
	for {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})
		if err == nil {
			break // Successfully connected
		}

		if attempt == 5 {
			return nil, fmt.Errorf("failed to connect to database after 5 attempts: %v", err)
		}

		fmt.Printf("Database connection attempt %d failed, retrying in 1 second...\n", attempt)

		time.Sleep(1 * time.Second)
		attempt++
	}

	// Auto migrate the schema
	if err := db.AutoMigrate(&Device{}, &DeviceData{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %v", err)
	}

	return &Database{DB: db}, nil
}

// DeviceDataModel implementation
type DeviceDataModelImpl struct {
	db *gorm.DB
}

func NewDeviceDataModel(db *gorm.DB) DeviceDataModel {
	return &DeviceDataModelImpl{db: db}
}

func (m *DeviceDataModelImpl) CreateLog(logEntry *DeviceData) error {
	// Convert sensor arrays to JSON strings
	if logEntry.Sensor1 == "" {
		sensor1JSON, _ := json.Marshal([2]float64{0, 0})
		logEntry.Sensor1 = string(sensor1JSON)
	}
	if logEntry.Sensor2 == "" {
		sensor2JSON, _ := json.Marshal([2]float64{0, 0})
		logEntry.Sensor2 = string(sensor2JSON)
	}
	if logEntry.Sensor3 == "" {
		sensor3JSON, _ := json.Marshal([2]float64{0, 0})
		logEntry.Sensor3 = string(sensor3JSON)
	}

	return m.db.Create(logEntry).Error
}

func (m *DeviceDataModelImpl) GetByDeviceID(deviceID uint) ([]*DeviceData, error) {
	var logs []*DeviceData
	err := m.db.Where("device_id = ?", deviceID).Order("timestamp DESC").Find(&logs).Error
	return logs, err
}

func (m *DeviceDataModelImpl) GetBySerialNumber(serialNumber string) ([]*DeviceData, error) {
	var logs []*DeviceData
	err := m.db.Where("serial_number = ?", serialNumber).Order("timestamp DESC").Find(&logs).Error
	return logs, err
}

func (m *DeviceDataModelImpl) GetByIMEI(imei string) ([]*DeviceData, error) {
	var logs []*DeviceData
	err := m.db.Where("imei = ?", imei).Order("timestamp DESC").Find(&logs).Error
	return logs, err
}

func (m *DeviceDataModelImpl) GetLatestByDeviceID(deviceID uint) (*DeviceData, error) {
	var logEntry DeviceData
	err := m.db.Where("device_id = ?", deviceID).Order("timestamp DESC").First(&logEntry).Error
	if err != nil {
		return nil, err
	}
	return &logEntry, nil
}

func (m *DeviceDataModelImpl) GetAllLogs() ([]*DeviceData, error) {
	var logs []*DeviceData
	err := m.db.Order("timestamp DESC").Find(&logs).Error
	return logs, err
}

// DeviceModel implementation
type DeviceModelImpl struct {
	db *gorm.DB
}

func NewDeviceModel(db *gorm.DB) DeviceModel {
	return &DeviceModelImpl{db: db}
}

func (m *DeviceModelImpl) CreateDevice(device *Device) error {
	return m.db.Create(device).Error
}

func (m *DeviceModelImpl) GetBySerialNumber(serialNumber string) (*Device, error) {
	var device Device
	err := m.db.Where("serial_number = ?", serialNumber).First(&device).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

func (m *DeviceModelImpl) GetByID(id uint) (*Device, error) {
	var device Device
	err := m.db.First(&device, id).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

func (m *DeviceModelImpl) UpdateDevice(device *Device) error {
	return m.db.Save(device).Error
}

func (m *DeviceModelImpl) GetAllDevices() ([]*Device, error) {
	var devices []*Device
	err := m.db.Find(&devices).Error
	return devices, err
}

func (m *DeviceModelImpl) DeleteDevice(id uint) error {
	return m.db.Delete(&Device{}, id).Error
}

// Models holds all database models
type Models struct {
	Device     DeviceModel
	DeviceData DeviceDataModel
}

// NewModels creates new model instances
func NewModels(db *gorm.DB) *Models {
	return &Models{
		Device:     NewDeviceModel(db),
		DeviceData: NewDeviceDataModel(db),
	}
}
