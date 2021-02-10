package notifier

import (
	"bytes"
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
)

type Webhook struct {
	Url string
}

var _ Notifier = &Webhook{}

var client = http.Client{}

func (w *Webhook) Notify(name string) {
	payload := struct {
		Name string `json:"name"`
	}{
		name,
	}

	jsonPayload, err := json.Marshal(&payload)
	if err != nil {
		log.WithError(err).Warn("could not update via webhook")
		return
	}

	req, err := http.NewRequest(http.MethodPost, w.Url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.WithError(err).Warn("could not create webhook update request")
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.WithError(err).Warn("could not execute webhook request")
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		log.WithFields(log.Fields{
			"status_code": resp.StatusCode,
		}).Warn("non 2xx response code from webhook server")
		return
	}

	log.WithField("video", name).Info("webhook updated")
}