package main

import (
	"log"
	"math/rand"
	"net/http"
	"time"
)

const size = 21

func main() {
	rand.Seed(time.Now().UnixNano())

	http.HandleFunc("/", serveHTML)

	log.Println("running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
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

	// make sure start/end corners are open
	m[0][0] = 0
	m[0][size-1] = 0
	m[size-1][0] = 0
	m[size-1][size-1] = 0

	return m
}

func serveHTML(w http.ResponseWriter, r *http.Request) {
	maze := newMaze()

	w.Write([]byte(`
<!DOCTYPE html>
<html>
<body style="margin:0;background:#fff">
<canvas id="c"></canvas>
<script>
const c = document.getElementById("c")
const ctx = c.getContext("2d")

const maze = ` + formatMazeJS(maze) + `
const tile = 20
const size = maze.length
c.width = size * tile
c.height = size * tile

for (let y=0;y<size;y++) {
	for (let x=0;x<size;x++) {
		if (maze[y][x]) {
			ctx.fillStyle = "#0a1a3a"
			ctx.fillRect(x*tile,y*tile,tile,tile)
		}
	}
}
</script>
</body>
</html>
	`))
}

// helper to format Go 2D slice as JS array
func formatMazeJS(m [][]int) string {
	s := "["
	for y, row := range m {
		s += "["
		for x, val := range row {
			s += string('0' + val)
			if x < len(row)-1 {
				s += ","
			}
		}
		s += "]"
		if y < len(m)-1 {
			s += ","
		}
	}
	s += "]"
	return s
}
