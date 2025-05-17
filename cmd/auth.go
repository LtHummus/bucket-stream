package cmd

import (
	"fmt"
	"os"

	"github.com/lthummus/bucket-stream/config"
	"github.com/lthummus/bucket-stream/twitch"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var streamName string

func init() {
	authCommand.Flags().StringVarP(&streamName, "stream", "s", "", "name of stream to authenticate")

	authCommand.MarkFlagRequired("stream")
}

func getStream(config *config.Configuration) *config.StreamConfiguration {
	for _, curr := range config.Streams {
		if curr.Name == streamName {
			return &curr
		}
	}

	return nil
}

var authCommand = &cobra.Command{
	Use: "auth",
	Run: func(cmd *cobra.Command, args []string) {
		config.ReadConfig()

		streamConfig := getStream(&config.ReadConfiguration)
		if streamConfig == nil {
			fmt.Fprintf(os.Stderr, "no stream with name %s found in config\n", streamName)
			os.Exit(2)
		}

		if streamConfig.TwitchCredentials.ClientID == "" {
			fmt.Fprintf(os.Stderr, "missing twitch client id in stream %s\n", streamName)
			os.Exit(3)
		}

		if streamConfig.TwitchCredentials.ClientSecret == "" {
			fmt.Fprintf(os.Stderr, "missing twitch client secret in stream %s\n", streamName)
			os.Exit(4)
		}

		fmt.Printf("handling auth....\n")
		fmt.Printf("go to\n%s\n\n", twitch.GenerateAuthUrl(streamConfig.TwitchCredentials))
		fmt.Printf("Authorization code: ")

		var code string
		_, err := fmt.Scanf("%s", &code)
		if err != nil {
			log.WithError(err).Fatal("could not read input")
		}

		err = twitch.Handshake(code, streamConfig.TwitchCredentials)
		if err != nil {
			log.WithError(err).Fatal("could not update tokens")
		}
	},
}
