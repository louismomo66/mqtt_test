package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"mqtt/data"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// APIHandler handles HTTP API requests
type APIHandler struct {
	models *data.Models
}

// NewAPIHandler creates a new API handler
func NewAPIHandler(models *data.Models) *APIHandler {
	return &APIHandler{models: models}
}

// SetupRoutes configures all the routes
func (h *APIHandler) SetupRoutes() *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS middleware
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Root route
	r.Get("/", h.rootHandler)

	// Health check
	r.Get("/health", h.healthCheck)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Device routes
		r.Route("/devices", func(r chi.Router) {
			r.Get("/", h.getAllDevices)
			r.Post("/", h.createDevice)
			r.Route("/{deviceID}", func(r chi.Router) {
				r.Get("/", h.getDeviceByID)
				r.Put("/", h.updateDevice)
				r.Delete("/", h.deleteDevice)
				r.Get("/logs", h.getDeviceLogs)
				r.Get("/logs/latest", h.getLatestDeviceLog)
			})
			r.Get("/serial/{serialNumber}", h.getDeviceBySerialNumber)
			r.Get("/serial/{serialNumber}/logs", h.getDeviceLogsBySerialNumber)
		})

		// Device logs routes
		r.Route("/logs", func(r chi.Router) {
			r.Get("/", h.getAllLogs)
			r.Get("/imei/{imei}", h.getLogsByIMEI)
			r.Get("/serial/{serialNumber}", h.getLogsBySerialNumber)
		})
	})

	return r
}

// rootHandler returns information about the API
func (h *APIHandler) rootHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"service":   "MQTT Backend API",
		"version":   "1.0.0",
		"status":    "running",
		"timestamp": time.Now().UTC(),
		"endpoints": map[string]string{
			"health": "/health",
			"api":    "/api/v1",
		},
	}
	writeJSON(w, http.StatusOK, response)
}

// healthCheck returns a simple health check response
func (h *APIHandler) healthCheck(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"service":   "mqtt-backend",
	}
	writeJSON(w, http.StatusOK, response)
}

// getAllDevices returns all devices
func (h *APIHandler) getAllDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := h.models.Device.GetAllDevices()
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get devices: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"devices": devices,
		"count":   len(devices),
	})
}

// createDevice creates a new device
func (h *APIHandler) createDevice(w http.ResponseWriter, r *http.Request) {
	var device data.Device
	if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.models.Device.CreateDevice(&device); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create device: %v", err))
		return
	}

	writeJSON(w, http.StatusCreated, device)
}

// getDeviceByID returns a device by ID
func (h *APIHandler) getDeviceByID(w http.ResponseWriter, r *http.Request) {
	deviceIDStr := chi.URLParam(r, "deviceID")
	deviceID, err := strconv.ParseUint(deviceIDStr, 10, 32)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid device ID")
		return
	}

	device, err := h.models.Device.GetByID(uint(deviceID))
	if err != nil {
		writeError(w, http.StatusNotFound, "Device not found")
		return
	}

	writeJSON(w, http.StatusOK, device)
}

// updateDevice updates a device
func (h *APIHandler) updateDevice(w http.ResponseWriter, r *http.Request) {
	deviceIDStr := chi.URLParam(r, "deviceID")
	deviceID, err := strconv.ParseUint(deviceIDStr, 10, 32)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid device ID")
		return
	}

	var device data.Device
	if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	device.ID = uint(deviceID)
	if err := h.models.Device.UpdateDevice(&device); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update device: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, device)
}

// deleteDevice deletes a device
func (h *APIHandler) deleteDevice(w http.ResponseWriter, r *http.Request) {
	deviceIDStr := chi.URLParam(r, "deviceID")
	deviceID, err := strconv.ParseUint(deviceIDStr, 10, 32)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid device ID")
		return
	}

	if err := h.models.Device.DeleteDevice(uint(deviceID)); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete device: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Device deleted successfully"})
}

// getDeviceLogs returns logs for a specific device
func (h *APIHandler) getDeviceLogs(w http.ResponseWriter, r *http.Request) {
	deviceIDStr := chi.URLParam(r, "deviceID")
	deviceID, err := strconv.ParseUint(deviceIDStr, 10, 32)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid device ID")
		return
	}

	logs, err := h.models.DeviceData.GetByDeviceID(uint(deviceID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get device logs: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"logs":  logs,
		"count": len(logs),
	})
}

// getLatestDeviceLog returns the latest log for a specific device
func (h *APIHandler) getLatestDeviceLog(w http.ResponseWriter, r *http.Request) {
	deviceIDStr := chi.URLParam(r, "deviceID")
	deviceID, err := strconv.ParseUint(deviceIDStr, 10, 32)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid device ID")
		return
	}

	log, err := h.models.DeviceData.GetLatestByDeviceID(uint(deviceID))
	if err != nil {
		writeError(w, http.StatusNotFound, "No logs found for device")
		return
	}

	writeJSON(w, http.StatusOK, log)
}

// getDeviceBySerialNumber returns a device by serial number
func (h *APIHandler) getDeviceBySerialNumber(w http.ResponseWriter, r *http.Request) {
	serialNumber := chi.URLParam(r, "serialNumber")
	if serialNumber == "" {
		writeError(w, http.StatusBadRequest, "Serial number is required")
		return
	}

	device, err := h.models.Device.GetBySerialNumber(serialNumber)
	if err != nil {
		writeError(w, http.StatusNotFound, "Device not found")
		return
	}

	writeJSON(w, http.StatusOK, device)
}

// getDeviceLogsBySerialNumber returns logs for a device by serial number
func (h *APIHandler) getDeviceLogsBySerialNumber(w http.ResponseWriter, r *http.Request) {
	serialNumber := chi.URLParam(r, "serialNumber")
	if serialNumber == "" {
		writeError(w, http.StatusBadRequest, "Serial number is required")
		return
	}

	logs, err := h.models.DeviceData.GetBySerialNumber(serialNumber)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get device logs: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"logs":  logs,
		"count": len(logs),
	})
}

// getAllLogs returns all device logs (with pagination)
func (h *APIHandler) getAllLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := h.models.DeviceData.GetAllLogs()
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get logs: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"logs":  logs,
		"count": len(logs),
	})
}

// getLogsByIMEI returns logs by IMEI
func (h *APIHandler) getLogsByIMEI(w http.ResponseWriter, r *http.Request) {
	imei := chi.URLParam(r, "imei")
	if imei == "" {
		writeError(w, http.StatusBadRequest, "IMEI is required")
		return
	}

	logs, err := h.models.DeviceData.GetByIMEI(imei)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get logs by IMEI: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"logs":  logs,
		"count": len(logs),
	})
}

// getLogsBySerialNumber returns logs by serial number
func (h *APIHandler) getLogsBySerialNumber(w http.ResponseWriter, r *http.Request) {
	serialNumber := chi.URLParam(r, "serialNumber")
	if serialNumber == "" {
		writeError(w, http.StatusBadRequest, "Serial number is required")
		return
	}

	logs, err := h.models.DeviceData.GetBySerialNumber(serialNumber)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get logs by serial number: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"logs":  logs,
		"count": len(logs),
	})
}

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]interface{}{
		"error":   message,
		"status":  status,
		"message": message,
	})
}
