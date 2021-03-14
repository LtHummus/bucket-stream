package main

import (
	"fmt"
	"github.com/spf13/viper"
	"math/rand"
	"path"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/lthummus/bucket-stream/notifier"
	"github.com/lthummus/bucket-stream/server"
	"github.com/lthummus/bucket-stream/streamer"
	"github.com/lthummus/bucket-stream/twitch"
	"github.com/lthummus/bucket-stream/videostorage"
)

type config struct {
	ffmpegPath               string
	twitchClientId           string
	twitchAuthToken          string
	bucketName               string
	s3EndpointOverride       string
	twitchEndpointOverride   string
	notificationWebhookUrl   string
	enumerationPeriodMinutes int
	serverPort               int
}

func readConfig() (config, error) {
	viper.SetConfigName("bucket-stream")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/config")
	viper.AddConfigPath(".")

	viper.SetDefault("ffmpegPath", "ffmpeg")
	viper.SetDefault("serverPort", 8080)
	viper.SetDefault("storage.enumerationPeriodMinutes", 1440)

	viper.SetEnvPrefix("BUCKET_STREAM")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// nop
		} else {
			return config{}, err
		}
	}

	return config{
		ffmpegPath:               viper.GetString("ffmpeg_path"),
		twitchClientId:           viper.GetString("twitch.client_id"),
		twitchAuthToken:          viper.GetString("twitch.auth_token"),
		bucketName:               viper.GetString("storage.bucket_name"),
		s3EndpointOverride:       viper.GetString("storage.endpoint"),
		twitchEndpointOverride:   viper.GetString("twitch.endpoint"),
		notificationWebhookUrl:   viper.GetString("notification_webhook_url"),
		enumerationPeriodMinutes: viper.GetInt("storage.enumeration_period_minutes"),
		serverPort:               viper.GetInt("server_port"),
	}, nil
}

func main() {
	config, err := readConfig()
	if err != nil {
		panic(fmt.Sprintf("could not read config: %s", err))
	}

	// set up logging and initialize the RNG
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	log.Info("hello world!")
	rand.Seed(time.Now().Unix())

	log.WithField("ffmpegPath", config.ffmpegPath).Info("ffmpeg path")

	twitchApi := &twitch.Api{
		ClientId:  config.twitchClientId,
		AuthToken: config.twitchAuthToken,
	}

	// initialize the twitch API
	twitchApi.GetUserInfo()

	// read the source bucket
	bucketName := config.bucketName
	if bucketName == "" {
		log.Fatal("video bucket name not set")
	}

	// read the twitch endpoint URL (which includes the stream key -- see README for more details)
	twitchEndpoint := config.twitchEndpointOverride
	if twitchEndpoint == "" {
		potentialEndpoint := twitchApi.GetTwitchEndpointUrl()
		if potentialEndpoint == "" {
			log.Fatal("could not get stream key from environment variable or twitch api")
		}
		twitchEndpoint = potentialEndpoint
	} else {
		log.Info("using explicit twitch endpoint")
	}

	// initialize video storage
	storage := videostorage.New(bucketName, config.s3EndpointOverride, config.enumerationPeriodMinutes)
	log.WithField("bucket", bucketName).Info("video storage initialized")

	var notifiers []notifier.Notifier
	if webhookUrl := config.notificationWebhookUrl; webhookUrl != "" {
		log.Info("initializing webhook notifier")
		notifiers = append(notifiers, &notifier.Webhook{
			Url: webhookUrl,
		})
	}

	// start streamer
	strm := streamer.Streamer{
		FfmpegPath:     config.ffmpegPath,
		TwitchEndpoint: twitchEndpoint,
	}

	// start server
	srv := server.Server{
		Storage:  storage,
		Streamer: &strm,
		Port:     config.serverPort,
	}
	go srv.StartServer()

	// main loop of the app
	for {
		// pick a video
		log.Info("starting cycle")
		pickedVideo, buf := storage.PickVideo()
		log.WithFields(log.Fields{
			"video": pickedVideo,
		}).Info("winner picked")

		// update the stream title
		streamTitle := strings.TrimPrefix(strings.TrimSuffix(path.Base(pickedVideo), path.Ext(pickedVideo)), "/")
		go twitchApi.UpdateStreamTitle(streamTitle)

		for _, curr := range notifiers {
			go curr.Notify(streamTitle)
		}

		// start streaming
		log.WithFields(log.Fields{
			"video": pickedVideo,
		}).Info("opened stream")
		strm.StartFfmpegStream(pickedVideo, buf)
		log.WithFields(log.Fields{
			"video": pickedVideo,
		}).Info("cycle complete")

		if !srv.ShouldContinue() {
			log.Info("server says we should stop. so stopping")
			break
		}
	}

}
