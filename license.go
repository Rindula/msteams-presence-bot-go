package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const defaultLicenseServerURL = "https://license.rindula.de"
const licenseCheckInterval = 15 * time.Minute

type licenseValidationRequest struct {
	LicenseKey string `json:"license_key"`
	DeviceID   string `json:"device_id"`
}

type licenseValidationResponse struct {
	Valid       bool       `json:"valid"`
	Reason      string     `json:"reason,omitempty"`
	Customer    string     `json:"customer,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Activations int        `json:"activations,omitempty"`
}

func validateLicense(ctx context.Context, client *http.Client, serverURL, licenseKey, deviceID string) (licenseValidationResponse, error) {
	licenseKey = strings.TrimSpace(licenseKey)
	deviceID = strings.TrimSpace(deviceID)
	if licenseKey == "" {
		return licenseValidationResponse{}, fmt.Errorf("LICENSE_KEY is not set")
	}
	if deviceID == "" {
		return licenseValidationResponse{}, fmt.Errorf("LICENSE_DEVICE_ID is not set and the hostname is unavailable")
	}

	body, err := json.Marshal(licenseValidationRequest{LicenseKey: licenseKey, DeviceID: deviceID})
	if err != nil {
		return licenseValidationResponse{}, fmt.Errorf("encode license request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(serverURL, "/")+"/api/v1/licenses/validate", bytes.NewReader(body))
	if err != nil {
		return licenseValidationResponse{}, fmt.Errorf("create license request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return licenseValidationResponse{}, fmt.Errorf("license server request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return licenseValidationResponse{}, fmt.Errorf("license server returned HTTP %d", resp.StatusCode)
	}

	var result licenseValidationResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 32<<10)).Decode(&result); err != nil {
		return licenseValidationResponse{}, fmt.Errorf("decode license server response: %w", err)
	}
	return result, nil
}

func currentLicenseDeviceID() (string, error) {
	if deviceID := strings.TrimSpace(os.Getenv("LICENSE_DEVICE_ID")); deviceID != "" {
		return deviceID, nil
	}
	if hostname, err := os.Hostname(); err == nil && strings.TrimSpace(hostname) != "" {
		return strings.TrimSpace(hostname), nil
	}
	return "", fmt.Errorf("cannot determine a device ID; set LICENSE_DEVICE_ID")
}

func authenticateLicense() error {
	deviceID, err := currentLicenseDeviceID()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	result, err := validateLicense(ctx, &http.Client{Timeout: 15 * time.Second}, defaultLicenseServerURL, os.Getenv("LICENSE_KEY"), deviceID)
	if err != nil {
		return err
	}
	if !result.Valid {
		if result.Reason == "" {
			result.Reason = "rejected"
		}
		return fmt.Errorf("license rejected: %s", result.Reason)
	}
	return nil
}

func periodicLicenseCheck() {
	ticker := time.NewTicker(licenseCheckInterval)
	defer ticker.Stop()
	for range ticker.C {
		if err := authenticateLicense(); err != nil {
			log.Fatalln("Periodic license validation failed:", err)
		}
	}
}
