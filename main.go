package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rindula/msteams-presence-bot-go/homeassistant"
	"github.com/rindula/msteams-presence-bot-go/token"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/joho/godotenv"
)

var version string = "development"
var latestVersion Release

type Device struct {
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
	Name         string `json:"name"`
	SwVersion    string `json:"sw_version"`
	Identifiers  string `json:"identifiers"`
}

type HomeassistantDevice struct {
	Name                   string                       `json:"name,omitempty"`
	AvailabilityMode       string                       `json:"availability_mode,omitempty"`
	Device                 Device                       `json:"device,omitempty"`
	UniqueId               string                       `json:"unique_id,omitempty"`
	StateTopic             string                       `json:"state_topic"`
	ValueTemplate          string                       `json:"value_template,omitempty"`
	ExpireAfter            int                          `json:"expire_after,omitempty"`
	Icon                   string                       `json:"icon,omitempty"`
	AvailabilityTemplate   string                       `json:"availability_template,omitempty"`
	JsonAttributesTopic    string                       `json:"json_attributes_topic,omitempty"`
	JsonAttributesTemplate string                       `json:"json_attributes_template,omitempty"`
	DeviceClass            homeassistant.DeviceClass    `json:"device_class,omitempty"`
	PayloadAvailable       string                       `json:"payload_available,omitempty"`
	PayloadNotAvailable    string                       `json:"payload_not_available,omitempty"`
	EntityCategory         homeassistant.EntityCategory `json:"entity_category,omitempty"`
	LatestVersionTopic     string                       `json:"latest_version_topic,omitempty"`
	LatestVersionTemplate  string                       `json:"latest_version_template,omitempty"`
	ReleaseUrl             string                       `json:"release_url,omitempty"`
}

type Version struct {
	Version string  `json:"version"`
	Latest  Release `json:"latest"`
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
			log.Fatalln("Error creating .env file", err)
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
			log.Fatalln("Please fill in the .env file")
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
		log.Println("Error loading .env file")
	}
	latestVersion = Release{TagName: version, Url: ""}

	// initialize mqtt client
	port, _ := strconv.Atoi(os.Getenv("MQTT_PORT"))
	opts := mqtt.NewClientOptions().AddBroker(fmt.Sprintf("tcp://%s:%d", os.Getenv("MQTT_HOST"), port))
	opts.SetClientID(fmt.Sprintf("go-presence-bot-%v", time.Now().UnixNano()))
	opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		log.Printf("TOPIC: %s\n", msg.Topic())
		log.Printf("MSG: %s\n", msg.Payload())
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
		sendDeviceDescriptionMqtt(client)
	})
	client := mqtt.NewClient(opts)
	if mqttToken := client.Connect(); mqttToken.Wait() && mqttToken.Error() != nil {
		panic(mqttToken.Error())
	}
	go updateCheck()
	go sendDeviceDescription(client)
	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		// check if client is still connected, else panic
		if !client.IsConnected() {
			panic("MQTT client is not connected")
		}
		presence := getPresence(token.GetToken())
		presenceJson, _ := json.Marshal(presence)
		log.Println(string(presenceJson))

		token := client.Publish("msteams/presence", 0, false, string(presenceJson))
		go func() {
			token.Wait()
			if token.Error() != nil {
				log.Panicln("Error publishing presence:", token.Error())
			}
		}()
		v := Version{Version: version, Latest: latestVersion}
		versionJson, _ := json.Marshal(v)
		versionToken := client.Publish("msteams/version", 0, false, versionJson)
		go func() {
			versionToken.Wait()
			if versionToken.Error() != nil {
				log.Panicln("Error publishing version:", versionToken.Error())
			}
		}()
	}
}

func getPresence(token token.Token) Presence {
	presence := Presence{
		Availability:  "unknown",
		Activity:      "unknown",
		StatusMessage: nil,
	}
	// get presence from microsoft graph api
	url := "https://graph.microsoft.com/v1.0/me/presence"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println("Error requesting presence", err)
		return presence
	}
	req.Header.Add("Authorization", "Bearer "+token.Token)
	data, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("Error requesting presence", err)
		return presence
	}
	defer func() {
		err := data.Body.Close()
		if err != nil {
			log.Fatalln("Error closing response body", err)
		}
	}()

	if data.StatusCode != 200 {
		log.Println("Error requesting presence", data.StatusCode)
		return presence
	}

	body, err := io.ReadAll(data.Body)
	if err != nil {
		log.Println("Error reading response body", err)
		return presence
	}
	json.Unmarshal(body, &presence)

	return presence
}

func sendDeviceDescription(client mqtt.Client) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		sendDeviceDescriptionMqtt(client)
	}
}

func sendDeviceDescriptionMqtt(client mqtt.Client) {
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
		Name:                "Teams Status Message",
		AvailabilityMode:    "all",
		Device:              device,
		UniqueId:            "teams_presence_status",
		StateTopic:          "msteams/presence",
		ValueTemplate:       "{{ value_json.statusMessage.message.content }}",
		ExpireAfter:         int(expiration),
		Icon:                "mdi:eye",
		DeviceClass:         homeassistant.DeviceClassNone,
		PayloadNotAvailable: "",
	}

	sensor_update := HomeassistantDevice{
		Name:                  "Teams Status Update",
		AvailabilityMode:      "all",
		Device:                device,
		UniqueId:              "teams_presence_update",
		StateTopic:            "msteams/version",
		ValueTemplate:         "{{ value_json.version }}",
		ExpireAfter:           int(expiration),
		Icon:                  "mdi:update",
		DeviceClass:           homeassistant.DeviceClassFirmware,
		EntityCategory:        homeassistant.EntityCategoryDiagnostic,
		LatestVersionTopic:    "msteams/version",
		LatestVersionTemplate: "{{ value_json.latest.tag_name }}",
		ReleaseUrl:            "{{ value_json.latest.url }}",
	}
	sensorAvailabilityJSON, _ := json.Marshal(sensor_availability)
	sensorActivityJSON, _ := json.Marshal(sensor_activity)
	sensorStatusJSON, _ := json.Marshal(sensor_status)
	sensorUpdateJSON, _ := json.Marshal(sensor_update)
	client.Publish("homeassistant/sensor/teams/availability/config", 1, false, string(sensorAvailabilityJSON))
	client.Publish("homeassistant/sensor/teams/activity/config", 1, false, string(sensorActivityJSON))
	client.Publish("homeassistant/sensor/teams/status/config", 1, false, string(sensorStatusJSON))
	client.Publish("homeassistant/sensor/teams/update/config", 1, false, string(sensorUpdateJSON))
}
