package config

type TwitchCredentials struct {
	AuthToken    string `mapstructure:"auth_token,omitempty"`
	ClientID     string `mapstructure:"client_id,omitempty"`
	ClientSecret string `mapstructure:"client_secret,omitempty"`
	RefreshToken string `mapstructure:"refresh_token,omitempty"`
}

type StreamConfiguration struct {
	Name              string             `mapstructure:"name"`
	NotificationURLs  []string           `mapstructure:"notification_urls,omitempty"`
	Endpoint          string             `mapstrucutre:"endpoint,omitempty"`
	TwitchCredentials *TwitchCredentials `mapstructure:"twitch,omitempty"`
}

type S3Configuration struct {
	Bucket string `mapstructure:"bucket"`
	Region string `mapstructure:"region"`
}

type Configuration struct {
	S3      S3Configuration       `mapstructure:"s3"`
	FFmpeg  *FfmpegConfiguration  `mapstructure:"ffmpeg"`
	Streams []StreamConfiguration `mapstructure:"streams"`
}

type FfmpegConfiguration struct {
	Path string `mapstructure:"path"`
}

func (c *Configuration) AsMap() map[string]any {
	var streams []map[string]any

	for _, curr := range c.Streams {
		streamMap := map[string]any{
			"name": curr.Name,
		}

		if len(curr.NotificationURLs) > 0 {
			streamMap["notification_urls"] = curr.NotificationURLs
		}

		if curr.Endpoint != "" {
			streamMap["endpoint"] = curr.Endpoint
		}

		if curr.TwitchCredentials != nil {
			twitchCredsMap := map[string]any{
				"client_id": curr.TwitchCredentials.ClientID,
			}

			if curr.TwitchCredentials.ClientSecret != "" {
				twitchCredsMap["client_secret"] = curr.TwitchCredentials.ClientSecret
			}

			if curr.TwitchCredentials.AuthToken != "" {
				twitchCredsMap["auth_token"] = curr.TwitchCredentials.AuthToken
			}

			if curr.TwitchCredentials.RefreshToken != "" {
				twitchCredsMap["refresh_token"] = curr.TwitchCredentials.RefreshToken
			}

			streamMap["twitch"] = twitchCredsMap
		}

		streams = append(streams, streamMap)
	}

	m := map[string]any{
		"s3": map[string]any{
			"bucket": c.S3.Bucket,
			"region": c.S3.Region,
		},
		"streams": streams,
	}

	if c.FFmpeg != nil && c.FFmpeg.Path != "" {
		m["ffmpeg"] = map[string]any{
			"path": c.FFmpeg.Path,
		}
	}

	return m
}
