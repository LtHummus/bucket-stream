package twitch

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

type Api struct {
	ClientId      string
	AuthToken     string
	BroadcasterId int
}

var client = http.Client{}

// doTwitchRequest contains all the common logic for performing an authenticated request to the Twitch API and decoding
// the response in to the struct provided from the JSON response. If any errors happen, an error will be returned. If
// the HTTP response code is 204 NO CONTENT, then the function returns without attempting to decode the body and `nil` can
// be passed in as the second parameter. This function assumes that client id and auth token is set.
func (a *Api) doTwitchRequest(req *http.Request, res interface{}) error {
	req.Header.Set("Client-id", a.ClientId)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.AuthToken))

	resp, err := client.Do(req)
	if err != nil {
		log.WithError(err).Error("error making twitch request")
		return err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"statusCode": resp.StatusCode,
		}).Error("error extracting twitch response body")
		return err
	}

	if resp.StatusCode == http.StatusNoContent {
		// no content in the response, so just ignore it and return now
		return nil
	}

	if resp.StatusCode != http.StatusOK  {
		log.WithFields(log.Fields{
			"status_code": resp.StatusCode,
			"body": string(body),
		}).Error("error making http request")
		return errors.New("error making http request")
	}

	err = json.Unmarshal(body, res)
	if err != nil {
		log.WithError(err).Error("unable to retrieve user information")
		return err
	}

	return nil
}

// GetUserInfo updates the BroadcasterId for the Api struct for the user that owns the given AuthToken.
func (a *Api) GetUserInfo() {
	if a.ClientId == "" || a.AuthToken == "" {
		log.Warn("twitch api config not set...skipping getting user info")
		return
	}

	log.Info("getting user id")

	url := "https://api.twitch.tv/helix/users"

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.WithError(err).Fatal("error building request for user info")
	}

	var payload struct {
		Data []struct {
			Id          string `json:"id"`
			Login       string `json:"login"`
			DisplayName string `json:"display_name"`
		} `json:"data"`
	}

	err = a.doTwitchRequest(req, &payload)
	if err != nil {
		log.WithError(err).Fatal("could not request user info from twitch")
	}

	a.BroadcasterId, err = strconv.Atoi(payload.Data[0].Id)
	if err != nil {
		log.WithFields(log.Fields{
			"payload": payload,
		}).Fatal("could not assign broadcaster id")
	}

	log.WithFields(log.Fields{
		"broadcaster_id": a.BroadcasterId,
		"account_name": payload.Data[0].DisplayName,
	}).Info("set broadcaster id")
}

// GetTwitchEndpointUrl returns a complete endpoint URL with the stream key embedded.
func (a *Api) GetTwitchEndpointUrl() string {
	endpointUrl := a.GetClosestTwitchEndpoint()
	if endpointUrl == "" {
		return ""
	}

	streamKey := a.GetStreamKey()
	if streamKey == "" {
		return ""
	}

	log.Info("built full endpoint url")
	return strings.Replace(endpointUrl, "{stream_key}", streamKey, 1)
}

// GetClosestTwitchEndpoint gets the closest ingestion endpoint URL template from Twitch
func (a *Api) GetClosestTwitchEndpoint() string {
	if a.ClientId == "" || a.AuthToken == "" || a.BroadcasterId == 0 {
		log.Warn("twitch api config not set...returning empty string")
		return ""
	}

	log.Info("starting lookup of twitch ingestion endpoints")

	url := "https://ingest.twitch.tv/ingests"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.WithError(err).Fatal("error building request for user info")
	}

	var payload struct {
		Ingests []struct {
			Id          int    `json:"_id"`
			Name        string `json:"name"`
			UrlTemplate string `json:"url_template"`
		} `json:"ingests"`
	}

	err = a.doTwitchRequest(req, &payload)
	if err != nil {
		log.WithError(err).Fatal("could not get twitch endpoints")
	}

	log.WithField("endpoint_count", len(payload.Ingests)).Info("found endpoints")

	bestEndpointName := payload.Ingests[0].Name
	bestEndpointUrl := payload.Ingests[0].UrlTemplate

	log.WithField("endpoint_name", bestEndpointName).Info("picked endpoint")

	return bestEndpointUrl
}

// GetStreamKey fetches the user's stream key from the Twitch API
func (a *Api) GetStreamKey() string {
	if a.ClientId == "" || a.AuthToken == "" || a.BroadcasterId == 0 {
		log.Warn("twitch api config not set...returning empty string")
		return ""
	}

	url := fmt.Sprintf("https://api.twitch.tv/helix/streams/key?broadcaster_id=%d", a.BroadcasterId)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.WithError(err).Fatal("error building request for stream key")
	}

	var payload struct {
		Data []struct {
			StreamKey string `json:"stream_key"`
		} `json:"data"`
	}

	err = a.doTwitchRequest(req, &payload)
	if err != nil {
		log.WithError(err).Fatal("could not get stream key")
	}

	log.Info("retrieved stream key")

	return payload.Data[0].StreamKey
}

// UpdateStreamTitle sets the title of the user's stream to the given stream
func (a *Api) UpdateStreamTitle(title string) {
	if a.ClientId == "" || a.AuthToken == "" || a.BroadcasterId == 0 {
		log.WithField("video", title).Warn("twitch api config not set...skipping title update")
		return
	}

	payload, err := json.Marshal(map[string]interface{}{
		"title": title,
	})
	if err != nil {
		log.WithField("video", title).WithError(err).Fatal("unable to marshal JSON")
	}

	url := fmt.Sprintf("https://api.twitch.tv/helix/channels?broadcaster_id=%d", a.BroadcasterId)

	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(payload))
	if err != nil {
		log.WithField("video", title).WithError(err).Fatal("could not construct request")
	}
	req.Header.Set("Content-Type", "application/json")

	err = a.doTwitchRequest(req, nil)
	if err != nil {
		log.WithField("video", title).WithError(err).Warn("could not update twitch title")
	} else {
		log.WithField("video", title).Info("updated twitch stream title")
	}
}
