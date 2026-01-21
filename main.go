package main

import (
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

// ----- Maze Config -----
const COLS = 30
const ROWS = 20

type Cell struct {
	Walls  [4]bool // top, right, bottom, left
	Visited bool
}

var maze [ROWS][COLS]Cell

// ----- Maze Generation (DFS/Backtracking) -----
func generateMaze() {
	// initialize all walls
	for y := 0; y < ROWS; y++ {
		for x := 0; x < COLS; x++ {
			maze[y][x] = Cell{Walls: [4]bool{true, true, true, true}, Visited: false}
		}
	}

	type Pos struct{ x, y int }
	stack := []Pos{{0, 0}}
	maze[0][0].Visited = true

	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		x, y := cur.x, cur.y

		// find unvisited neighbors
		neighbors := []struct {
			nx, ny, dir int
		}{}
		dirs := []struct{ dx, dy, dir int }{
			{0, -1, 0}, {1, 0, 1}, {0, 1, 2}, {-1, 0, 3},
		}

		for _, d := range dirs {
			nx, ny := x+d.dx, y+d.dy
			if nx >= 0 && ny >= 0 && nx < COLS && ny < ROWS && !maze[ny][nx].Visited {
				neighbors = append(neighbors, struct{ nx, ny, dir int }{nx, ny, d.dir})
			}
		}

		if len(neighbors) > 0 {
			// choose random neighbor
			choice := neighbors[rand.Intn(len(neighbors))]
			nx, ny, dir := choice.nx, choice.ny, choice.dir

			// remove walls
			maze[y][x].Walls[dir] = false
			maze[ny][nx].Walls[(dir+2)%4] = false

			maze[ny][nx].Visited = true
			stack = append(stack, Pos{nx, ny})
		} else {
			stack = stack[:len(stack)-1]
		}
	}

	// open simple entrance & exit
	maze[0][0].Walls[3] = false
	maze[ROWS-1][COLS-1].Walls[1] = false
}

func main() {
	rand.Seed(time.Now().UnixNano())
	generateMaze()

	http.HandleFunc("/", serveHTML)
	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// ----- Serve HTML -----
func serveHTML(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`<!doctype html>
<html>
<head><meta charset="UTF-8"><title>Go Maze</title>
<style>
html, body { margin:0; padding:0; overflow:hidden; background:#e5e5e5; }
canvas { display:block; margin:auto; background:white; border:4px solid black; }
</style></head>
<body>
<canvas id="mazeCanvas"></canvas>
<script>
const COLS = ` + strconv.Itoa(COLS) + `;
const ROWS = ` + strconv.Itoa(ROWS) + `;
const maze = ` + mazeToJS() + `;

const canvas = document.getElementById("mazeCanvas");
const ctx = canvas.getContext("2d");
function resize() {
  canvas.width = window.innerWidth;
  canvas.height = window.innerHeight;
}
window.addEventListener("resize", resize);
resize();

let CELL = Math.min((canvas.width-20)/COLS, (canvas.height-20)/ROWS);
let OFFSET_X = (canvas.width - COLS*CELL)/2;
let OFFSET_Y = (canvas.height-ROWS*CELL)/2;

// ----- Player -----
const player = {
  x: COLS/2 + 0.5,
  y: ROWS/2 + 0.5,
  r: CELL*0.4,
  speed: 6,
  color: "#"+Math.floor(Math.random()*16777215).toString(16),
};
const keys = {};
window.addEventListener("keydown", e=>keys[e.key.toLowerCase()]=true);
window.addEventListener("keyup", e=>keys[e.key.toLowerCase()]=false);

// ----- Drawing -----
function drawMaze(){
  ctx.clearRect(0,0,canvas.width,canvas.height);
  ctx.strokeStyle="black";
  ctx.lineWidth=2;
  for(let y=0; y<ROWS; y++){
    for(let x=0; x<COLS; x++){
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
  if(cell.walls[0] && py<0.1) return true;
  if(cell.walls[1] && px>0.9) return true;
  if(cell.walls[2] && py>0.9) return true;
  if(cell.walls[3] && px<0.1) return true;
  return false;
}

// ----- Movement -----
let lastTime=performance.now();
function movePlayer(delta){
  let nx = player.x, ny=player.y;
  const s = player.speed*delta;
  if(keys["w"]||keys["arrowup"]){ let t=ny-s; if(!hitWall(nx,t)) ny=t; }
  if(keys["s"]||keys["arrowdown"]){ let t=ny+s; if(!hitWall(nx,t)) ny=t; }
  if(keys["a"]||keys["arrowleft"]){ let t=nx-s; if(!hitWall(t,ny)) nx=t; }
  if(keys["d"]||keys["arrowright"]){ let t=nx+s; if(!hitWall(t,ny)) nx=t; }
  player.x = nx; player.y = ny;
}

function loop(now){
  const delta=(now-lastTime)/1000;
  lastTime=now;
  movePlayer(delta);
  drawMaze();
  ctx.fillStyle = player.color;
  ctx.beginPath();
  ctx.arc(OFFSET_X+player.x*CELL, OFFSET_Y+player.y*CELL, player.r, 0, Math.PI*2);
  ctx.fill();
  requestAnimationFrame(loop);
}
requestAnimationFrame(loop);
</script>
</body></html>
`))
}

// ----- Serialize Maze to JS -----
func mazeToJS() string {
	s := "["
	for y := 0; y < ROWS; y++ {
		s += "["
		for x := 0; x < COLS; x++ {
			c := maze[y][x]
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
