package streamer

import (
	"bufio"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type Streamer struct {
	sync.Mutex

	FfmpegPath     string
	TwitchEndpoint string

	VideoStart time.Time
	PlayCount  int

	video string
}

func (s *Streamer) SetVideo(video string) {
	s.Lock()
	defer s.Unlock()

	s.video = video
}

func (s *Streamer) GetVideo() string {
	s.Lock()
	defer s.Unlock()

	return s.video
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
func (s *Streamer) StartFfmpegStream(name string, videoInput io.ReadCloser) {
	s.Lock()
	s.video = name
	s.VideoStart = time.Now()
	s.PlayCount += 1
	s.Unlock()

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
		s.TwitchEndpoint,
	}
	log.WithField("video", name).Info("beginning stream")

	// build the process
	r := exec.Command(s.FfmpegPath, command...)
	r.Stdin = videoInput          // hook the video byte stream to the stdin of ffmpeg
	stderr, err := r.StderrPipe() // set up reading from ffmpeg's output
	if err != nil {
		log.WithField("video", name).WithError(err).Fatal("error opening stderr")
	}
	if err = r.Start(); err != nil {
		log.WithField("video", name).WithError(err).Fatal("error starting ffmpeg")
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		captureOutput(stderr)
		wg.Done()
	}()

	wg.Wait() // wait until stream is done
	log.WithField("video", name).Info("Waiting for process to exit")
	err = r.Wait()
	if err != nil {
		log.WithField("video", name).WithError(err).Fatal("error on wait")
	}
	// close everything
	err = videoInput.Close()
	if err != nil {
		log.WithField("video", name).WithError(err).Fatal("error closing video input")
	}
	log.WithField("video", name).Info("closed video input stream")
	log.WithField("video", name).Info("stream finished")
}
