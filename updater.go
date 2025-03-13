package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type Release struct {
	TagName string `json:"tag_name"`
	Url     string `json:"html_url"`
}

var apiBase string = "https://api.github.com/repos/Rindula/msteams-presence-bot-go"

func updateCheck() {
	lv, err := _updateCheck()
	if err != nil {
		log.Println("Error checking for updates:", err)
	} else {
		latestVersion = lv
	}
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		lv, err := _updateCheck()
		if err != nil {
			log.Println("Error checking for updates:", err)
		} else {
			latestVersion = lv
		}
		log.Println(version, latestVersion)
	}
}

func _updateCheck() (Release, error) {
	resp, err := http.Get(fmt.Sprintf("%s/releases/latest", apiBase))
	if err != nil {
		log.Println("Error checking for updates", err)
		return Release{}, fmt.Errorf("error checking for updates: %w", err)
	}
	if resp.StatusCode != 200 {
		log.Println("Error checking for updates", resp.StatusCode)
		return Release{}, fmt.Errorf("error checking for updates: %d", resp.StatusCode)
	}
	var release Release
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error reading response body", err)
		return Release{}, fmt.Errorf("error reading response body: %w", err)
	}
	resp.Body.Close()
	err = json.Unmarshal(data, &release)
	if err != nil {
		log.Println("Error parsing response body", err)
		return Release{}, fmt.Errorf("error parsing response body: %w", err)
	}

	if release.TagName != version {
		log.Printf("New version available: %s -> %s\n", version, release.TagName)
		log.Printf("Download: %s\n", release.Url)
	}

	return release, nil
}
