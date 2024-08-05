package main

type Message struct {
	Content     string `json:"content"`
	ContentType string `json:"contentType,omitempty"`
}

type StatusMessage struct {
	Message Message `json:"message"`
}

type Presence struct {
	Availability  string         `json:"availability"`
	Activity      string         `json:"activity"`
	StatusMessage *StatusMessage `json:"statusMessage"`
}
