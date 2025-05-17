package streamer

import (
	"bufio"
	"context"
	"io"
	"os/exec"
	"path"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/lthummus/bucket-stream/config"
	"github.com/lthummus/bucket-stream/notifier"
	"github.com/lthummus/bucket-stream/twitch"
	"github.com/lthummus/bucket-stream/videostorage"
)

type Streamer struct {
	lock *sync.Mutex

	name string

	storage          videostorage.Storage
	notificationURLs []notifier.Notifier

	ffmpegPath     string
	streamEndpoint string
	twitch         *twitch.Api

	videoStart time.Time
	playCount  int

	video string
}

func New(cfg config.StreamConfiguration,
	storage videostorage.Storage,
	ffmpegPath string) *Streamer {

	if cfg.Endpoint == "" && cfg.TwitchCredentials.ClientID == "" {
		log.WithField("name", cfg.Name).Fatal("one of stream endpoint or twitch credentials should be set")
	}

	var tAPI *twitch.Api

	trueEndpoint := cfg.Endpoint

	if cfg.TwitchCredentials.ClientID != "" {
		tAPI = &twitch.Api{Credentials: &cfg.TwitchCredentials}
		tAPI.GetUserInfo()
		trueEndpoint = tAPI.GetTwitchEndpointUrl()
		log.WithFields(log.Fields{
			"name": cfg.Name,
		}).Info("using twitch")
	}

	var notifiers []notifier.Notifier
	for _, curr := range cfg.NotificationURLs {
		notifiers = append(notifiers, &notifier.Webhook{
			Url: curr,
		})
	}

	return &Streamer{
		lock:             &sync.Mutex{},
		name:             cfg.Name,
		storage:          storage,
		notificationURLs: notifiers,
		ffmpegPath:       ffmpegPath,
		streamEndpoint:   trueEndpoint,
		twitch:           tAPI,
	}
}

func (s *Streamer) Run() {
	for {
		log.WithField("name", s.name).Info("starting cycle")
		pickedVideo, buf := s.storage.PickVideo(context.Background())
		log.WithFields(log.Fields{
			"name":         s.name,
			"picked_video": pickedVideo,
		}).Info("selected winner")

		streamTitle := strings.TrimPrefix(strings.TrimSuffix(path.Base(pickedVideo), path.Ext(pickedVideo)), "/")
		if s.twitch != nil {
			go s.twitch.UpdateStreamTitle(streamTitle)
		}

		for _, curr := range s.notificationURLs {
			go curr.Notify(streamTitle)
		}

		log.WithFields(log.Fields{
			"name":  s.name,
			"video": pickedVideo,
		}).Info("opened stream")
		s.StartFfmpegStream(pickedVideo, buf)
		log.WithFields(log.Fields{
			"name":  s.name,
			"video": pickedVideo,
		}).Info("cycle complete")
	}
}

func (s *Streamer) SetVideo(video string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.video = video
}

func (s *Streamer) GetVideo() string {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.video
}

func (s *Streamer) GetVideoStart() time.Time {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.videoStart
}

func (s *Streamer) PlayCount() int {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.playCount
}

func captureOutput(r io.Reader) {
	reader := bufio.NewReader(r)
	var line string
	var err error
	for {
		line, err = reader.ReadString('\n')
		if err != nil && err != io.EOF {
			log.WithError(err).Warn("unable to read ffmpeg output")
			break
		}
		line = strings.TrimSpace(line)
		if line != "" {
			log.Warn(line)
		}
		if err != nil {
			break
		}
	}
}

// StartFfmpegStream starts streaming to twitch. This requires a path to the ffmpeg executable, the twitch endpoint,
// the video's name (for logging) and an `io.ReadCloser` to read video data from. The video is assumed to be in an
// FLV container with codecs that Twitch is happy with (see README for more details).
func (s *Streamer) StartFfmpegStream(videoName string, videoInput io.ReadCloser) {
	s.lock.Lock()
	s.video = videoName
	s.videoStart = time.Now()
	s.playCount += 1
	s.lock.Unlock()

	var command = []string{
		"-loglevel", // only log warnings
		"warning",
		"-hide_banner", // don't bother echoing out the codecs and build information
		"-re",          // do this in real time
		"-i",           // read from stdin
		"-",
		"-c", // don't actually encode
		"copy",
		"-f", // output format
		"flv",
		"-flvflags", // don't complain about not being
		"no_duration_filesize",
		s.streamEndpoint,
	}
	log.WithField("name", s.name).WithField("video_name", videoName).Info("beginning stream")

	// build the process
	r := exec.Command(s.ffmpegPath, command...)
	r.Stdin = videoInput          // hook the video byte stream to the stdin of ffmpeg
	stderr, err := r.StderrPipe() // set up reading from ffmpeg's output
	if err != nil {
		log.WithField("video", videoName).WithError(err).Fatal("error opening stderr")
	}
	if err = r.Start(); err != nil {
		log.WithField("name", s.name).WithField("video_name", videoName).WithError(err).Fatal("error starting ffmpeg")
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		captureOutput(stderr)
		wg.Done()
	}()

	wg.Wait() // wait until stream is done
	log.WithField("name", s.name).WithField("video_name", videoName).Info("Waiting for process to exit")
	err = r.Wait()
	if err != nil {
		log.WithField("video", videoName).WithError(err).Fatal("error on wait")
	}
	// close everything
	err = videoInput.Close()
	if err != nil {
		log.WithField("video", videoName).WithError(err).Fatal("error closing video input")
	}
	log.WithField("name", s.name).WithField("video_name", videoName).Info("closed video input stream")
	log.WithField("name", s.name).WithField("video_name", videoName).Info("stream finished")
}
