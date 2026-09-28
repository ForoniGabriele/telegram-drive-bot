// internal/healthcheck/server.go
package healthcheck

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
)

// Start avvia un server HTTP minimo per le verifiche di integrità di Render
func Start() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "10000" // porta predefinita di Render
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})

	go func() {
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			slog.Error("healthcheck server failed", "error", err)
		}
	}()
}
