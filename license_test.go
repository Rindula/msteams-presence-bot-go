package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidateLicense(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/licenses/validate" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("content type = %q", got)
		}
		var request licenseValidationRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.LicenseKey != "TEST-KEY" || request.DeviceID != "device-1" {
			t.Fatalf("request body = %+v", request)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(licenseValidationResponse{Valid: true, Customer: "Test"})
	}))
	defer server.Close()

	result, err := validateLicense(context.Background(), server.Client(), server.URL, " TEST-KEY ", " device-1 ")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Valid || result.Customer != "Test" {
		t.Fatalf("result = %+v", result)
	}
}

func TestValidateLicenseRejectsInvalidLicense(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(licenseValidationResponse{Reason: "expired"})
	}))
	defer server.Close()

	result, err := validateLicense(context.Background(), server.Client(), server.URL, "KEY", "device")
	if err != nil {
		t.Fatal(err)
	}
	if result.Valid || result.Reason != "expired" {
		t.Fatalf("result = %+v", result)
	}
}

func TestValidateLicenseRequiresCredentials(t *testing.T) {
	client := &http.Client{}
	if _, err := validateLicense(context.Background(), client, "http://localhost", "", "device"); err == nil {
		t.Fatal("empty license key was accepted")
	}
	if _, err := validateLicense(context.Background(), client, "http://localhost", "key", ""); err == nil {
		t.Fatal("empty device ID was accepted")
	}
}
