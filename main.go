package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"
	"math"

	"github.com/gorilla/websocket"
)

const (
	GridSize   = 41 // must be odd
	CellSize   = 20
	PlayerSize = 6
)


type Player struct {
	ID string  `json:"id"`
	X  float64 `json:"x"`
	Y  float64 `json:"y"`
}

type GameState struct {
	Maze    [][]int           `json:"maze"`
	Players map[string]*Player `json:"players"`
}

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	clients = make(map[*websocket.Conn]string)
	players = make(map[string]*Player)
	maze    [][]int
	mu      sync.Mutex
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

regenerate := false

// Move player once
tryMove(p, input.DX, input.DY)

// Clamp position inside grid bounds
p.X = math.Max(0, math.Min(p.X, float64(GridSize*CellSize)))
p.Y = math.Max(0, math.Min(p.Y, float64(GridSize*CellSize)))

// Now check exits
for _, pl := range players {
	if isExit(pl) {
		regenerate = true
		break
	}
}


if regenerate {
    // Generate new maze
    maze = generateMaze(maze)
    // Reset all players to center
    for _, pl := range players {
        pl.X = float64(GridSize*CellSize) / 2
        pl.Y = float64(GridSize*CellSize) / 2
    }
}

// Broadcast state after possible regeneration
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
		dirs := [][2]int{{2, 0}, {-2, 0}, {0, 2}, {0, -2}}
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

	// Reduce dead-ends by punching extra paths
// Add loops WITHOUT breaking maze structure
m[1][1] = 0
carve(1, 1)

// Add loops WITHOUT breaking maze structure
for y := 2; y < GridSize-2; y += 2 {
	for x := 2; x < GridSize-2; x += 2 {
		if rand.Float64() < 0.25 {
			dirs := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
			d := dirs[rand.Intn(len(dirs))]
			m[y+d[1]][x+d[0]] = 0
		}
	}
}




	// Edge exits
	// Guaranteed exits (bottom-left & bottom-right)
// Guaranteed 2 exits anywhere on edges
edges := [][2]int{}

// Top & bottom rows
for x := 1; x < GridSize-1; x++ {
	edges = append(edges, [2]int{x, 0})              // top
	edges = append(edges, [2]int{x, GridSize-1})    // bottom
}
// Left & right columns
for y := 1; y < GridSize-1; y++ {
	edges = append(edges, [2]int{0, y})             // left
	edges = append(edges, [2]int{GridSize-1, y})    // right
}

// Pick 2 distinct random exits
perm := rand.Perm(len(edges))
exit1 := edges[perm[0]]
exit2 := edges[perm[1]]

// Carve them
m[exit1[1]][exit1[0]] = 0
m[exit2[1]][exit2[0]] = 0

// Optional: carve one adjacent cell inward so player can move
dx1, dy1 := 0, 0
dx2, dy2 := 0, 0

if exit1[0] == 0 { dx1 = 1 }
if exit1[0] == GridSize-1 { dx1 = -1 }
if exit1[1] == 0 { dy1 = 1 }
if exit1[1] == GridSize-1 { dy1 = -1 }
m[exit1[1]+dy1][exit1[0]+dx1] = 0

if exit2[0] == 0 { dx2 = 1 }
if exit2[0] == GridSize-1 { dx2 = -1 }
if exit2[1] == 0 { dy2 = 1 }
if exit2[1] == GridSize-1 { dy2 = -1 }
m[exit2[1]+dy2][exit2[0]+dx2] = 0


m[GridSize-1][GridSize-2] = 0
m[GridSize-2][GridSize-2] = 0

	// Slight similarity
	if previous != nil {
	for y := 1; y < GridSize-1; y++ {
		for x := 1; x < GridSize-1; x++ {
			if rand.Float64() < 0.08 {
				m[y][x] = previous[y][x]
			}
		}
	}
}

	cx := GridSize / 2
cy := GridSize / 2

m[cy][cx] = 0
m[cy][cx-1] = 0
m[cy-1][cx] = 0
m[cy-1][cx-1] = 0
	
	return m
}

func isExit(p *Player) bool {
	gx := int(p.X) / CellSize
	gy := int(p.Y) / CellSize

	// Must be inside grid
	if gx < 0 || gy < 0 || gx >= GridSize || gy >= GridSize {
		return false
	}

	// Open cell on any edge = exit
	if maze[gy][gx] == 0 {
		return gx == 0 ||
			gx == GridSize-1 ||
			gy == 0 ||
			gy == GridSize-1
	}

	return false
}



/* =========================
   Player Logic
========================= */

func spawnPlayer(id string) *Player {
	cx := GridSize / 2
	cy := GridSize / 2

	// If center is a wall, search nearby
	for r := 0; r < GridSize; r++ {
		for y := cy - r; y <= cy+r; y++ {
			for x := cx - r; x <= cx+r; x++ {
				if x >= 0 && y >= 0 && x < GridSize && y < GridSize {
					if maze[y][x] == 0 {
						return &Player{
							ID: id,
							X:  float64(x*CellSize + CellSize/2),
							Y:  float64(y*CellSize + CellSize/2),
						}
					}
				}
			}
		}
	}

	// Fallback (should never hit)
	return &Player{ID: id}
}


// Check collision with maze walls
func canMove(nx, ny float64) bool {
	points := [][2]float64{
		{nx - PlayerSize, ny - PlayerSize},
		{nx + PlayerSize, ny - PlayerSize},
		{nx - PlayerSize, ny + PlayerSize},
		{nx + PlayerSize, ny + PlayerSize},
	}

	for _, pt := range points {
		gx := int(pt[0]) / CellSize
		gy := int(pt[1]) / CellSize

		if gx < 0 || gy < 0 || gx >= GridSize || gy >= GridSize {
	// Allow movement outside → exit
	return true
}

		if maze[gy][gx] == 1 {
			return false
		}
	}

	return true
}

// Try to move player with wall collision
// Try to move player with wall collision (axis-separated)
func tryMove(p *Player, dx, dy float64) {
	// Move X first
	if dx != 0 {
		nx := p.X + dx
		if canMove(nx, p.Y) {
			p.X = nx
		}
	}

	// Move Y second
	if dy != 0 {
		ny := p.Y + dy
		if canMove(p.X, ny) {
			p.Y = ny
		}
	}
}


func randID() string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
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
canvas { display:block; margin:auto; background:#111; }
</style>
</head>
<body>
<canvas id="c"></canvas>
<script>
const protocol = location.protocol === "https:" ? "wss://" : "ws://";
const ws = new WebSocket(protocol + location.host + "/ws");
const c = document.getElementById("c");
const ctx = c.getContext("2d");
const CELL = ` + strconv.Itoa(CellSize) + `;

let maze = [];
let players = {};
let canvasSized = false;

ws.onmessage = e => {
	const state = JSON.parse(e.data);
	maze = state.maze;
	players = state.players;

	if (!canvasSized) {
		c.width = maze.length * CELL;
		c.height = maze.length * CELL;
		canvasSized = true;
	}

	draw();
};

function draw() {
	ctx.clearRect(0,0,c.width,c.height);

	for (let y=0; y<maze.length; y++) {
		for (let x=0; x<maze.length; x++) {
			if (maze[y][x] === 1) {
				ctx.fillStyle = "#444";
				ctx.fillRect(x*CELL, y*CELL, CELL, CELL);
			}
		}
	}

	for (let id in players) {
		const p = players[id];
		ctx.fillStyle = "white";
		ctx.beginPath();
		ctx.arc(p.x, p.y, 6, 0, Math.PI*2);
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









