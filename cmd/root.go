package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/lthummus/bucket-stream/config"
	"github.com/lthummus/bucket-stream/server"
	"github.com/lthummus/bucket-stream/streamer"
	"github.com/lthummus/bucket-stream/videostorage"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCommand.AddCommand(authCommand)
}

var rootCommand = &cobra.Command{
	Use: "bucket-stream",
	Run: func(cmd *cobra.Command, args []string) {
		log.SetFormatter(&log.TextFormatter{
			FullTimestamp: true,
		})
		log.Info("hello world!")

		config.ReadConfig()

		// get the ffmpeg path
		ffmpegPath := viper.GetString("ffmpeg.path")
		if ffmpegPath == "" {
			log.Warn("FFMPEG_PATH not set. I hope `ffmpeg` is in your $PATH!")
			ffmpegPath = "ffmpeg"
		}

		storage := videostorage.New(context.Background(), config.ReadConfiguration.S3.Bucket, config.ReadConfiguration.S3.Region)

		streamers := make([]*streamer.Streamer, len(config.ReadConfiguration.Streams))
		for i, curr := range config.ReadConfiguration.Streams {
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
	},
}

func Execute() {
	if err := rootCommand.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}
}
