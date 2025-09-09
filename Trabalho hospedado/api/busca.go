package api

import (
	"encoding/json"
	"net/http"

	"github.com/natsalete/Cavaleiros-do-Zodiaco-Algoritmo-A/game"
)

func GameHandler(w http.ResponseWriter, r *http.Request) {
	game.EnableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	g := game.NovoJogo()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(g)
}
