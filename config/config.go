package config

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	FFMPEGPath       = "ffmpeg.path"
	S3Bucket         = "s3.bucket"
	NotificationURLs = "notification_urls"
	StreamSite       = "stream.site"
	RTMPUrl          = "rtmp.url"

	TwitchEndpoint            = "twitch.endpoint"
	TwitchClientIdConfKey     = "twitch.client_id"
	TwitchAuthTokenConfKey    = "twitch.auth_token"
	TwitchRefreshTokenConfKey = "twitch.refresh_token"
	TwitchClientSecretConfKey = "twitch.client_secret"
)

func ReadConfig() {
	viper.AddConfigPath(".")
	viper.SetConfigName("bucket-stream")
	viper.SetConfigType("yaml")
	err := viper.ReadInConfig()
	if err != nil {
		log.WithError(err).Fatal("could not read config file")
	}
}
