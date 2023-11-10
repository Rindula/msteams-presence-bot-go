package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

var tokenFile string = "token.data"
var refreshToken string

func saveToken(token string, validUntil int64, refreshToken string) bool {
	// Save token to file
	// try saving token to file
	fmt.Println("Saving token to file...")
	file, errCreateFile := os.Create(tokenFile)
	if errCreateFile != nil {
		fmt.Println("Error saving token")
		return false
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	jsonstring, _ := json.Marshal(map[string]string{"token": token, "validUntil": strconv.FormatInt(validUntil, 10), "refreshToken": refreshToken})
	encoded := base64.StdEncoding.EncodeToString(jsonstring)
	_, errWrite := writer.WriteString(encoded)
	if errWrite != nil {
		fmt.Println("Error saving token")
		return false
	}

	return true
}

func getToken() string {
	// Get token from file
	fileContent, err := os.ReadFile(tokenFile)
	if err != nil {
		fmt.Println("Error reading token file:", err)
		requestToken()
		return getToken()
	}
	var tokenMap map[string]string
	decoded, _ := base64.StdEncoding.DecodeString(string(fileContent))
	json.Unmarshal(decoded, &tokenMap)
	val, ok := tokenMap["token"]
	if ok {

		// Check if token is valid and not expired
		validUntil, errValidUntil := strconv.ParseInt(tokenMap["validUntil"], 10, 64)
		var okRefreshToken bool
		refreshToken, okRefreshToken = tokenMap["refreshToken"]

		if errValidUntil != nil {
			fmt.Println("Error reading token file")
			return ""
		}
		if validUntil < time.Now().Unix() {
			// token is expired
			if !okRefreshToken {
				fmt.Println("Error reading token file")
				return ""
			}
			// request new token
			return requestToken()
		}

		return val
	}
	return ""
}

func requestToken() string {
	if refreshToken == "" {
		requestRefreshToken()
	}
	// request microsoft graph token
	fmt.Println("Requesting usable token...")
	clientId := goDotEnvVariable("CLIENT_ID")
	scope := goDotEnvVariable("GRAPH_USER_SCOPES")
	tenantId := goDotEnvVariable("AUTH_TENANT")
	urlString := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantId)
	payloadData := url.Values{}
	payloadData.Set("grant_type", "refresh_token")
	payloadData.Set("client_id", clientId)
	payloadData.Set("scope", scope)
	payloadData.Set("refresh_token", refreshToken)
	payload := strings.NewReader(payloadData.Encode())
	req, err := http.NewRequest("POST", urlString, payload)
	if err != nil {
		fmt.Println("Error requesting token")
		os.Exit(1)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	tokenResponse, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Error requesting token:", err)
		os.Exit(1)
	}
	defer tokenResponse.Body.Close()

	if tokenResponse.StatusCode != 200 {
		var body map[string]interface{}
		json.NewDecoder(tokenResponse.Body).Decode(&body)
		fmt.Println("Error requesting token:", tokenResponse.Status, body)
		return ""
	}

	var tokenMap map[string]interface{}
	json.NewDecoder(tokenResponse.Body).Decode(&tokenMap)
	refreshToken = tokenMap["refresh_token"].(string)
	token := tokenMap["access_token"].(string)
	validUntil := time.Now().Unix() + int64(tokenMap["expires_in"].(float64))
	if !saveToken(token, validUntil, refreshToken) {
		fmt.Println("Error saving token")
		return ""
	}
	return token
}

func requestRefreshToken() string {
	// request microsoft graph token
	fmt.Println("Requesting refresh token...")
	clientId := goDotEnvVariable("CLIENT_ID")
	scope := goDotEnvVariable("GRAPH_USER_SCOPES")
	tenantId := goDotEnvVariable("AUTH_TENANT")
	url := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/devicecode", tenantId)
	payload := strings.NewReader(fmt.Sprintf("client_id=%s&scope=%s", clientId, scope))
	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		fmt.Println("Error requesting token")
		os.Exit(1)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	deviceCodeResponse, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Error requesting token")
		os.Exit(1)
	}
	defer deviceCodeResponse.Body.Close()

	var deviceCodeMap map[string]interface{}
	json.NewDecoder(deviceCodeResponse.Body).Decode(&deviceCodeMap)

	worksUntil := time.Now().Unix() + int64(deviceCodeMap["expires_in"].(float64))

	fmt.Println("Please go to " + deviceCodeMap["verification_uri"].(string) + " and enter the code " + deviceCodeMap["user_code"].(string))
	for {
		// check if token is valid
		token := checkToken(deviceCodeMap["device_code"].(string))
		fmt.Printf("Waiting for %d more seconds\r", worksUntil-time.Now().Unix())
		if token != "" {
			refreshToken = token
			fmt.Println("Token received                     ")
			return token
		}
		time.Sleep(time.Duration(deviceCodeMap["interval"].(float64)) * time.Second)

		if time.Now().Unix() > worksUntil {
			fmt.Println("Token expired")
			return ""
		}
	}
}

func checkToken(deviceCode string) string {
	// check if token is valid
	clientId := goDotEnvVariable("CLIENT_ID")
	scope := goDotEnvVariable("GRAPH_USER_SCOPES")
	tenantId := goDotEnvVariable("AUTH_TENANT")
	url := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantId)
	payload := strings.NewReader(fmt.Sprintf("grant_type=urn:ietf:params:oauth:grant-type:device_code&client_id=%s&scope=%s&device_code=%s", clientId, scope, deviceCode))
	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		fmt.Println("Error requesting token")
		os.Exit(1)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	tokenResponse, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Error requesting token")
		os.Exit(1)
	}
	defer tokenResponse.Body.Close()

	if tokenResponse.StatusCode != 200 {
		return ""
	}

	var tokenMap map[string]interface{}
	json.NewDecoder(tokenResponse.Body).Decode(&tokenMap)
	return tokenMap["refresh_token"].(string)
}
