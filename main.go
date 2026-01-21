package main

import (
	"log"
	"math/rand"
	"net/http"
	"time"
)

const size = 21 // must be odd for maze paths

type Cell struct {
	Top, Right, Bottom, Left bool
	Visited                  bool
}

var maze [size][size]Cell

func main() {
	rand.Seed(time.Now().UnixNano())
	generateMaze()
	http.HandleFunc("/", serveHTML)
	log.Println("running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func generateMaze() {
	// Initialize all walls
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			maze[y][x] = Cell{Top: true, Right: true, Bottom: true, Left: true}
		}
	}

	dfs(1, 1)
}

func dfs(x, y int) {
	maze[y][x].Visited = true
	dirs := []struct{ dx, dy int; wall string }{
		{0, -1, "Top"},
		{1, 0, "Right"},
		{0, 1, "Bottom"},
		{-1, 0, "Left"},
	}
	rand.Shuffle(len(dirs), func(i, j int) { dirs[i], dirs[j] = dirs[j], dirs[i] })

	for _, d := range dirs {
		nx, ny := x+d.dx*2, y+d.dy*2
		if nx > 0 && ny > 0 && nx < size-1 && ny < size-1 && !maze[ny][nx].Visited {
			// Remove wall between current and next
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
}

func serveHTML(w http.ResponseWriter, r *http.Request) {
	// Convert maze to JS
	w.Write([]byte(`
<!DOCTYPE html>
<html>
<body style="margin:0;background:#fff">
<canvas id="c"></canvas>
<script>
const c = document.getElementById("c")
const ctx = c.getContext("2d")
const size = ` + string(size) + `
const tile = 20
c.width = size*tile + tile
c.height = size*tile + tile
const maze = ` + mazeToJS() + `

ctx.strokeStyle = "#0a1a3a"
ctx.lineWidth = 2

for (let y=0;y<size;y++) {
	for (let x=0;x<size;x++) {
		const cell = maze[y][x]
		const px = x*tile + tile/2
		const py = y*tile + tile/2
		if (cell.Top) ctx.beginPath(); ctx.moveTo(px,py); ctx.lineTo(px+tile,py); ctx.stroke()
		if (cell.Right) ctx.beginPath(); ctx.moveTo(px+tile,py); ctx.lineTo(px+tile,py+tile); ctx.stroke()
		if (cell.Bottom) ctx.beginPath(); ctx.moveTo(px,py+tile); ctx.lineTo(px+tile,py+tile); ctx.stroke()
		if (cell.Left) ctx.beginPath(); ctx.moveTo(px,py); ctx.lineTo(px,py+tile); ctx.stroke()
	}
}
</script>
</body>
</html>
	`))
}

// Convert Go maze to JS array
func mazeToJS() string {
	s := "["
	for y := 0; y < size; y++ {
		s += "["
		for x := 0; x < size; x++ {
			c := maze[y][x]
			s += "{"
			s += "Top:" + boolToJS(c.Top) + ","
			s += "Right:" + boolToJS(c.Right) + ","
			s += "Bottom:" + boolToJS(c.Bottom) + ","
			s += "Left:" + boolToJS(c.Left)
			s += "}"
			if x < size-1 {
				s += ","
			}
		}
		s += "]"
		if y < size-1 {
			s += ","
		}
	}
	s += "]"
	return s
}

func boolToJS(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
