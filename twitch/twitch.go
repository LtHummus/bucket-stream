package twitch

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	twitchClientIdConfKey     = "twitch.client_id"
	twitchAuthTokenConfKey    = "twitch.auth_token"
	twitchRefreshTokenConfKey = "twitch.refresh_token"
	twitchClientSecretConfKey = "twitch.client_secret"
)

type Api struct {
	BroadcasterId int
}

func getTwitchClientId() string {
	return viper.GetString(twitchClientIdConfKey)
}

func getTwitchAuthToken() string {
	return viper.GetString(twitchAuthTokenConfKey)
}

func getTwitchRefreshToken() string {
	return viper.GetString(twitchRefreshTokenConfKey)
}

func getTwitchClientSecret() string {
	return viper.GetString(twitchClientSecretConfKey)
}

func updateTwitchCredentials(accessToken string, refreshToken string) {
	viper.Set(twitchAuthTokenConfKey, accessToken)
	viper.Set(twitchRefreshTokenConfKey, refreshToken)
	err := viper.WriteConfig()
	if err != nil {
		log.WithError(err).Warn("could not write config")
	} else {
		log.Info("updated twitch credentials written")
	}
}

var client = http.Client{}

// refreshTwitchToken uses our oauth2 client credentials to refresh the access token for our user. This would probably
// be better served with a real oauth2 client, but whatever...
func (a *Api) refreshTwitchToken() error {
	log.Info("refreshing twitch tokens")

	payload := url.Values{}
	payload.Set("grant_type", "refresh_token")
	payload.Set("refresh_token", getTwitchRefreshToken())
	payload.Set("client_id", getTwitchClientId())
	payload.Set("client_secret", getTwitchClientSecret())

	req, err := http.NewRequest(http.MethodPost, "https://id.twitch.tv/oauth2/token", strings.NewReader(payload.Encode()))
	if err != nil {
		log.WithError(err).Warn("could not refresh token")
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		log.WithError(err).Warn("could not do http request")
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.WithField("status_code", resp.Status).Warn("error from twitch server")
		return errors.New("twitch auth failure")
	}

	var refreshResult struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	err = json.NewDecoder(resp.Body).Decode(&refreshResult)
	if err != nil {
		log.WithError(err).Warn("could not decode twitch token response")
		return err
	}

	updateTwitchCredentials(refreshResult.AccessToken, refreshResult.RefreshToken)
	log.Info("twitch tokens updated")

	return nil
}

func validateTwitchToken() (bool, error) {
	req, err := http.NewRequest(http.MethodGet, "https://id.twitch.tv/oauth2/validate", nil)
	if err != nil {
		log.WithError(err).Warn("could not create validation payload")
		return false, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", getTwitchAuthToken()))

	resp, err := client.Do(req)
	if err != nil {
		log.WithError(err).Warn("error doing validation request")
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	return true, nil

}

// doTwitchRequest contains all the common logic for performing an authenticated request to the Twitch API and decoding
// the response in to the struct provided from the JSON response. If any errors happen, an error will be returned. If
// the HTTP response code is 204 NO CONTENT, then the function returns without attempting to decode the body and `nil` can
// be passed in as the second parameter. This function assumes that client id and auth token is set.
func (a *Api) doTwitchRequest(req *http.Request, res interface{}) error {
	valid, err := validateTwitchToken()
	if err != nil {
		log.WithError(err).Warn("could not validate twitch token")
		return err
	}

	if !valid {
		err := a.refreshTwitchToken()
		if err != nil {
			log.WithError(err).Warn("could not refresh token")
			return err
		}
	}

	req.Header.Set("Client-id", getTwitchClientId())
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", getTwitchAuthToken()))

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

	if resp.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{
			"status_code": resp.StatusCode,
			"body":        string(body),
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
	if getTwitchClientId() == "" || getTwitchAuthToken() == "" {
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
		"account_name":   payload.Data[0].DisplayName,
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
	if getTwitchClientId() == "" || getTwitchAuthToken() == "" || a.BroadcasterId == 0 {
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
	if getTwitchClientId() == "" || getTwitchAuthToken() == "" || a.BroadcasterId == 0 {
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
	if getTwitchClientId() == "" || getTwitchAuthToken() == "" || a.BroadcasterId == 0 {
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
