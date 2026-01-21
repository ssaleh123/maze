package main

import (
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	GridW     = 21
	GridH     = 21
	CellSize = 20
	TickRate = 60
)

type Player struct {
	ID string  `json:"id"`
	X  int     `json:"x"`
	Y  int     `json:"y"`
}

type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	clients   = make(map[*websocket.Conn]*Player)
	clientsMu sync.Mutex

	maze     [][]int
	mazeSeed int64 = time.Now().UnixNano()
)

func main() {
	rand.Seed(mazeSeed)
	maze = generateMaze(mazeSeed)

	http.HandleFunc("/ws", handleWS)

	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

/* -------------------- WEBSOCKET -------------------- */

func handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	player := &Player{
		ID: randID(),
		X:  GridW / 2,
		Y:  GridH / 2,
	}

	clientsMu.Lock()
	clients[conn] = player
	clientsMu.Unlock()

	sendState()

	for {
		var input map[string]string
		if err := conn.ReadJSON(&input); err != nil {
			break
		}

		movePlayer(player, input["dir"])

		if isExit(player.X, player.Y) {
			resetGame()
		}

		sendState()
	}

	clientsMu.Lock()
	delete(clients, conn)
	clientsMu.Unlock()
	conn.Close()
}

/* -------------------- GAME LOGIC -------------------- */

func movePlayer(p *Player, dir string) {
	dx, dy := 0, 0
	switch dir {
	case "up":
		dy = -1
	case "down":
		dy = 1
	case "left":
		dx = -1
	case "right":
		dx = 1
	}

	nx, ny := p.X+dx, p.Y+dy
	if nx >= 0 && ny >= 0 && nx < GridW && ny < GridH && maze[ny][nx] == 0 {
		p.X, p.Y = nx, ny
	}
}

func isExit(x, y int) bool {
	return (x == 0 && y == 0) ||
		(x == GridW-1 && y == 0) ||
		(x == 0 && y == GridH-1) ||
		(x == GridW-1 && y == GridH-1)
}

func resetGame() {
	mazeSeed += rand.Int63n(9999)
	maze = generateMaze(mazeSeed)

	for _, p := range clients {
		p.X = GridW / 2
		p.Y = GridH / 2
	}
}

/* -------------------- MAZE -------------------- */

func generateMaze(seed int64) [][]int {
	rand.Seed(seed)

	m := make([][]int, GridH)
	for y := range m {
		m[y] = make([]int, GridW)
		for x := range m[y] {
			m[y][x] = 1
		}
	}

	var carve func(x, y int)
	carve = func(x, y int) {
		dirs := [][2]int{{2, 0}, {-2, 0}, {0, 2}, {0, -2}}
		rand.Shuffle(len(dirs), func(i, j int) { dirs[i], dirs[j] = dirs[j], dirs[i] })

		for _, d := range dirs {
			nx, ny := x+d[0], y+d[1]
			if nx > 0 && ny > 0 && nx < GridW-1 && ny < GridH-1 && m[ny][nx] == 1 {
				m[ny][nx] = 0
				m[y+d[1]/2][x+d[0]/2] = 0
				carve(nx, ny)
			}
		}
	}

	m[1][1] = 0
	carve(1, 1)

	// open corner exits
	m[0][0] = 0
	m[0][GridW-1] = 0
	m[GridH-1][0] = 0
	m[GridH-1][GridW-1] = 0

	// open center spawn
	m[GridH/2][GridW/2] = 0

	return m
}

/* -------------------- SYNC -------------------- */

func sendState() {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	players := []*Player{}
	for _, p := range clients {
		players = append(players, p)
	}

	state := Message{
		Type: "state",
		Data: map[string]interface{}{
			"players": players,
			"maze":    maze,
		},
	}

	for c := range clients {
		c.WriteJSON(state)
	}
}

/* -------------------- UTIL -------------------- */

func randID() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}
