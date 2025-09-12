package game

import (
	"container/heap"
	"fmt"
	"math"
	"net/http"
	"time"
)

// ---------------- Constantes ----------------
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

// ---------------- Structs ----------------
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

// ---------------- Inicialização ----------------
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
		// Novas posições das casas conforme solicitado
		Casas: []CasaZodiaco{
			{"Áries", 50, Point{5, 31}},
			{"Touro", 55, Point{5, 14}},
			{"Gêmeos", 60, Point{10, 15}},
			{"Câncer", 70, Point{10, 28}},
			{"Leão", 75, Point{14, 38}},
			{"Virgem", 80, Point{18, 30}},
			{"Libra", 85, Point{18, 10}},
			{"Escorpião", 90, Point{25, 10}},
			{"Sagitário", 95, Point{25, 27}},
			{"Capricórnio", 100, Point{32, 34}},
			{"Aquário", 110, Point{32, 18}},
			{"Peixes", 120, Point{38, 22}},
		},
		// Novas posições de entrada e saída
		Entrada:      Point{5, 38},
		GrandeMestre: Point{38, 38},
	}

	game.inicializarMapa()
	return game
}

func (g *Game) inicializarMapa() {
	g.Mapa = make([][]int, g.Size)
	
	// Inicializar todo o mapa como MONTANHOSO primeiro
	for i := range g.Mapa {
		g.Mapa[i] = make([]int, g.Size)
		for j := range g.Mapa[i] {
			g.Mapa[i][j] = MONTANHOSO
		}
	}

	// Criar caminhos entre as casas usando apenas terreno PLANO e ROCHOSO
	g.criarCaminhos()

	// Definir posições especiais
	g.Mapa[g.Entrada.X][g.Entrada.Y] = ENTRADA
	g.Mapa[g.GrandeMestre.X][g.GrandeMestre.Y] = GRANDE_MESTRE

	// Posicionar as casas do zodíaco
	for i, casa := range g.Casas {
		if casa.Posicao.X >= 0 && casa.Posicao.X < g.Size &&
			casa.Posicao.Y >= 0 && casa.Posicao.Y < g.Size {
			g.Mapa[casa.Posicao.X][casa.Posicao.Y] = CASA_ZODIACO + i
		}
	}
}

func (g *Game) criarCaminhos() {
	// Lista de todos os pontos importantes (entrada, casas, grande mestre)
	pontosImportantes := []Point{g.Entrada}
	for _, casa := range g.Casas {
		pontosImportantes = append(pontosImportantes, casa.Posicao)
	}
	pontosImportantes = append(pontosImportantes, g.GrandeMestre)

	// Criar caminhos entre todos os pontos importantes
	for i := 0; i < len(pontosImportantes); i++ {
		for j := i + 1; j < len(pontosImportantes); j++ {
			g.criarCaminhoEntrePontos(pontosImportantes[i], pontosImportantes[j])
		}
	}

	// Criar área navegável ao redor das casas (lados esquerdo e direito)
	for _, casa := range g.Casas {
		g.criarAreaNavegavel(casa.Posicao)
	}
	
	// Criar área navegável ao redor da entrada e grande mestre
	g.criarAreaNavegavel(g.Entrada)
	g.criarAreaNavegavel(g.GrandeMestre)
}

func (g *Game) criarCaminhoEntrePontos(inicio, fim Point) {
	// Usar algoritmo simples para criar caminho em linha reta com variações
	x1, y1 := inicio.X, inicio.Y
	x2, y2 := fim.X, fim.Y

	// Diferenças
	dx := x2 - x1
	dy := y2 - y1

	// Número de passos
	passos := int(math.Max(math.Abs(float64(dx)), math.Abs(float64(dy))))

	if passos == 0 {
		return
	}

	// Incrementos
	stepX := float64(dx) / float64(passos)
	stepY := float64(dy) / float64(passos)

	// Criar o caminho
	for i := 0; i <= passos; i++ {
		x := int(math.Round(float64(x1) + stepX*float64(i)))
		y := int(math.Round(float64(y1) + stepY*float64(i)))

		if x >= 0 && x < g.Size && y >= 0 && y < g.Size {
			// Alternar entre PLANO e ROCHOSO para variedade
			if (x+y)%2 == 0 {
				g.Mapa[x][y] = PLANO
			} else {
				g.Mapa[x][y] = ROCHOSO
			}
		}
	}
}

func (g *Game) criarAreaNavegavel(centro Point) {
	// Criar uma área 5x5 ao redor do ponto central
	for dx := -2; dx <= 2; dx++ {
		for dy := -2; dy <= 2; dy++ {
			x := centro.X + dx
			y := centro.Y + dy

			if x >= 0 && x < g.Size && y >= 0 && y < g.Size {
				// Só modificar se ainda for terreno montanhoso
				if g.Mapa[x][y] == MONTANHOSO {
					// Alternar entre PLANO e ROCHOSO
					if (x+y)%2 == 0 {
						g.Mapa[x][y] = PLANO
					} else {
						g.Mapa[x][y] = ROCHOSO
					}
				}
			}
		}
	}

	// Criar corredores laterais (esquerda e direita)
	for i := -5; i <= 5; i++ {
		// Corredor à esquerda
		if centro.Y-3 >= 0 {
			x := centro.X + i
			y := centro.Y - 3
			if x >= 0 && x < g.Size && g.Mapa[x][y] == MONTANHOSO {
				if (x+y)%2 == 0 {
					g.Mapa[x][y] = PLANO
				} else {
					g.Mapa[x][y] = ROCHOSO
				}
			}
		}

		// Corredor à direita
		if centro.Y+3 < g.Size {
			x := centro.X + i
			y := centro.Y + 3
			if x >= 0 && x < g.Size && g.Mapa[x][y] == MONTANHOSO {
				if (x+y)%2 == 0 {
					g.Mapa[x][y] = PLANO
				} else {
					g.Mapa[x][y] = ROCHOSO
				}
			}
		}
	}
}

// ---------------- Utilidades ----------------
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

// ---------------- Algoritmo A* ----------------
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

// ---------------- Utils ----------------
func EnableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}
