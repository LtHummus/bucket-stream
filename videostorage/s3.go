package videostorage

import (
	"io"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"


	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/aws/session"
)

type videoStorage struct {
	sync.Mutex

	bucket     string
	client     *s3.S3
	downloader *s3manager.Downloader

	videos     *[]string
	videoCount int
}

var _ Storage = &videoStorage{}

// New constructs a new video storage that reads from an S3 bucket given by parameter. This constructor will
// construct the struct as well as kick off an update thread that periodically polls the S3 bucket for videos.
// Any object without the .flv extension is ignored. The polling period defaults to once every 24 hours, but can
// be overridden by the VIDEO_ENUMERATION_PERIOD_MINUTES environment variable
func New(bucket string) *videoStorage {
	manager := s3.New(session.Must(session.NewSession()))
	vs := &videoStorage{
		bucket:     bucket,
		client:     manager,
		downloader: s3manager.NewDownloaderWithClient(manager),
	}

	videoEnumerationPeriodMinutes := 24 * 60

	if enumerationUpdate, err := strconv.Atoi(os.Getenv("VIDEO_ENUMERATION_PERIOD_MINUTES")); err == nil {
		videoEnumerationPeriodMinutes = enumerationUpdate
	}

	log.WithFields(log.Fields{
		"bucket": bucket,
		"update_period_minutes": videoEnumerationPeriodMinutes,
	}).Info("initializing update thread")

	vs.enumerate()


	go func() {
		log.WithField("bucket", vs.bucket).Info("starting update background thread")
		updateTicker := time.NewTicker(time.Duration(videoEnumerationPeriodMinutes) * time.Minute)

		for {
			<-updateTicker.C
			vs.enumerate()
		}
	}()

	return vs
}

func (vs *videoStorage) PickVideo() (string, io.ReadCloser) {
	vs.Lock()
	winnerIdx := rand.Intn(vs.videoCount)
	winnerVideo := (*vs.videos)[winnerIdx]
	vs.Unlock()

	return winnerVideo, vs.getBuffer(winnerVideo)
}

// enumerate retrieves all the objects in a bucket and keeps track of all the objects with keys ending in .flv. This
// is designed to be run at construction of the struct + every once in a while (defaults every 24 hours, but can be
// customized).
func (vs *videoStorage) enumerate() {
	log.WithField("bucket", vs.bucket).Info("starting video enumeration")
	res := make([]string, 0)

	var continuationToken *string
	for {
		log.WithField("continuation_token", continuationToken).Debug("sending listobjects request")
		lor, err := vs.client.ListObjectsV2(&s3.ListObjectsV2Input{
			Bucket:            &vs.bucket,
			ContinuationToken: continuationToken,
		})
		if err != nil {
			panic(err)
		}

		for _, curr := range lor.Contents {
			if strings.HasSuffix(*curr.Key, ".flv") {
				res = append(res, *curr.Key)
			} else {
				log.WithFields(log.Fields{
					"bucket": vs.bucket,
					"key":    *curr.Key,
				}).Warn("skipping as it is not a valid video")
			}
		}

		if !*lor.IsTruncated {
			break
		}

		continuationToken = lor.ContinuationToken
	}

	vs.Lock()
	defer vs.Unlock()
	vs.videos = &res
	vs.videoCount = len(res)
	log.WithFields(log.Fields{
		"bucket": vs.bucket,
		"count":  vs.videoCount,
	}).Info("finished video enumeration")
}

// getBuffer pulls the object info for the given key and opens an `io.ReadCloser` for the object
func (vs *videoStorage) getBuffer(key string) io.ReadCloser {
	res, err := vs.client.GetObject(&s3.GetObjectInput{
		Key:    &key,
		Bucket: &vs.bucket,
	})
	if err != nil {
		log.WithError(err).Fatal("error getting object")
	}
	return res.Body
}
