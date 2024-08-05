package token

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

var tokenFile string = "token.data"

type Token struct {
	Token        string
	ValidUntil   int64
	RefreshToken string
}

func saveToken(token Token) bool {
	// Save token to file
	// try saving token to file
	fmt.Println("Saving token to file...")
	file, errCreateFile := os.Create(tokenFile)
	if errCreateFile != nil {
		fmt.Println("[pasgka] Error saving token", errCreateFile)
		return false
	}
	defer file.Close()

	b := bytes.Buffer{}
	e := gob.NewEncoder(&b)
	errEncode := e.Encode(token)
	if errEncode != nil {
		fmt.Println("[snogim] Error saving token", errEncode)
		return false
	}

	writer := bufio.NewWriter(file)
	_, errWrite := writer.WriteString(base64.StdEncoding.EncodeToString(b.Bytes()))
	if errWrite != nil {
		fmt.Println("[oseigj] Error saving token", errWrite)
		return false
	}

	return true
}

func GetToken() Token {
	var token Token

	// Get token from file
	fileContent, err := os.ReadFile(tokenFile)
	if err != nil {
		log.Println("[siogre] Error reading token file:", err)
		requestToken(&Token{})
		return GetToken()
	}

	decoded, errDecode := base64.StdEncoding.DecodeString(string(fileContent))
	if errDecode != nil {
		log.Println("[usieog] Error reading token file:", errDecode)
		requestToken(&Token{})
		return GetToken()
	}
	b := bytes.Buffer{}
	b.Write(decoded)
	d := gob.NewDecoder(&b)
	errDecode = d.Decode(&token)
	if errDecode != nil {
		log.Println("[srhigf] Error reading token file:", errDecode)
		requestToken(&Token{})
		return GetToken()
	}

	// Check if token is valid and not expired
	if token.ValidUntil < time.Now().Unix() {
		// request new token
		return requestToken(&token)
	}

	return token

}

func requestToken(oldToken *Token) Token {
	if oldToken.RefreshToken == "" {
		requestRefreshToken(oldToken)
	}
	// request microsoft graph token
	log.Println("Requesting usable token...")
	clientId := os.Getenv("CLIENT_ID")
	scope := os.Getenv("GRAPH_USER_SCOPES")
	tenantId := os.Getenv("AUTH_TENANT")
	urlString := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantId)
	payloadData := url.Values{}
	payloadData.Set("grant_type", "refresh_token")
	payloadData.Set("client_id", clientId)
	payloadData.Set("scope", scope)
	payloadData.Set("refresh_token", oldToken.RefreshToken)
	resp, err := http.PostForm(urlString, payloadData)
	if err != nil {
		log.Println("[xgjipt] Error requesting token", err)
		return Token{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		var body map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&body)
		log.Println("[akfswg] Error requesting token:", resp.Status, body)
		if oldToken.RefreshToken != "" {
			t := Token{}
			return requestToken(&t)
		} else {
			return Token{}
		}
	}

	var tokenMap map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&tokenMap)
	var token Token
	token.RefreshToken = tokenMap["refresh_token"].(string)
	token.Token = tokenMap["access_token"].(string)
	token.ValidUntil = time.Now().Unix() + int64(tokenMap["expires_in"].(float64))
	if !saveToken(token) {
		log.Println("[rhejil] Error saving token")
		return Token{}
	}
	return token
}

func requestRefreshToken(oldToken *Token) string {
	// request microsoft graph token
	fmt.Println("Requesting refresh token...")
	clientId := os.Getenv("CLIENT_ID")
	scope := os.Getenv("GRAPH_USER_SCOPES")
	tenantId := os.Getenv("AUTH_TENANT")
	deviceCodeUrl := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/devicecode", tenantId)
	payloadData := url.Values{}
	payloadData.Add("client_id", clientId)
	payloadData.Add("scope", scope)
	resp, err := http.PostForm(deviceCodeUrl, payloadData)
	if err != nil {
		log.Fatalln("[dfiqwc] Error requesting token on device code flow with url", deviceCodeUrl, err)
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Fatalln("[dfibet] Error closing response body (refreshtoken)", err)
		}
	}()

	var deviceCodeMap map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&deviceCodeMap)

	worksUntil := time.Now().Unix() + int64(deviceCodeMap["expires_in"].(float64))

	log.Println("Please go to " + deviceCodeMap["verification_uri"].(string) + " and enter the code " + deviceCodeMap["user_code"].(string))
	for {
		// check if token is valid
		token := checkToken(deviceCodeMap["device_code"].(string))
		log.Printf("Waiting for %d more seconds", worksUntil-time.Now().Unix())
		if token != "" {
			oldToken.RefreshToken = token
			log.Println("Token received                     ")
			return token
		}
		time.Sleep(time.Duration(deviceCodeMap["interval"].(float64)) * time.Second)

		if time.Now().Unix() > worksUntil {
			log.Println("Token expired")
			return ""
		}
	}
}

func checkToken(deviceCode string) string {
	// check if token is valid
	clientId := os.Getenv("CLIENT_ID")
	scope := os.Getenv("GRAPH_USER_SCOPES")
	tenantId := os.Getenv("AUTH_TENANT")
	tokenUrl := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantId)
	payloadData := url.Values{}
	payloadData.Add("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
	payloadData.Add("client_id", clientId)
	payloadData.Add("scope", scope)
	payloadData.Add("device_code", deviceCode)
	resp, err := http.PostForm(tokenUrl, payloadData)
	if err != nil {
		log.Fatalln("[ruiurb] Error requesting token", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "" // no token yet, try again in the next run
	}

	var tokenMap map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&tokenMap)
	return tokenMap["refresh_token"].(string)
}
