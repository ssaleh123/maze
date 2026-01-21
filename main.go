package main

import (
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const size = 21

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var (
	mu      sync.Mutex
	maze    = newMaze()
	players = map[*websocket.Conn][2]int{}
)

func main() {
	rand.Seed(time.Now().UnixNano())

	http.HandleFunc("/", serveHTML)
	http.HandleFunc("/ws", ws)

	log.Println("running on :8080")
	http.ListenAndServe(":8080", nil)
}

func serveHTML(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(html))
}

func ws(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer c.Close()

	mu.Lock()
	players[c] = [2]int{size / 2, size / 2}
	mu.Unlock()

	send()

	for {
		var dir string
		if c.ReadJSON(&dir) != nil {
			break
		}

		move(c, dir)

		if exit(players[c]) {
			reset()
		}

		send()
	}

	mu.Lock()
	delete(players, c)
	mu.Unlock()
}

func newMaze() [][]int {
	m := make([][]int, size)
	for y := range m {
		m[y] = make([]int, size)
		for x := range m[y] {
			if rand.Intn(100) < 30 {
				m[y][x] = 1
			}
		}
	}

	c := size / 2
	m[c][c] = 0

	m[0][0] = 0
	m[0][size-1] = 0
	m[size-1][0] = 0
	m[size-1][size-1] = 0

	return m
}

func move(c *websocket.Conn, d string) {
	mu.Lock()
	defer mu.Unlock()

	p := players[c]
	x, y := p[0], p[1]

	switch d {
	case "up":
		y--
	case "down":
		y++
	case "left":
		x--
	case "right":
		x++
	}

	if x < 0 || y < 0 || x >= size || y >= size {
		return
	}

	if maze[y][x] == 0 {
		players[c] = [2]int{x, y}
	}
}

func exit(p [2]int) bool {
	x, y := p[0], p[1]
	return (x == 0 && y == 0) ||
		(x == 0 && y == size-1) ||
		(x == size-1 && y == 0) ||
		(x == size-1 && y == size-1)
}

func reset() {
	mu.Lock()
	defer mu.Unlock()

	maze = newMaze()
	for c := range players {
		players[c] = [2]int{size / 2, size / 2}
	}
}

func send() {
	mu.Lock()
	defer mu.Unlock()

	state := map[string]any{
		"maze":    maze,
		"players": players,
	}

	for c := range players {
		c.WriteJSON(state)
	}
}

const html = `
<!DOCTYPE html>
<html>
<body style="margin:0;background:#fff">
<canvas id="c"></canvas>

<script>
const ws = new WebSocket("ws://" + location.host + "/ws")
const c = document.getElementById("c")
const ctx = c.getContext("2d")

c.width = 420
c.height = 420
const tile = 20

let state = null

ws.onmessage = e => {
	state = JSON.parse(e.data)
	draw()
}

document.addEventListener("keydown", e => {
	const m = {
		ArrowUp: "up", ArrowDown: "down",
		ArrowLeft: "left", ArrowRight: "right",
		w: "up", s: "down", a: "left", d: "right"
	}[e.key]
	if (m) ws.send(JSON.stringify(m))
})

function draw() {
	if (!state) return

	ctx.clearRect(0,0,c.width,c.height)

	for (let y=0;y<state.maze.length;y++) {
		for (let x=0;x<state.maze[y].length;x++) {
			if (state.maze[y][x]) {
				ctx.fillStyle = "#0a1a3a"
				ctx.fillRect(x*tile,y*tile,tile,tile)
			}
		}
	}

	ctx.fillStyle = "red"
	for (let k in state.players) {
		const p = state.players[k]
		ctx.fillRect(p[0]*tile,p[1]*tile,tile,tile)
	}
}
</script>
</body>
</html>
`
