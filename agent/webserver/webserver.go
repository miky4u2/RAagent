package webserver

import (
	"github.com/miky4u2/RAagent/agent/config"
	"github.com/miky4u2/RAagent/agent/webserver/handler"
	"github.com/miky4u2/RAagent/agent/webserver/handler/tasks"
	"golang.org/x/time/rate"
	"net/http"
	"path/filepath"
)

var limiter *rate.Limiter

// Start HTTP Server
//
func Start() error {

	// Create a new Middleware rate limiter
	limiter = rate.NewLimiter(rate.Limit(config.Settings.RateLimit), config.Settings.RateLimitBurst)

	// Set routing
	mux := http.NewServeMux()
	mux.HandleFunc("/tasks/new", tasks.New)
	mux.HandleFunc("/tasks/status", tasks.Status)
	mux.HandleFunc("/update", handler.Update)
	mux.HandleFunc("/ctl", handler.Ctl)

	// TLS certificate and key paths
	cert := filepath.Join(config.AppBasePath, "conf", "cert.pem")
	key := filepath.Join(config.AppBasePath, "conf", "key.pem")

	// Launch TLS HTTP server
	err := http.ListenAndServeTLS(config.Settings.AgentBindIP+`:`+config.Settings.AgentBindPort, cert, key, limit(mux))
	if err != nil {
		return err
	}
	return err
}

func limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
