package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/joho/godotenv"
)

func main() {
	// create .env file, if not exists
	if _, err := os.Stat(".env"); os.IsNotExist(err) {
		file, err := os.Create(".env")
		if err != nil {
			fmt.Println("Error creating .env file", err)
			os.Exit(1)
		}
		defer file.Close()
		// check if the environment variables are set and exit if not
		if os.Getenv("CLIENT_ID") == "" || os.Getenv("AUTH_TENANT") == "" || os.Getenv("GRAPH_USER_SCOPES") == "" || os.Getenv("MQTT_USER") == "" || os.Getenv("MQTT_PASSWORD") == "" {
			file.WriteString("CLIENT_ID=\n")
			file.WriteString("AUTH_TENANT=\n")
			file.WriteString("GRAPH_USER_SCOPES='user.read offline_access'\n")
			file.WriteString("MQTT_USER=\n")
			file.WriteString("MQTT_PASSWORD=\n")
			file.WriteString("MQTT_HOST=\n")
			file.WriteString("MQTT_PORT=1883\n")
			fmt.Println("Please fill in the .env file")
			os.Exit(1)
		} else {
			// fill in the .env file
			envs := os.Environ()
			envMap := make(map[string]string, len(envs))
			for _, s := range envs {
				pair := strings.SplitN(s, "=", 2)
				envMap[pair[0]] = pair[1]
			}
			envString, _ := godotenv.Marshal(envMap)
			file.WriteString(envString)
		}
	}

	// load .env file
	err := godotenv.Load(".env")

	if err != nil {
		fmt.Println("Error loading .env file")
	}

	// initialize mqtt client
	port, _ := strconv.Atoi(os.Getenv("MQTT_PORT"))
	opts := mqtt.NewClientOptions().AddBroker(fmt.Sprintf("tcp://%s:%d", os.Getenv("MQTT_HOST"), port))
	opts.SetClientID(fmt.Sprintf("go-presence-bot-%v", time.Now().UnixNano()))
	opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		fmt.Printf("TOPIC: %s\n", msg.Topic())
		fmt.Printf("MSG: %s\n", msg.Payload())
	})
	opts.SetPingTimeout(1 * time.Second)
	opts.SetKeepAlive(2 * time.Second)
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(15 * time.Second)
	opts.SetUsername(os.Getenv("MQTT_USER"))
	opts.SetPassword(os.Getenv("MQTT_PASSWORD"))
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		fmt.Println("Connected as", opts.ClientID)
		sensor_availability := "{\"name\": \"Teams Availability\",\"availability_mode\": \"all\",\"device\": {\"manufacturer\": \"DIY\",\"model\": \"Go\",\"name\": \"Teams Status\",\"sw_version\": \"1.2.0\",\"identifiers\": \"Teams Status\"},\"unique_id\": \"teams_presence_availablility\",\"state_topic\": \"msteams/presence\",\"value_template\": \"{{ value_json.availablility }}\",\"expire_after\": 120,\"icon\": \"mdi:eye\",\"platform\": \"mqtt\"}"
		sensor_activity := "{\"name\": \"Teams Activity\",\"availability_mode\": \"all\",\"device\": {\"manufacturer\": \"DIY\",\"model\": \"Go\",\"name\": \"Teams Status\",\"sw_version\": \"1.2.0\",\"identifiers\": \"Teams Status\"},\"unique_id\": \"teams_presence_activity\",\"state_topic\": \"msteams/presence\",\"value_template\": \"{{ value_json.activity }}\",\"expire_after\": 120,\"icon\": \"mdi:eye\",\"platform\": \"mqtt\"}"
		client.Publish("homeassistant/sensor/teams/availability/config", 0, false, sensor_availability)
		client.Publish("homeassistant/sensor/teams/activity/config", 0, false, sensor_activity)
	})
	client := mqtt.NewClient(opts)
	if mqttToken := client.Connect(); mqttToken.Wait() && mqttToken.Error() != nil {
		panic(mqttToken.Error())
	}
	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		presence := getPresence(getToken())
		presenceJson, _ := json.Marshal(presence)
		client.Publish("msteams/presence", 0, false, string(presenceJson))
	}
}

func getPresence(token Token) map[string]interface{} {
	// get presence from microsoft graph api
	url := "https://graph.microsoft.com/v1.0/me/presence"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Error requesting presence", err)
		os.Exit(1)
	}
	req.Header.Add("Authorization", "Bearer "+token.Token)
	data, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Error requesting presence", err)
		os.Exit(1)
	}
	defer data.Body.Close()

	if data.StatusCode != 200 {
		fmt.Println("Error requesting presence", data.StatusCode)
		os.Exit(1)
	}

	var presenceMap map[string]interface{}
	json.NewDecoder(data.Body).Decode(&presenceMap)

	return map[string]interface{}{"availablility": presenceMap["availability"].(string), "activity": presenceMap["activity"].(string)}
}
