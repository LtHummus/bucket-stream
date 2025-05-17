package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/lthummus/bucket-stream/streamer"
	"github.com/lthummus/bucket-stream/videostorage"
)

type Server struct {
	Storage  videostorage.Storage
	Streamer []*streamer.Streamer

	start time.Time
}

func (s *Server) StartServer() {
	s.start = time.Now()

	log.Info("initializing web server")

	mux := http.NewServeMux()

	mux.HandleFunc("GET /ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		w.Write([]byte(`{"message":"ok"}`))
	})
	mux.HandleFunc("GET /stats", func(w http.ResponseWriter, r *http.Request) {
		streamerData := make([]map[string]any, len(s.Streamer))
		for i, curr := range s.Streamer {
			m := map[string]any{
				"currently_playing":      curr.GetVideo(),
				"time_since_video_start": time.Since(curr.GetVideoStart()),
				"videos_played":          curr.PlayCount(),
			}
			streamerData[i] = m
		}

		res := map[string]any{
			"total_uptime": time.Since(s.start).String(),
			"video_count":  s.Storage.GetVideoCount(),
			"streams":      streamerData,
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
