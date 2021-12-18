package main

import (
	"fmt"
	"math/rand"
	"os"
	"path"
	"strings"
	"time"

	"github.com/lthummus/bucket-stream/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/lthummus/bucket-stream/notifier"
	"github.com/lthummus/bucket-stream/server"
	"github.com/lthummus/bucket-stream/streamer"
	"github.com/lthummus/bucket-stream/twitch"
	"github.com/lthummus/bucket-stream/videostorage"
)

func handleAuth() {
	fmt.Printf("handling auth...\n")
	fmt.Printf("go to\n%s\n\n", twitch.GenerateAuthUrl())
	fmt.Printf("Authorization code: ")

	var code string
	_, err := fmt.Scanf("%s", &code)
	if err != nil {
		log.WithError(err).Warn("could not read input")
	}

	err = twitch.Handshake(code)
	if err != nil {
		log.WithError(err).Warn("could not update tokens")
	}

}

func main() {
	// set up logging and initialize the RNG
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	log.Info("hello world!")
	rand.Seed(time.Now().Unix())

	config.ReadConfig()

	if len(os.Args) > 1 && os.Args[1] == "auth" {
		handleAuth()
		twitchApi := &twitch.Api{}
		twitchApi.GetUserInfo()
		os.Exit(0)
	}

	// get the ffmpeg path
	ffmpegPath := viper.GetString("ffmpeg.path")
	if ffmpegPath == "" {
		log.Warn("FFMPEG_PATH not set. I hope `ffmpeg` is in your $PATH!")
		ffmpegPath = "ffmpeg"
	}

	twitchApi := &twitch.Api{}

	// initialize the twitch API
	twitchApi.GetUserInfo()

	// read the source bucket
	bucketName := viper.GetString("s3.bucket")
	if bucketName == "" {
		log.Fatal("environment variable VIDEO_BUCKET_NAME is empty")
	}

	// read the twitch endpoint URL (which includes the stream key -- see README for more details)
	twitchEndpoint := viper.GetString("twitch.endpoint")
	if twitchEndpoint == "" {
		potentialEndpoint := twitchApi.GetTwitchEndpointUrl()
		if potentialEndpoint == "" {
			log.Fatal("could not get stream key from environment variable or twitch api")
		}
		twitchEndpoint = potentialEndpoint
	} else {
		log.Info("using twitch.endpoint from config")
	}

	// initialize video storage
	storage := videostorage.New(bucketName)
	log.WithField("bucket", bucketName).Info("video storage initialized")

	notifierURLs := viper.GetStringSlice("notification_urls")

	var notifiers []notifier.Notifier
	for _, curr := range notifierURLs {
		notifiers = append(notifiers, &notifier.Webhook{
			Url: curr,
		})
	}

	// start streamer
	strm := streamer.Streamer{
		FfmpegPath:     ffmpegPath,
		TwitchEndpoint: twitchEndpoint,
	}

	// start server
	srv := server.Server{
		Storage:  storage,
		Streamer: &strm,
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
