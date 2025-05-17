package videostorage

import (
	"context"
	"io"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type VideoStorage struct {
	sync.Mutex

	bucket     string
	client     *s3.Client
	downloader *manager.Downloader

	videos     *[]string
	videoCount int
}

var _ Storage = &VideoStorage{}

// New constructs a new video storage that reads from an S3 bucket given by parameter. This constructor will
// construct the struct as well as kick off an update thread that periodically polls the S3 bucket for videos.
// Any object without the .flv extension is ignored. The polling period defaults to once every 24 hours, but can
// be overridden by the VIDEO_ENUMERATION_PERIOD_MINUTES environment variable
func New(ctx context.Context, bucket string) *VideoStorage {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		panic(err)
	}

	s3Client := s3.NewFromConfig(cfg)

	vs := &VideoStorage{
		bucket:     bucket,
		client:     s3Client,
		downloader: manager.NewDownloader(s3Client),
	}

	videoEnumerationPeriodMinutes := 24 * 60

	if configPeriod := viper.GetInt("video_enumeration_period_minutes"); configPeriod != 0 {
		videoEnumerationPeriodMinutes = configPeriod
	}

	log.WithFields(log.Fields{
		"bucket":                bucket,
		"update_period_minutes": videoEnumerationPeriodMinutes,
	}).Info("initializing update thread")

	vs.ForceEnumerate(ctx)

	go func() {
		log.WithField("bucket", vs.bucket).Info("starting update background thread")
		updateTicker := time.NewTicker(time.Duration(videoEnumerationPeriodMinutes) * time.Minute)

		for {
			<-updateTicker.C
			vs.ForceEnumerate(context.Background())
		}
	}()

	return vs
}

func (vs *VideoStorage) PickVideo(ctx context.Context) (string, io.ReadCloser) {
	vs.Lock()
	winnerIdx := rand.Intn(vs.videoCount)
	winnerVideo := (*vs.videos)[winnerIdx]
	vs.Unlock()

	return winnerVideo, vs.getBuffer(ctx, winnerVideo)
}

func (vs *VideoStorage) GetVideoCount() int {
	vs.Lock()
	defer vs.Unlock()

	return vs.videoCount
}

// ForceEnumerate retrieves all the objects in a bucket and keeps track of all the objects with keys ending in .flv. This
// is designed to be run at construction of the struct + every once in a while (defaults every 24 hours, but can be
// customized).
func (vs *VideoStorage) ForceEnumerate(ctx context.Context) {
	log.WithField("bucket", vs.bucket).Info("starting video enumeration")
	res := make([]string, 0)

	var continuationToken *string
	for {
		log.WithField("continuation_token", continuationToken).Debug("sending listobjects request")
		lor, err := vs.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
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
func (vs *VideoStorage) getBuffer(ctx context.Context, key string) io.ReadCloser {
	res, err := vs.client.GetObject(ctx, &s3.GetObjectInput{
		Key:    &key,
		Bucket: &vs.bucket,
	})
	if err != nil {
		log.WithError(err).Fatal("error getting object")
	}
	return res.Body
}
