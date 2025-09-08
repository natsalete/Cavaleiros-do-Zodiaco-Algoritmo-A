package main

import (
	"container/heap"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"time"
)

// Tipos de terreno
const (
	MONTANHOSO    = 0
	PLANO         = 1
	ROCHOSO       = 2
	ENTRADA       = 3
	GRANDE_MESTRE = 4
	CASA_ZODIACO  = 5
)

var CUSTOS_TERRENO = map[int]int{
	MONTANHOSO: 200,
	PLANO:      1,
	ROCHOSO:    5,
}

type CavaleiroBronze struct {
	Nome         string  `json:"nome"`
	PoderCosmico float64 `json:"poder_cosmico"`
	Energia      int     `json:"energia"`
}

type CasaZodiaco struct {
	Nome        string `json:"nome"`
	Dificuldade int    `json:"dificuldade"`
	Posicao     Point  `json:"posicao"`
}

type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Node struct {
	Point
	G       int
	H       int
	F       int
	Parent  *Node
	CasaID  int
	Visited []bool
}

type PriorityQueue []*Node

func (pq PriorityQueue) Len() int           { return len(pq) }
func (pq PriorityQueue) Less(i, j int) bool { return pq[i].F < pq[j].F }
func (pq PriorityQueue) Swap(i, j int)      { pq[i], pq[j] = pq[j], pq[i] }

func (pq *PriorityQueue) Push(x interface{}) {
	*pq = append(*pq, x.(*Node))
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[0 : n-1]
	return item
}

type Game struct {
	Mapa         [][]int           `json:"mapa"`
	Cavaleiros   []CavaleiroBronze `json:"cavaleiros"`
	Casas        []CasaZodiaco     `json:"casas"`
	Entrada      Point             `json:"entrada"`
	GrandeMestre Point             `json:"grande_mestre"`
	Size         int               `json:"size"`
}

type ResultadoBusca struct {
	Sucesso      bool          `json:"sucesso"`
	Caminho      []Point       `json:"caminho"`
	CustoTotal   int           `json:"custo_total"`
	Duracao      string        `json:"duracao"`
	Estatisticas Estatisticas  `json:"estatisticas"`
}

type Estatisticas struct {
	TamanhoCaminho     int     `json:"tamanho_caminho"`
	CustoMedioPorPasso float64 `json:"custo_medio_por_passo"`
	CasasVisitadas     []bool  `json:"casas_visitadas"`
	TempoExecucao      string  `json:"tempo_execucao"`
}

func NovoJogo() *Game {
	game := &Game{
		Size: 42,
		Cavaleiros: []CavaleiroBronze{
			{"Seiya", 1.5, 5},
			{"Shiryu", 1.4, 5},
			{"Hyoga", 1.3, 5},
			{"Shun", 1.2, 5},
			{"Ikki", 1.1, 5},
		},
		Casas: []CasaZodiaco{
			{"Áries", 50, Point{5, 5}},
			{"Touro", 55, Point{10, 8}},
			{"Gêmeos", 60, Point{15, 12}},
			{"Câncer", 70, Point{20, 16}},
			{"Leão", 75, Point{25, 20}},
			{"Virgem", 80, Point{30, 24}},
			{"Libra", 85, Point{35, 28}},
			{"Escorpião", 90, Point{32, 32}},
			{"Sagitário", 95, Point{28, 36}},
			{"Capricórnio", 100, Point{24, 38}},
			{"Aquário", 110, Point{20, 40}},
			{"Peixes", 120, Point{15, 41}},
		},
		Entrada:      Point{41, 20},
		GrandeMestre: Point{1, 41},
	}

	game.inicializarMapa()
	return game
}

func (g *Game) inicializarMapa() {
	g.Mapa = make([][]int, g.Size)
	for i := range g.Mapa {
		g.Mapa[i] = make([]int, g.Size)
		for j := range g.Mapa[i] {
			if (i+j)%3 == 0 {
				g.Mapa[i][j] = MONTANHOSO
			} else if (i+j)%3 == 1 {
				g.Mapa[i][j] = PLANO
			} else {
				g.Mapa[i][j] = ROCHOSO
			}
		}
	}

	g.Mapa[g.Entrada.X][g.Entrada.Y] = ENTRADA
	g.Mapa[g.GrandeMestre.X][g.GrandeMestre.Y] = GRANDE_MESTRE

	for i, casa := range g.Casas {
		if casa.Posicao.X >= 0 && casa.Posicao.X < g.Size &&
			casa.Posicao.Y >= 0 && casa.Posicao.Y < g.Size {
			g.Mapa[casa.Posicao.X][casa.Posicao.Y] = CASA_ZODIACO + i
		}
	}
}

func distanciaManhattan(a, b Point) int {
	return int(math.Abs(float64(a.X-b.X)) + math.Abs(float64(a.Y-b.Y)))
}

func (g *Game) posicaoValida(p Point) bool {
	return p.X >= 0 && p.X < g.Size && p.Y >= 0 && p.Y < g.Size
}

func (g *Game) obterVizinhos(p Point) []Point {
	vizinhos := []Point{
		{p.X - 1, p.Y},
		{p.X + 1, p.Y},
		{p.X, p.Y - 1},
		{p.X, p.Y + 1},
	}

	var validosResult []Point
	for _, v := range vizinhos {
		if g.posicaoValida(v) {
			validosResult = append(validosResult, v)
		}
	}
	return validosResult
}

func (g *Game) custoMovimento(p Point) int {
	terreno := g.Mapa[p.X][p.Y]

	if terreno >= CASA_ZODIACO {
		return CUSTOS_TERRENO[PLANO]
	}

	if custo, existe := CUSTOS_TERRENO[terreno]; existe {
		return custo
	}
	return CUSTOS_TERRENO[PLANO]
}

func (g *Game) tempoBatalha(casaID int, cavaleirosParticipantes []int) float64 {
	if casaID < 0 || casaID >= len(g.Casas) {
		return 0
	}

	casa := g.Casas[casaID]
	somaPoderCosmico := 0.0

	for _, idx := range cavaleirosParticipantes {
		if idx >= 0 && idx < len(g.Cavaleiros) && g.Cavaleiros[idx].Energia > 0 {
			somaPoderCosmico += g.Cavaleiros[idx].PoderCosmico
		}
	}

	if somaPoderCosmico == 0 {
		return math.Inf(1)
	}

	return float64(casa.Dificuldade) / somaPoderCosmico
}

func todasCasasVisitadas(visited []bool) bool {
	for _, v := range visited {
		if !v {
			return false
		}
	}
	return true
}

func (g *Game) AStar() ResultadoBusca {
	inicio := time.Now()

	openSet := &PriorityQueue{}
	heap.Init(openSet)

	inicialVisited := make([]bool, len(g.Casas))
	inicial := &Node{
		Point:   g.Entrada,
		G:       0,
		H:       distanciaManhattan(g.Entrada, g.GrandeMestre),
		F:       distanciaManhattan(g.Entrada, g.GrandeMestre),
		Parent:  nil,
		CasaID:  -1,
		Visited: inicialVisited,
	}

	heap.Push(openSet, inicial)
	visited := make(map[string]*Node)

	for openSet.Len() > 0 {
		atual := heap.Pop(openSet).(*Node)

		chave := fmt.Sprintf("%d,%d,%v", atual.X, atual.Y, atual.Visited)

		if existente, existe := visited[chave]; existe {
			if existente.G <= atual.G {
				continue
			}
		}
		visited[chave] = atual

		if atual.Point == g.GrandeMestre && todasCasasVisitadas(atual.Visited) {
			var caminho []Point
			no := atual
			custoTotal := atual.G

			for no != nil {
				caminho = append([]Point{no.Point}, caminho...)
				no = no.Parent
			}

			duracao := time.Since(inicio)
			
			return ResultadoBusca{
				Sucesso:    true,
				Caminho:    caminho,
				CustoTotal: custoTotal,
				Duracao:    duracao.String(),
				Estatisticas: Estatisticas{
					TamanhoCaminho:     len(caminho),
					CustoMedioPorPasso: float64(custoTotal) / float64(len(caminho)),
					CasasVisitadas:     atual.Visited,
					TempoExecucao:      duracao.String(),
				},
			}
		}

		for _, vizinho := range g.obterVizinhos(atual.Point) {
			custoMovimento := g.custoMovimento(vizinho)
			novoG := atual.G + custoMovimento

			terreno := g.Mapa[vizinho.X][vizinho.Y]
			casaID := -1
			novasVisited := make([]bool, len(atual.Visited))
			copy(novasVisited, atual.Visited)

			if terreno >= CASA_ZODIACO {
				casaID = terreno - CASA_ZODIACO
				if casaID < len(g.Casas) {
					cavaleirosDisponiveis := []int{}
					for i := range g.Cavaleiros {
						if g.Cavaleiros[i].Energia > 0 {
							cavaleirosDisponiveis = append(cavaleirosDisponiveis, i)
						}
					}

					if len(cavaleirosDisponiveis) > 0 {
						tempoBatalha := g.tempoBatalha(casaID, cavaleirosDisponiveis)
						novoG += int(tempoBatalha)
						novasVisited[casaID] = true
					}
				}
			}

			h := distanciaManhattan(vizinho, g.GrandeMestre)

			casasRestantes := 0
			for _, visitada := range novasVisited {
				if !visitada {
					casasRestantes++
				}
			}
			h += casasRestantes * 50

			novoNo := &Node{
				Point:   vizinho,
				G:       novoG,
				H:       h,
				F:       novoG + h,
				Parent:  atual,
				CasaID:  casaID,
				Visited: novasVisited,
			}

			heap.Push(openSet, novoNo)
		}
	}

	duracao := time.Since(inicio)
	return ResultadoBusca{
		Sucesso: false,
		Duracao: duracao.String(),
	}
}

// Handlers HTTP
func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func getGameState(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	game := NovoJogo()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(game)
}

func executarBusca(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	game := NovoJogo()
	resultado := game.AStar()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resultado)
}

func serveStatic(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		http.ServeFile(w, r, "index.html")
	} else {
		http.NotFound(w, r)
	}
}

func main() {
	fmt.Println("🌟 Servidor Cavaleiros do Zodíaco iniciando...")
	fmt.Println("🌐 Acesse: http://localhost:8080")

	http.HandleFunc("/", serveStatic)
	http.HandleFunc("/api/game", getGameState)
	http.HandleFunc("/api/busca", executarBusca)

	fmt.Println("🚀 Servidor rodando na porta 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}