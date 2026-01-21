package main

import (
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

// ===== Maze Config =====
const (
	COLS = 40
)

var ROWS int

type Cell struct {
	X, Y    int
	Walls   [4]bool // top, right, bottom, left
	Visited bool
}

var grid [][]Cell

func main() {
	rand.Seed(time.Now().UnixNano())
	// Rows will be calculated based on canvas size in JS, but pick default
	ROWS = 25
	generateMaze()

	http.HandleFunc("/", serveHTML)
	log.Println("Running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// ===== Maze Generation (DFS / stack) =====
func generateMaze() {
	grid = make([][]Cell, ROWS)
	for y := 0; y < ROWS; y++ {
		grid[y] = make([]Cell, COLS)
		for x := 0; x < COLS; x++ {
			grid[y][x] = Cell{
				X:      x,
				Y:      y,
				Walls:  [4]bool{true, true, true, true},
				Visited: false,
			}
		}
	}

	stack := []Cell{grid[0][0]}
	current := &grid[0][0]

	DIRS := []struct{ X, Y int }{
		{0, -1}, // top
		{1, 0},  // right
		{0, 1},  // bottom
		{-1, 0}, // left
	}

	for len(stack) > 0 {
		current.Visited = true
		// find unvisited neighbors
		neighbors := []struct {
			Cell *Cell
			Dir  int
		}{}
		for i, d := range DIRS {
			nx, ny := current.X+d.X, current.Y+d.Y
			if nx >= 0 && ny >= 0 && nx < COLS && ny < ROWS {
				n := &grid[ny][nx]
				if !n.Visited {
					neighbors = append(neighbors, struct {
						Cell *Cell
						Dir  int
					}{n, i})
				}
			}
		}

		if len(neighbors) > 0 {
			choice := neighbors[rand.Intn(len(neighbors))]
			// remove walls between current and neighbor
			current.Walls[choice.Dir] = false
			choice.Cell.Walls[(choice.Dir+2)%4] = false
			stack = append(stack, *current)
			current = choice.Cell
		} else {
			current = &stack[len(stack)-1]
			stack = stack[:len(stack)-1]
		}
	}

	// Open exits (like Firebase code)
	grid[0][0].Walls[0] = false
	grid[0][0].Walls[3] = false
	grid[0][COLS-1].Walls[1] = false
	grid[ROWS-1][0].Walls[3] = false
	grid[ROWS-1][COLS-1].Walls[2] = false
}

// ===== Serve HTML =====
func serveHTML(w http.ResponseWriter, r *http.Request) {
	html := `<!doctype html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>Go Maze Multiplayer</title>
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

const COLS = ` + strconv.Itoa(COLS) + `;
const ROWS = ` + strconv.Itoa(ROWS) + `;
const PADDING = 50;
let CELL, OFFSET_X, OFFSET_Y;

function recalcGrid(){
	CELL = Math.floor((window.innerWidth-PADDING*2)/COLS);
	OFFSET_X = (window.innerWidth-COLS*CELL)/2;
	OFFSET_Y = (window.innerHeight-ROWS*CELL)/2;
	canvas.width = window.innerWidth;
	canvas.height = window.innerHeight;
}
window.addEventListener("resize", recalcGrid);
recalcGrid();

const grid = ` + mazeToJS() + `;

// ===== Player =====
const player = {x:0.5, y:0.5, r:15, color:"#"+Math.floor(Math.random()*16777215).toString(16)};
const FIXED_SPEED = 5;
const keys = {};
window.addEventListener("keydown", e => keys[e.key.toLowerCase()]=true);
window.addEventListener("keyup", e => keys[e.key.toLowerCase()]=false);

// ===== Maze Drawing =====
function drawMaze(){
	ctx.clearRect(0,0,canvas.width,canvas.height);
	ctx.strokeStyle="black";
	ctx.lineWidth=2;
	for(let y=0;y<ROWS;y++){
		for(let x=0;x<COLS;x++){
			const c=grid[y][x];
			const px=OFFSET_X+x*CELL;
			const py=OFFSET_Y+y*CELL;
			if(c.walls[0]) ctx.strokeRect(px,py,CELL,1);
			if(c.walls[1]) ctx.strokeRect(px+CELL,py,1,CELL);
			if(c.walls[2]) ctx.strokeRect(px,py+CELL,CELL,1);
			if(c.walls[3]) ctx.strokeRect(px,py,1,CELL);
		}
	}
}

// ===== Collision =====
function hitWall(nx, ny){
	const cx=Math.floor(nx);
	const cy=Math.floor(ny);
	if(cx<0||cy<0||cx>=COLS||cy>=ROWS) return true;
	const cell=grid[cy][cx];
	const px=nx-cx, py=ny-cy;
	if(cell.walls[0] && py<0.05) return true;
	if(cell.walls[1] && px>0.95) return true;
	if(cell.walls[2] && py>0.95) return true;
	if(cell.walls[3] && px<0.05) return true;
	return false;
}

// ===== Player Movement =====
let lastTime = performance.now();
function movePlayer(delta){
	let nx = player.x, ny = player.y;
	const s = FIXED_SPEED * delta;
	if(keys["a"]||keys["arrowleft"]){ let t=nx-s; if(!hitWall(t,ny)) nx=t; }
	if(keys["d"]||keys["arrowright"]){ let t=nx+s; if(!hitWall(t,ny)) nx=t; }
	if(keys["w"]||keys["arrowup"]){ let t=ny-s; if(!hitWall(nx,t)) ny=t; }
	if(keys["s"]||keys["arrowdown"]){ let t=ny+s; if(!hitWall(nx,t)) ny=t; }
	player.x = nx; player.y = ny;
}

// ===== Draw Loop =====
function loop(now){
	const delta=(now-lastTime)/1000;
	lastTime=now;
	movePlayer(delta);
	drawMaze();
	ctx.fillStyle=player.color;
	ctx.beginPath();
	ctx.arc(OFFSET_X+player.x*CELL, OFFSET_Y+player.y*CELL, player.r, 0, Math.PI*2);
	ctx.fill();
	requestAnimationFrame(loop);
}
requestAnimationFrame(loop);
</script>
</body>
</html>
`
	w.Write([]byte(html))
}

// ===== Convert Go maze to JS =====
func mazeToJS() string {
	s := "["
	for y := 0; y < ROWS; y++ {
		s += "["
		for x := 0; x < COLS; x++ {
			c := grid[y][x]
			s += "{walls:[" + boolToJS(c.Walls[0]) + "," + boolToJS(c.Walls[1]) + "," + boolToJS(c.Walls[2]) + "," + boolToJS(c.Walls[3]) + "]}"
			if x < COLS-1 { s += "," }
		}
		s += "]"
		if y < ROWS-1 { s += "," }
	}
	s += "]"
	return s
}

func boolToJS(b bool) string {
	if b { return "true" }
	return "false"
}
