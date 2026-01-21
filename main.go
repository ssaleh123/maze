package main

import (
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

const mazeSize = 21 // must be odd for proper maze

type Cell struct {
	Top, Right, Bottom, Left bool
	Visited                  bool
}

var maze [mazeSize][mazeSize]Cell

func main() {
	rand.Seed(time.Now().UnixNano())
	generateMaze()
	http.HandleFunc("/", serveHTML)
	log.Println("running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// ===== Maze Generation using DFS / Recursive Backtracker =====
func generateMaze() {
	// initialize all walls
	for y := 0; y < mazeSize; y++ {
		for x := 0; x < mazeSize; x++ {
			maze[y][x] = Cell{Top: true, Right: true, Bottom: true, Left: true}
		}
	}
	dfs(1, 1)
}

func dfs(x, y int) {
	maze[y][x].Visited = true
	dirs := []struct{ dx, dy int; wall string }{
		{0, -1, "Top"}, {1, 0, "Right"}, {0, 1, "Bottom"}, {-1, 0, "Left"},
	}
	rand.Shuffle(len(dirs), func(i, j int) { dirs[i], dirs[j] = dirs[j], dirs[i] })

	for _, d := range dirs {
		nx, ny := x+d.dx*2, y+d.dy*2
		if nx > 0 && ny > 0 && nx < mazeSize-1 && ny < mazeSize-1 && !maze[ny][nx].Visited {
			switch d.wall {
			case "Top":
				maze[y-1][x].Bottom = false
				maze[y][x].Top = false
			case "Right":
				maze[y][x+1].Left = false
				maze[y][x].Right = false
			case "Bottom":
				maze[y+1][x].Top = false
				maze[y][x].Bottom = false
			case "Left":
				maze[y][x-1].Right = false
				maze[y][x].Left = false
			}
			dfs(nx, ny)
		}
	}

	// optionally open entrance and exit
	maze[1][0].Left = false
	maze[mazeSize-2][mazeSize-1].Right = false
}

// ===== Serve HTML / JS =====
func serveHTML(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<title>Maze Game</title>
<style>
html, body { margin:0; padding:0; overflow:hidden; background:#e5e5e5; }
canvas { display:block; margin:auto; background:white; border:4px solid black; }
</style>
</head>
<body>
<canvas id="mazeCanvas"></canvas>
<script>
const canvas = document.getElementById("mazeCanvas");
const ctx = canvas.getContext("2d");
const COLS = ` + strconv.Itoa(mazeSize) + `;
const ROWS = ` + strconv.Itoa(mazeSize) + `;
const CELL = 30;
canvas.width = COLS*CELL;
canvas.height = ROWS*CELL;

// ===== Maze Data =====
const maze = ` + mazeToJS() + `;

// ===== Player =====
const player = {x:0.5, y:1.5, r:10, color:"#"+Math.floor(Math.random()*16777215).toString(16)};
const keys = {};

// ===== Input =====
window.addEventListener("keydown", e => keys[e.key.toLowerCase()] = true);
window.addEventListener("keyup", e => keys[e.key.toLowerCase()] = false);

// ===== Collision =====
function hitWall(nx, ny){
	const cx = Math.floor(nx);
	const cy = Math.floor(ny);
	if(cx<0||cy<0||cx>=COLS||cy>=ROWS) return true;
	const cell = maze[cy][cx];
	const px = nx-cx;
	const py = ny-cy;
	if(cell.Top && py<0.05) return true;
	if(cell.Right && px>0.95) return true;
	if(cell.Bottom && py>0.95) return true;
	if(cell.Left && px<0.05) return true;
	return false;
}

// ===== Draw Maze =====
function drawMaze(){
	ctx.clearRect(0,0,canvas.width,canvas.height);
	ctx.strokeStyle="black";
	ctx.lineWidth=2;
	for(let y=0;y<ROWS;y++){
		for(let x=0;x<COLS;x++){
			const c = maze[y][x];
			const px = x*CELL;
			const py = y*CELL;
			if(c.Top){ ctx.beginPath(); ctx.moveTo(px,py); ctx.lineTo(px+CELL,py); ctx.stroke(); }
			if(c.Right){ ctx.beginPath(); ctx.moveTo(px+CELL,py); ctx.lineTo(px+CELL,py+CELL); ctx.stroke(); }
			if(c.Bottom){ ctx.beginPath(); ctx.moveTo(px,py+CELL); ctx.lineTo(px+CELL,py+CELL); ctx.stroke(); }
			if(c.Left){ ctx.beginPath(); ctx.moveTo(px,py); ctx.lineTo(px,py+CELL); ctx.stroke(); }
		}
	}
}

// ===== Move Player =====
function movePlayer(delta){
	const speed = 5*delta;
	let nx = player.x;
	let ny = player.y;
	if(keys["w"]||keys["arrowup"]){ let t=ny-speed; if(!hitWall(nx,t)) ny=t; }
	if(keys["s"]||keys["arrowdown"]){ let t=ny+speed; if(!hitWall(nx,t)) ny=t; }
	if(keys["a"]||keys["arrowleft"]){ let t=nx-speed; if(!hitWall(t,ny)) nx=t; }
	if(keys["d"]||keys["arrowright"]){ let t=nx+speed; if(!hitWall(t,ny)) nx=t; }
	player.x = nx;
	player.y = ny;
}

// ===== Draw Loop =====
let lastTime = performance.now();
function loop(now){
	const delta = (now - lastTime)/1000;
	lastTime = now;
	movePlayer(delta);
	drawMaze();
	ctx.fillStyle = player.color;
	ctx.beginPath();
	ctx.arc(player.x*CELL, player.y*CELL, player.r, 0, Math.PI*2);
	ctx.fill();
	requestAnimationFrame(loop);
}
requestAnimationFrame(loop);
</script>
</body>
</html>
	`))
}

func mazeToJS() string {
	s := "["
	for y := 0; y < mazeSize; y++ {
		s += "["
		for x := 0; x < mazeSize; x++ {
			c := maze[y][x]
			s += "{Top:" + boolToJS(c.Top) + ",Right:" + boolToJS(c.Right) + ",Bottom:" + boolToJS(c.Bottom) + ",Left:" + boolToJS(c.Left) + "}"
			if x < mazeSize-1 { s += "," }
		}
		s += "]"
		if y < mazeSize-1 { s += "," }
	}
	s += "]"
	return s
}

func boolToJS(b bool) string {
	if b { return "true" }
	return "false"
}
