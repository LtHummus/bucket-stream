package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/lthummus/bucket-stream/streamer"
	"github.com/lthummus/bucket-stream/videostorage"
)

type Server struct {
	Storage  videostorage.Storage
	Streamer *streamer.Streamer

	shouldContinue atomic.Bool
	start          time.Time
}

func (s *Server) StartServer() {
	s.shouldContinue.Store(true)
	s.start = time.Now()

	log.Info("initializing web server")

	mux := http.NewServeMux()

	mux.HandleFunc("GET /ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		w.Write([]byte(`{"message":"ok"}`))
	})
	mux.HandleFunc("PUT /continue/no", func(w http.ResponseWriter, r *http.Request) {
		s.SetContinue(false)
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("PUT /continue/yes", func(w http.ResponseWriter, r *http.Request) {
		s.SetContinue(true)
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /stats", func(w http.ResponseWriter, r *http.Request) {
		res := map[string]any{
			"total_uptime":           time.Since(s.start).String(),
			"should_continue":        s.ShouldContinue(),
			"video_count":            s.Storage.GetVideoCount(),
			"currently_playing":      s.Streamer.GetVideo(),
			"time_since_video_start": time.Since(s.Streamer.VideoStart).String(),
			"videos_played":          s.Streamer.PlayCount,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_ = json.NewEncoder(w).Encode(res)
	})

	serverPort := viper.GetInt("server.port")
	if serverPort == 0 {
		serverPort = 8088
	}

	srv := http.Server{
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  2 * time.Minute,
		Addr:         fmt.Sprintf("0.0.0.0:%d", serverPort),
		Handler:      mux,
	}

	log.WithField("port", serverPort).Info("starting server")
	err := srv.ListenAndServe()
	if err != nil {
		log.WithError(err).Error("could not start server")
	}
}

func (s *Server) SetContinue(cont bool) {
	log.WithField("new_value", cont).Info("updating continue")
	s.shouldContinue.Store(cont)
}

func (s *Server) ShouldContinue() bool {
	return s.shouldContinue.Load()
}
