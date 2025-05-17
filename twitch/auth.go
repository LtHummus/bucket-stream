package twitch

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/lthummus/bucket-stream/config"
	log "github.com/sirupsen/logrus"
)

const redirectUrl = "http://localhost"

func GenerateAuthUrl(twitchCreds *config.TwitchCredentials) string {
	u := url.Values{}
	u.Set("client_id", twitchCreds.ClientID)
	u.Set("redirect_uri", redirectUrl)
	u.Set("response_type", "code")
	u.Set("scope", "channel:read:stream_key user:edit:broadcast")

	return fmt.Sprintf("https://id.twitch.tv/oauth2/authorize?%s", u.Encode())
}

func Handshake(code string, twitchCreds *config.TwitchCredentials) error {
	req, err := http.NewRequest(http.MethodPost, "https://id.twitch.tv/oauth2/token", nil)
	if err != nil {
		log.WithError(err).Warn("could not build request")
		return err
	}

	q := url.Values{}
	q.Set("client_id", twitchCreds.ClientID)
	q.Set("client_secret", twitchCreds.ClientSecret)
	q.Set("code", code)
	q.Set("grant_type", "authorization_code")
	q.Set("redirect_uri", redirectUrl)

	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		log.WithError(err).Warn("could not do token exchange")
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.WithError(err).Fatal("could not read error response body")
		}
		fmt.Printf("%s\n", string(body))
		log.Fatal("error on exchange")
	}

	var tokenPayload struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	err = json.NewDecoder(resp.Body).Decode(&tokenPayload)
	if err != nil {
		log.WithError(err).Warn("could not decode JSON token payload")
		return err
	}

	twitchCreds.AuthToken = tokenPayload.AccessToken
	twitchCreds.RefreshToken = tokenPayload.RefreshToken

	return config.SaveConfig()
}
