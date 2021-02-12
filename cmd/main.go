package main

import (
	"math/rand"
	"os"
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

func main() {
	// set up logging and initialize the RNG
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	log.Info("hello world!")
	rand.Seed(time.Now().Unix())

	// get the ffmpeg path
	ffmpegPath := os.Getenv("FFMPEG_PATH")
	if ffmpegPath == "" {
		log.Warn("FFMPEG_PATH not set. I hope `ffmpeg` is in your $PATH!")
		ffmpegPath = "ffmpeg"
	}

	// read twitch configuration data
	clientId := os.Getenv("TWITCH_CLIENT_ID")
	authToken := os.Getenv("TWITCH_AUTH_TOKEN")

	twitchApi := &twitch.Api{
		ClientId:  clientId,
		AuthToken: authToken,
	}

	// initialize the twitch API
	twitchApi.GetUserInfo()

	// read the source bucket
	bucketName := os.Getenv("VIDEO_BUCKET_NAME")
	if bucketName == "" {
		log.Fatal("environment variable VIDEO_BUCKET_NAME is empty")
	}

	// read the twitch endpoint URL (which includes the stream key -- see README for more details)
	twitchEndpoint := os.Getenv("TWITCH_ENDPOINT")
	if twitchEndpoint == "" {
		potentialEndpoint := twitchApi.GetTwitchEndpointUrl()
		if potentialEndpoint == "" {
			log.Fatal("could not get stream key from environment variable or twitch api")
		}
		twitchEndpoint = potentialEndpoint
	} else {
		log.Info("using TWITCH_ENDPOINT environment variable")
	}

	// initialize video storage
	storage := videostorage.New(bucketName)
	log.WithField("bucket", bucketName).Info("video storage initialized")

	var notifiers []notifier.Notifier
	if webhookUrl := os.Getenv("NOTIFICATION_WEBHOOK_URL"); webhookUrl != "" {
		log.Info("initializing webhook notifier")
		notifiers = append(notifiers, &notifier.Webhook{
			Url: webhookUrl,
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
