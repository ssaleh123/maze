package main

import (
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ----- Maze Config -----
const COLS = 30
const ROWS = 20

type Cell struct {
	Walls  [4]bool // top, right, bottom, left
	Visited bool
}

var maze [ROWS][COLS]Cell
var mazeLock sync.Mutex

// ----- Player -----
type Player struct {
	ID    string  `json:"id"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Color string  `json:"color"`
}

var players = make(map[string]*Player)
var playersLock sync.Mutex

// ----- WebSocket -----
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// ----- Maze Generation -----
func generateMaze() {
	mazeLock.Lock()
	defer mazeLock.Unlock()
	for y := 0; y < ROWS; y++ {
		for x := 0; x < COLS; x++ {
			maze[y][x] = Cell{Walls: [4]bool{true, true, true, true}, Visited: false}
		}
	}

	type Pos struct{ x, y int }
	stack := []Pos{{0, 0}}
	maze[0][0].Visited = true

	dirs := []struct{ dx, dy, dir int }{{0, -1, 0}, {1, 0, 1}, {0, 1, 2}, {-1, 0, 3}}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		x, y := cur.x, cur.y
		var neighbors []struct{ nx, ny, dir int }

		for _, d := range dirs {
			nx, ny := x+d.dx, y+d.dy
			if nx >= 0 && ny >= 0 && nx < COLS && ny < ROWS && !maze[ny][nx].Visited {
				neighbors = append(neighbors, struct{ nx, ny, dir int }{nx, ny, d.dir})
			}
		}

		if len(neighbors) > 0 {
			choice := neighbors[rand.Intn(len(neighbors))]
			nx, ny, dir := choice.nx, choice.ny, choice.dir
			maze[y][x].Walls[dir] = false
			maze[ny][nx].Walls[(dir+2)%4] = false
			maze[ny][nx].Visited = true
			stack = append(stack, Pos{nx, ny})
		} else {
			stack = stack[:len(stack)-1]
		}
	}

	// ensure openings top-left and bottom-right
	maze[0][0].Walls[3] = false
	maze[ROWS-1][COLS-1].Walls[1] = false
}

// ----- WebSocket Handler -----
func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, _ := upgrader.Upgrade(w, r, nil)
	defer conn.Close()

	// Register player
	playerID := strconv.Itoa(rand.Int())
	player := &Player{
		ID:    playerID,
		X:     float64(COLS)/2 + 0.5,
		Y:     float64(ROWS)/2 + 0.5,
		Color: randomColor(),
	}

	playersLock.Lock()
	players[playerID] = player
	playersLock.Unlock()

	// Send initial state
	sendState(conn)

	for {
		var msg struct {
			X float64 `json:"x"`
			Y float64 `json:"y"`
		}
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}

		playersLock.Lock()
		player.X = msg.X
		player.Y = msg.Y
		playersLock.Unlock()

		// Check exit
		if (player.X < 1 && player.Y < 1) || (player.X > COLS-1 && player.Y > ROWS-1) {
			generateMaze()
			resetPlayers()
		}

		sendState(conn)
	}

	// Remove player on disconnect
	playersLock.Lock()
	delete(players, playerID)
	playersLock.Unlock()
	sendState(conn)
}

func sendState(conn *websocket.Conn) {
	mazeLock.Lock()
	defer mazeLock.Unlock()
	playersLock.Lock()
	defer playersLock.Unlock()
	conn.WriteJSON(struct {
		Maze    [ROWS][COLS]Cell `json:"maze"`
		Players []*Player        `json:"players"`
	}{maze, getPlayers()})
}

func getPlayers() []*Player {
	ps := []*Player{}
	for _, p := range players {
		ps = append(ps, p)
	}
	return ps
}

func resetPlayers() {
	playersLock.Lock()
	defer playersLock.Unlock()
	for _, p := range players {
		p.X = float64(COLS)/2 + 0.5
		p.Y = float64(ROWS)/2 + 0.5
	}
}

// ----- Random Color -----
func randomColor() string {
	return "#" + strconv.FormatInt(rand.Int63n(0xFFFFFF), 16)
}

// ----- Serve HTML -----
func serveHTML(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`<!doctype html>
<html>
<head>
<meta charset="UTF-8"><title>Go Maze Multiplayer</title>
<style>
html, body { margin:0; padding:0; overflow:hidden; background:#e5e5e5; }
canvas { display:block; margin:auto; background:white; border:4px solid black; }
</style>
</head>
<body>
<canvas id="mazeCanvas"></canvas>
<script>
let ws = new WebSocket("ws://"+location.host+"/ws");
let maze = [];
let players = [];
const COLS = ` + strconv.Itoa(COLS) + `;
const ROWS = ` + strconv.Itoa(ROWS) + `;

const canvas = document.getElementById("mazeCanvas");
const ctx = canvas.getContext("2d");
function resize() { canvas.width = window.innerWidth; canvas.height = window.innerHeight; }
window.addEventListener("resize", resize); resize();

let CELL = Math.min((canvas.width-20)/COLS, (canvas.height-20)/ROWS);
let OFFSET_X = (canvas.width - COLS*CELL)/2;
let OFFSET_Y = (canvas.height - ROWS*CELL)/2;

let playerID = null;
const keys = {};
window.addEventListener("keydown", e=>keys[e.key.toLowerCase()]=true);
window.addEventListener("keyup", e=>keys[e.key.toLowerCase()]=false);

ws.onmessage = (event)=>{
	const msg = JSON.parse(event.data);
	if(msg.maze) maze = msg.maze;
	if(msg.players) players = msg.players;
	if(!playerID && players.length>0) playerID = players[players.length-1].id;
}

function drawMaze(){
	ctx.clearRect(0,0,canvas.width,canvas.height);
	ctx.strokeStyle="black"; ctx.lineWidth=2;
	for(let y=0;y<ROWS;y++){
		for(let x=0;x<COLS;x++){
			const cell = maze[y][x];
			const px = OFFSET_X + x*CELL;
			const py = OFFSET_Y + y*CELL;
			if(cell.walls[0]) ctx.strokeRect(px,py,CELL,1);
			if(cell.walls[1]) ctx.strokeRect(px+CELL,py,1,CELL);
			if(cell.walls[2]) ctx.strokeRect(px,py+CELL,CELL,1);
			if(cell.walls[3]) ctx.strokeRect(px,py,1,CELL);
		}
	}
}

function hitWall(nx, ny){
	const cx=Math.floor(nx), cy=Math.floor(ny);
	if(cx<0||cy<0||cx>=COLS||cy>=ROWS) return true;
	const cell=maze[cy][cx];
	const px=nx-cx, py=ny-cy;
	if(cell.walls[0]&&py<0.1) return true;
	if(cell.walls[1]&&px>0.9) return true;
	if(cell.walls[2]&&py>0.9) return true;
	if(cell.walls[3]&&px<0.1) return true;
	return false;
}

let lastTime = performance.now();
function loop(now){
	const delta = (now-lastTime)/1000;
	lastTime=now;

	// move local player
	let p = players.find(x=>x.id===playerID);
	if(p){
		let nx=p.x, ny=p.y, s=delta*6;
		if(keys["w"]||keys["arrowup"]){ let t=ny-s; if(!hitWall(nx,t)) ny=t; }
		if(keys["s"]||keys["arrowdown"]){ let t=ny+s; if(!hitWall(nx,t)) ny=t; }
		if(keys["a"]||keys["arrowleft"]){ let t=nx-s; if(!hitWall(t,ny)) nx=t; }
		if(keys["d"]||keys["arrowright"]){ let t=nx+s; if(!hitWall(t,ny)) nx=t; }
		p.x=nx; p.y=ny;
		ws.send(JSON.stringify({x:nx,y:ny}));
	}

	drawMaze();
	for(let pl of players){
		ctx.fillStyle=pl.color;
		ctx.beginPath();
		ctx.arc(OFFSET_X+pl.x*CELL, OFFSET_Y+pl.y*CELL, CELL*0.4, 0, Math.PI*2);
		ctx.fill();
	}

	requestAnimationFrame(loop);
}
requestAnimationFrame(loop);
</script>
</body>
</html>`))
}

func main() {
	rand.Seed(time.Now().UnixNano())
	generateMaze()

	http.HandleFunc("/", serveHTML)
	http.HandleFunc("/ws", wsHandler)

	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
