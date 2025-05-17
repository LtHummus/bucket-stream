package config

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var sampleConfig = `
bucket: sample-storage-bucket
streams:
  - name: sample1
    notification_urls:
    - https://www.example.com
    twitch:
      auth_token: sample-auth-token
      client_id: sample-client-id
      client_secret: sample-client-secret
      refresh_token: sample-refresh-token
  - name: sample2 
    endpoint: rtmp://example.com/foo/bar
`

func TestConfig(t *testing.T) {
	viper.SetConfigType("yaml")
	err := viper.ReadConfig(strings.NewReader(sampleConfig))
	require.NoError(t, err)

	var c Configuration
	err = viper.Unmarshal(&c)
	require.NoError(t, err)

	assert.Equal(t, "sample-storage-bucket", c.Bucket)
	assert.Len(t, c.Streams, 2)

	assert.Equal(t, "sample1", c.Streams[0].Name)
	assert.Equal(t, []string{"https://www.example.com"}, c.Streams[0].NotificationURLs)
	assert.Empty(t, c.Streams[0].Endpoint)
	assert.Equal(t, "sample-auth-token", c.Streams[0].TwitchCredentials.AuthToken)
	assert.Equal(t, "sample-client-id", c.Streams[0].TwitchCredentials.ClientID)
	assert.Equal(t, "sample-client-secret", c.Streams[0].TwitchCredentials.ClientSecret)
	assert.Equal(t, "sample-refresh-token", c.Streams[0].TwitchCredentials.RefreshToken)

	assert.Equal(t, "sample2", c.Streams[1].Name)
	assert.Empty(t, c.Streams[1].NotificationURLs)
	assert.Equal(t, "rtmp://example.com/foo/bar", c.Streams[1].Endpoint)
}
