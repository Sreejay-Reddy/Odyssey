package odyssey

import (
	"encoding/json"
	"net/http"

	"github.com/sreejay-reddy/odyssey/odyssey-go/internal/execute"
)

type Server struct {
    client *Client
}

type ExecuteRequest struct {
    Key    string `json:"key"`
    Target string `json:"target"`
}

func New(client *Client) *Server {
    return &Server{
        client: client,
    }
}

func (s *Server) Serve(addr string) error {
    mux := http.NewServeMux()

    mux.HandleFunc("/execute", s.handleExecute)
    mux.HandleFunc("/health", s.handleHealth)

    return http.ListenAndServe(addr, mux)
}

func (s *Server) handleExecute(w http.ResponseWriter, r *http.Request) {
	var req ExecuteRequest
	ctx := r.Context()

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid JSON payload.", http.StatusBadRequest)
		return
	}

	if req.Key == "" {
		http.Error(w, "Missing key.", http.StatusBadRequest)
		return
	}

	if req.Target == "" {
		http.Error(w, "Missing target.", http.StatusBadRequest)
		return
	}

	conn, err := s.client.connect(ctx)
	if err != nil {
		http.Error(
			w,
			"Failed to connect to database.",
			http.StatusInternalServerError,
		)
		return
	}
	defer conn.Close(ctx)

	response, success, err := execute.Execute(
		ctx,
		conn,
		req.Key,
		req.Target,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !success {
		http.Error(
			w,
			"Execution was not completed.",
			http.StatusConflict,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(map[string]any{
		"status": "completed",
		"key":    req.Key,
		"target": req.Target,
		"result": response,
	}); err != nil {
		return
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}


