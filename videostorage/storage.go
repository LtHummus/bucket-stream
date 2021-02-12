package videostorage

import "io"

type Storage interface {
	// PickVideo should return a random video from storage. This should return the name of the video as well
	// as an `io.ReadCloser` to read the video
	PickVideo() (string, io.ReadCloser)
	ForceEnumerate()
	GetVideoCount() int
}
