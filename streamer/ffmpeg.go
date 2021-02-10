package streamer

import (
	"bufio"
	"io"
	"os/exec"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
)

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
func StartFfmpegStream(ffmpegPath string, twitchEndpoint string, name string, videoInput io.ReadCloser) {
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
		twitchEndpoint,
	}
	log.WithField("video", name).Info("beginning stream")

	// build the process
	r := exec.Command(ffmpegPath, command...)
	r.Stdin = videoInput // hook the video byte stream to the stdin of ffmpeg
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
