Multiplayer Maze (Go + WebSockets)

A real-time multiplayer maze game written in Go, featuring procedural maze generation, shared exits, and synchronized player movement in the browser.

Features

Procedurally generated maze (odd-sized grid)

Real-time multiplayer via WebSockets

Wall collision & smooth movement

Random edge exits that regenerate the maze

Shared world state (maze + players)

Single-file server with embedded HTML client

Tech Stack

Go

Gorilla WebSocket

HTML5 Canvas

Run Locally
go run main.go


Open in your browser:

http://localhost:8080



Controls

WASD – Move

Game Rules

Reaching an open edge tile regenerates the maze

All players reset to the center after regeneration
