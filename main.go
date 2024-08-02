package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"rindula/msteams-presence/homeassistant"
	"rindula/msteams-presence/token"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/joho/godotenv"
)

var version string

type Device struct {
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
	Name         string `json:"name"`
	SwVersion    string `json:"sw_version"`
	Identifiers  string `json:"identifiers"`
}

type HomeassistantDevice struct {
	Name                   string                    `json:"name,omitempty"`
	AvailabilityMode       string                    `json:"availability_mode,omitempty"`
	Device                 Device                    `json:"device,omitempty"`
	UniqueId               string                    `json:"unique_id,omitempty"`
	StateTopic             string                    `json:"state_topic,omitempty"`
	ValueTemplate          string                    `json:"value_template,omitempty"`
	ExpireAfter            int                       `json:"expire_after,omitempty"`
	Icon                   string                    `json:"icon,omitempty"`
	AvailabilityTemplate   string                    `json:"availability_template,omitempty"`
	JsonAttributesTopic    string                    `json:"json_attributes_topic,omitempty"`
	JsonAttributesTemplate string                    `json:"json_attributes_template,omitempty"`
	DeviceClass            homeassistant.DeviceClass `json:"device_class,omitempty"`
}

var expiration int64 = 120
var device = Device{
	Manufacturer: "Rindula",
	Model:        "Go",
	Name:         "Teams Status",
	SwVersion:    version,
	Identifiers:  "Teams Status",
}

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
		if os.Getenv("CLIENT_ID") == "" || os.Getenv("AUTH_TENANT") == "" || os.Getenv("GRAPH_USER_SCOPES") == "" || os.Getenv("MQTT_USER") == "" || os.Getenv("MQTT_PASSWORD") == "" || os.Getenv("MQTT_HOST") == "" {
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
	opts.SetAutoReconnect(false)
	opts.SetMaxReconnectInterval(15 * time.Second)
	opts.SetUsername(os.Getenv("MQTT_USER"))
	opts.SetPassword(os.Getenv("MQTT_PASSWORD"))
	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		panic("MQTT connection lost: " + err.Error())
	})
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		fmt.Println("Connected as", opts.ClientID)
		sensor_availability := HomeassistantDevice{
			Name:             "Teams Availability",
			AvailabilityMode: "all",
			Device:           device,
			UniqueId:         "teams_presence_availability",
			StateTopic:       "msteams/presence",
			ValueTemplate:    "{{ value_json.availability }}",
			ExpireAfter:      int(expiration),
			Icon:             "mdi:eye",
		}

		sensor_activity := HomeassistantDevice{
			Name:             "Teams Activity",
			AvailabilityMode: "all",
			Device:           device,
			UniqueId:         "teams_presence_activity",
			StateTopic:       "msteams/presence",
			ValueTemplate:    "{{ value_json.activity }}",
			ExpireAfter:      int(expiration),
			Icon:             "mdi:eye",
		}

		sensor_status := HomeassistantDevice{
			Name:                   "Teams Status Message",
			AvailabilityMode:       "all",
			Device:                 device,
			UniqueId:               "teams_presence_status",
			StateTopic:             "msteams/presence",
			ValueTemplate:          "{{ value_json.statusMessage.message.content |default('') }}",
			ExpireAfter:            int(expiration),
			Icon:                   "mdi:eye",
			DeviceClass:            homeassistant.DeviceClassNone,
		}
		sensorAvailabilityJSON, _ := json.Marshal(sensor_availability)
		sensorActivityJSON, _ := json.Marshal(sensor_activity)
		sensorStatusJSON, _ := json.Marshal(sensor_status)
		client.Publish("homeassistant/sensor/teams/availability/config", 1, false, string(sensorAvailabilityJSON))
		client.Publish("homeassistant/sensor/teams/activity/config", 1, false, string(sensorActivityJSON))
		client.Publish("homeassistant/sensor/teams/status/config", 1, false, string(sensorStatusJSON))
	})
	client := mqtt.NewClient(opts)
	if mqttToken := client.Connect(); mqttToken.Wait() && mqttToken.Error() != nil {
		panic(mqttToken.Error())
	}
	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		// check if client is still connected, else panic
		if !client.IsConnected() {
			panic("MQTT client is not connected")
		}
		presence := getPresence(token.GetToken())
		presenceJson, _ := json.Marshal(presence)
		fmt.Println(string(presenceJson))
		token := client.Publish("msteams/presence", 0, false, string(presenceJson))
		token.Wait()
	}
}

func getPresence(token token.Token) map[string]interface{} {
	defaultResponse := map[string]interface{}{"@odata.context": "", "availability": "unknown", "activity": "unknown", "statusMessage": nil, "id": ""}
	// get presence from microsoft graph api
	url := "https://graph.microsoft.com/v1.0/me/presence"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Error requesting presence", err)
		return defaultResponse
	}
	req.Header.Add("Authorization", "Bearer "+token.Token)
	data, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Error requesting presence", err)
		return defaultResponse
	}
	defer func() {
		err := data.Body.Close()
		if err != nil {
			fmt.Println("Error closing response body", err)
		}
	}()

	if data.StatusCode != 200 {
		fmt.Println("Error requesting presence", data.StatusCode)
		return defaultResponse
	}

	var presenceMap map[string]interface{}
	json.NewDecoder(data.Body).Decode(&presenceMap)

	return presenceMap
}
