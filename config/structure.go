package config

type TwitchCredentials struct {
	AuthToken    string `mapstructure:"auth_token"`
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	RefreshToken string `mapstructure:"refresh_token"`
}

type Configuration struct {
	Bucket  string `mapstructure:"bucket"`
	Streams []struct {
		Name              string            `mapstructure:"name"`
		NotificationURLs  []string          `mapstructure:"notification_urls"`
		Endpoint          string            `mapstrucutre:"endpoint"`
		TwitchCredentials TwitchCredentials `mapstructure:"twitch"`
	} `mapstructure:"streams"`
}
