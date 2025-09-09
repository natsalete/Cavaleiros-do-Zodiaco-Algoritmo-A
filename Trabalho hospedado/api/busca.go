// api/busca.go
package api

import (
	"encoding/json"
	"net/http"

	"github.com/natsalete/Cavaleiros-do-Zodiaco-Algoritmo-A/game"
)

func BuscaHandler(w http.ResponseWriter, r *http.Request) {
	game.EnableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	g := game.NovoJogo()
	resultado := g.AStar()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resultado)
}
