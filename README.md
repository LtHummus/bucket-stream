# BucketStream

BucketStream is a utility for streaming a collection of videos at random from an S3 bucket to a Twitch stream. Once configured with a Twitch Stream and an S3 bucket, this app will run forever, picking videos at random and streaming them to Twitch. Once one video ends, another video will be picked at random.

## Setup
This application assumes that all videos in the S3 bucket have been pre-encoded for Twitch. The advantage to this technique is the machine that this application runs on needs a minimal amount of processing power because it is just flinging bytes at Twitch without doing any actual encoding on the fly. You can pre-encode videos using the following `ffmpeg` command:
```
ffmpeg -i input.mkv -c:v libx264 -preset medium -b:v 3000k -maxrate 3000k -bufsize 6000k -vf "scale=1280:-1,format=yuv420p" -g 50 -c:a aac -b:a 128k -ac 2 -ar 44100 file.flv
```
(taken from https://trac.ffmpeg.org/wiki/EncodingForStreamingSites)

You might find some usefulness out of [bucket-filler](https://github.com/LtHummus/bucket-filler), which is what I used to prepare all the video files for my project.

## Configuration
All configuration is done via environment variables:

| Environment Variable Name          | Description                                                                                                     |
|------------------------------------|-----------------------------------------------------------------------------------------------------------------|
| `FFMPEG_PATH`                      | Path to the `ffmpeg` executable. If this is not set, it assumes `ffmpeg` is in your `$PATH`                     |
| `TWITCH_CLIENT_ID`                 | The client ID for the Twitch API (see below for more details).                                                  |
| `TWITCH_AUTH_TOKEN`                | The auth token for the Twitch API (again, see below).                                                           |
| `VIDEO_BUCKET_NAME`                | Name of the S3 bucket to source videos from.                                                                    |
| `TWITCH_ENDPOINT`                  | The Twitch endpoint you wish to use. If this is not set, the app will attempt to use the Twitch API to pull the Twitch ingestion endpoints + the user's stream key                                                                           |
| `VIDEO_ENUMERATION_PERIOD_MINUTES` | How often should the app scan for new videos in the S3 bucket. If not set, defaults to 1440 minutes (24 hours). |
| `NOTIFICATION_WEBHOOK_URL` | An optional URL to notify when a new video starts. The streamer will send an HTTP POST to this URL with a JSON dictionary with the video's title in the `name` field (e.g. `{"name":"some video title"}`.  |
| `PORT` | The port to run the internal API on (see below)

### Getting a Token

To fill in later. But tl;dr is you can create an app in Twitch's dev portal and use https://twitchapps.com/tokengen/ to get a token.

Scopes required:
* `channel:read:stream_key`
* `user:edit:broadcast`

## Internal API

bucket-stream also runs a small HTTP server with several endpoints to control behavior. By default, the server listens on port 8080 (but can be changed with the `PORT` environment variable. The following requests are handled:

| Endpoint | Description |
|----------|-------------|
| `GET /ping` | Returns a simple ok message :) |
| `GET /stats` | Gets stats about the current session. |
| `PUT /continue/no` | Tells bucket-stream to exit once the current video finishes playing |
| `PUT /continue/yes` | Tells bucket-stream to not exit once the current video finishes (essentially if you change your mind after the above command) |
| `POST /enumerate` | Rescan the S3 bucket for new videos |

