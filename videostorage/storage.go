package videostorage

import (
	"context"
	"io"
)

type Storage interface {
	// PickVideo should return a random video from storage. This should return the name of the video as well
	// as an `io.ReadCloser` to read the video
	PickVideo(ctx context.Context) (string, io.ReadCloser)
	ForceEnumerate(ctx context.Context)
	GetVideoCount() int
}
