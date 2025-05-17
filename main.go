package main

import (
	"context"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/lthummus/bucket-stream/config"
	"github.com/lthummus/bucket-stream/server"

	"github.com/lthummus/bucket-stream/streamer"
	"github.com/lthummus/bucket-stream/twitch"
	"github.com/lthummus/bucket-stream/videostorage"
)

func handleAuth() {
	panic("currently broken while i refactor")
	//fmt.Printf("handling auth...\n")
	//fmt.Printf("go to\n%s\n\n", twitch.GenerateAuthUrl())
	//fmt.Printf("Authorization code: ")
	//
	//var code string
	//_, err := fmt.Scanf("%s", &code)
	//if err != nil {
	//	log.WithError(err).Warn("could not read input")
	//}
	//
	//err = twitch.Handshake(code)
	//if err != nil {
	//	log.WithError(err).Warn("could not update tokens")
	//}

}

func main() {
	// set up logging and initialize the RNG
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	log.Info("hello world!")

	config.ReadConfig()

	var fullConfig config.Configuration
	err := viper.Unmarshal(&fullConfig)
	if err != nil {
		log.WithError(err).Fatal("could not parse config")
	}

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

	// read the source bucket
	bucketName := viper.GetString("s3.bucket")
	if bucketName == "" {
		log.Fatal("environment variable VIDEO_BUCKET_NAME is empty")
	}

	region := viper.GetString("s3.region")
	if region == "" {
		log.Fatal("region not set in config file")
	}

	// initialize video storage
	storage := videostorage.New(context.Background(), bucketName, region)
	log.WithField("bucket", bucketName).Info("video storage initialized")

	streamers := make([]*streamer.Streamer, len(fullConfig.Streams))
	for i, curr := range fullConfig.Streams {
		streamers[i] = streamer.New(curr, storage, ffmpegPath)
	}

	// start server
	srv := &server.Server{
		Storage:  storage,
		Streamer: streamers,
	}

	for _, curr := range streamers {
		go curr.Run()
	}

	srv.StartServer()

}
