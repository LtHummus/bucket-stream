package server

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/toorop/gin-logrus"

	"github.com/lthummus/bucket-stream/streamer"
	"github.com/lthummus/bucket-stream/videostorage"
)

type Server struct {
	sync.Mutex

	Storage  videostorage.Storage
	Streamer *streamer.Streamer

	shouldContinue bool
	start          time.Time
}

func (s *Server) StartServer() {
	s.shouldContinue = true
	s.start = time.Now()

	log.Info("initializing web server")

	ginLogger := log.New()
	ginLogger.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(ginlogrus.Logger(ginLogger), gin.Recovery())

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "ok",
		})
	})
	r.PUT("/continue/no", func(c *gin.Context) {
		s.SetContinue(false)
		c.JSON(200, gin.H{
			"message": "ok",
		})
	})
	r.PUT("/continue/yes", func(c *gin.Context) {
		s.SetContinue(true)
		c.JSON(200, gin.H{
			"message": "ok",
		})
	})
	r.GET("/stats", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"total_uptime":           time.Since(s.start).String(),
			"should_continue":        s.ShouldContinue(),
			"video_count":            s.Storage.GetVideoCount(),
			"currently_playing":      s.Streamer.GetVideo(),
			"time_since_video_start": time.Since(s.Streamer.VideoStart).String(),
			"videos_played":          s.Streamer.PlayCount,
		})
	})
	r.POST("/enumerate", func(c *gin.Context) {
		s.Storage.ForceEnumerate()
		c.JSON(200, gin.H{
			"message":     "ok",
			"video_count": s.Storage.GetVideoCount(),
		})
	})

	err := r.Run()
	if err != nil {
		log.WithError(err).Error("web server failed")
	}
}

func (s *Server) ShouldContinue() bool {
	s.Lock()
	defer s.Unlock()
	return s.shouldContinue
}

func (s *Server) SetContinue(cont bool) {
	s.Lock()
	defer s.Unlock()
	s.shouldContinue = cont
}
