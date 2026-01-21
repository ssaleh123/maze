package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)




const (
	GridSize   = 21 // must be odd
	CellSize   = 24
	PlayerSize = 6
)

type Player struct {
	ID string  `json:"id"`
	X  float64 `json:"x"`
	Y  float64 `json:"y"`
}

type GameState struct {
	Maze    [][]int          `json:"maze"`
	Players map[string]*Player `json:"players"`
}

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	clients   = make(map[*websocket.Conn]string)
	players   = make(map[string]*Player)
	maze      [][]int
	mu        sync.Mutex
)

func main() {
	rand.Seed(time.Now().UnixNano())
	maze = generateMaze(nil)

	http.HandleFunc("/", serveHTML)
	http.HandleFunc("/ws", wsHandler)

	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

/* =========================
   WebSocket
========================= */

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, _ := upgrader.Upgrade(w, r, nil)
	id := randID()

	mu.Lock()
	players[id] = spawnPlayer(id)
	clients[conn] = id
	mu.Unlock()

	defer func() {
		mu.Lock()
		delete(players, id)
		delete(clients, conn)
		mu.Unlock()
		conn.Close()
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var input struct {
			DX float64 `json:"dx"`
			DY float64 `json:"dy"`
		}
		json.Unmarshal(msg, &input)

		mu.Lock()
		p := players[id]
		tryMove(p, input.DX, input.DY)

		if isExit(p) {
			maze = generateMaze(maze)
			for _, pl := range players {
				pl.X = float64(GridSize*CellSize) / 2
				pl.Y = float64(GridSize*CellSize) / 2
			}
		}

		broadcast()
		mu.Unlock()
	}
}

func broadcast() {
	state := GameState{Maze: maze, Players: players}
	data, _ := json.Marshal(state)

	for c := range clients {
		c.WriteMessage(websocket.TextMessage, data)
	}
}

/* =========================
   Maze Logic
========================= */

func generateMaze(previous [][]int) [][]int {
	m := make([][]int, GridSize)
	for y := range m {
		m[y] = make([]int, GridSize)
		for x := range m[y] {
			m[y][x] = 1
		}
	}

	var carve func(x, y int)
	carve = func(x, y int) {
		dirs := [][2]int{{2,0},{-2,0},{0,2},{0,-2}}
		rand.Shuffle(len(dirs), func(i, j int) {
			dirs[i], dirs[j] = dirs[j], dirs[i]
		})

		for _, d := range dirs {
			nx, ny := x+d[0], y+d[1]
			if nx > 0 && ny > 0 && nx < GridSize-1 && ny < GridSize-1 {
				if m[ny][nx] == 1 {
					m[ny][nx] = 0
					m[y+d[1]/2][x+d[0]/2] = 0
					carve(nx, ny)
				}
			}
		}
	}

	m[1][1] = 0
	carve(1, 1)

	// Corner exits
	m[0][0] = 0
	m[0][GridSize-1] = 0
	m[GridSize-1][0] = 0
	m[GridSize-1][GridSize-1] = 0

	// Slight similarity
	if previous != nil {
		for y := 1; y < GridSize-1; y++ {
			for x := 1; x < GridSize-1; x++ {
				if rand.Float64() < 0.1 {
					m[y][x] = previous[y][x]
				}
			}
		}
	}

	return m
}

func isExit(p *Player) bool {
	gx := int(p.X) / CellSize
	gy := int(p.Y) / CellSize
	return (gx == 0 || gx == GridSize-1) && (gy == 0 || gy == GridSize-1)
}

/* =========================
   Player Logic
========================= */

func spawnPlayer(id string) *Player {
	return &Player{
		ID: id,
		X:  float64(GridSize*CellSize) / 2,
		Y:  float64(GridSize*CellSize) / 2,
	}
}

func tryMove(p *Player, dx, dy float64) {
	nx := p.X + dx
	ny := p.Y + dy

	gx := int(nx) / CellSize
	gy := int(ny) / CellSize

	if gx < 0 || gy < 0 || gx >= GridSize || gy >= GridSize {
		return
	}

	if maze[gy][gx] == 0 {
		p.X = nx
		p.Y = ny
	}
}

func randID() string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

/* =========================
   HTML + JS
========================= */

func serveHTML(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`<!doctype html>
<html>
<head>
<meta charset="utf-8">
<title>Multiplayer Maze</title>
<style>
body { margin:0; background:#111; }
canvas { display:block; margin:auto; background:#000; }
</style>
</head>
<body>
<canvas id="c"></canvas>
<script>
const ws = new WebSocket("ws://" + location.host + "/ws");
const c = document.getElementById("c");
const ctx = c.getContext("2d");
const CELL = ` + strconv.Itoa(CellSize) + `;

let maze = [];
let players = {};

ws.onmessage = e => {
	const state = JSON.parse(e.data);
	maze = state.maze;
	players = state.players;
	c.width = maze.length * CELL;
	c.height = maze.length * CELL;
	draw();
};

function draw() {
	ctx.clearRect(0,0,c.width,c.height);
	for (let y=0;y<maze.length;y++) {
		for (let x=0;x<maze.length;x++) {
			if (maze[y][x] === 1) {
				ctx.fillStyle = "#000";
				ctx.fillRect(x*CELL,y*CELL,CELL,CELL);
			}
		}
	}
	for (let id in players) {
		const p = players[id];
		ctx.fillStyle = "white";
		ctx.beginPath();
		ctx.arc(p.x,p.y,6,0,Math.PI*2);
		ctx.fill();
	}
}

const keys = {};
onkeydown = e => keys[e.key] = true;
onkeyup = e => keys[e.key] = false;

setInterval(() => {
	let dx = 0, dy = 0;
	if (keys["w"]) dy -= 2;
	if (keys["s"]) dy += 2;
	if (keys["a"]) dx -= 2;
	if (keys["d"]) dx += 2;
	ws.send(JSON.stringify({dx, dy}));
}, 16);
</script>
</body>
</html>`))
}


