package api

import (
	"encoding/json"
	"net/http"

	"../game"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	game.EnableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	// inicializa o jogo
	g := game.NovoJogo()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(g)
}
