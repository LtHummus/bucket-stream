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
All configuration is done via a YAML file, `bucket-stream.yaml`:

```yaml
notification_urls: # optional
  - https://example.com
s3:
  bucket: bucket-with-your-videos
twitch:
  auth_token: twitch_access_token_can_be_blank
  client_id: twitch_client_id
  client_secret: twitch_client_secret
  refresh_token: twitch_refresh_token_can_be_blank
```

### Getting a Token

Run the program with the single command line arugment `auth`. This will give a URL you can go to in order to authenticate your twitch account. The program will ask for an authorization code. Once auth'd, twitch will attempt to redirect you to http://localhost/?code=<some_string_here>. That string is what the program is looking for. The program will write your token + refresh token. Then run the app normally.

## Internal API

bucket-stream also runs a small HTTP server with several endpoints to control behavior. By default, the server listens on port 8080 (but can be changed with the `PORT` environment variable. The following requests are handled:

| Endpoint | Description |
|----------|-------------|
| `GET /ping` | Returns a simple ok message :) |
| `GET /stats` | Gets stats about the current session. |
| `PUT /continue/no` | Tells bucket-stream to exit once the current video finishes playing |
| `PUT /continue/yes` | Tells bucket-stream to not exit once the current video finishes (essentially if you change your mind after the above command) |
| `POST /enumerate` | Rescan the S3 bucket for new videos |

