package main

import (
	"container/list"
	"fmt"
	"image/color"
	"log"
	"math"
	"sync"

	ebiten "github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	TileSize      = 16
	ChunkSize     = 32
	Scale         = 2
	ScreenWidth   = 1024
	ScreenHeight  = 768
	ChunkPixel    = ChunkSize * TileSize
	VisibleChunks = 3
	ChunkWorkers  = 4
	MaxChunks     = 50
)

// Chunk implements a chunk of the world.
type Chunk struct {
	Image *ebiten.Image
	Ready bool
}

// World represents game state.
type World struct {
	Chunks     map[[2]int]*Chunk
	cacheQueue *list.List
	mu         sync.RWMutex
	genQueue   chan [2]int
}

// Game implements Game struct.
type Game struct {
	World   *World
	PlayerX float64
	PlayerY float64
}

// NewWorld creates a new world.
func NewWorld() *World {
	w := &World{
		Chunks:     make(map[[2]int]*Chunk),
		cacheQueue: list.New(),
		genQueue:   make(chan [2]int, 100),
	}

	for i := 0; i < ChunkWorkers; i++ {
		go w.chunkWorker()
	}

	return w
}

// chunkWorker is a worker that generates chunks.
func (w *World) chunkWorker() {
	for coords := range w.genQueue {
		cx, cy := coords[0], coords[1]
		chunk := GenerateChunk(cx, cy)

		w.mu.Lock()
		w.Chunks[[2]int{cx, cy}] = chunk
		w.cacheQueue.PushBack([2]int{cx, cy}) // Добавляем в LRU-кеш
		w.evictOldChunks()
		w.mu.Unlock()
	}
}

// GenerateChunk generates a chunk.
func GenerateChunk(cx, cy int) *Chunk {
	img := ebiten.NewImage(ChunkPixel, ChunkPixel)
	baseColor := color.RGBA{R: uint8(cx * 30 % 255), G: uint8(cy * 50 % 255), B: 100, A: 255}

	// Real-time generation of a palette in a chunk.
	// Then we will transfer the map tiles here.
	for x := 0; x < ChunkSize; x++ {
		for y := 0; y < ChunkSize; y++ {
			tileX, tileY := x*TileSize, y*TileSize
			tileColor := color.RGBA{
				R: baseColor.R + uint8(x*5%50),
				G: baseColor.G + uint8(y*5%50),
				B: baseColor.B,
				A: 255,
			}
			for dx := 0; dx < TileSize; dx++ {
				for dy := 0; dy < TileSize; dy++ {
					img.Set(tileX+dx, tileY+dy, tileColor)
				}
			}
		}
	}

	return &Chunk{Image: img, Ready: true}
}

// GetChunk gets a chunk by coordinates.
func (w *World) GetChunk(cx, cy int) *Chunk {
	key := [2]int{cx, cy}

	w.mu.RLock()
	chunk, exists := w.Chunks[key]
	w.mu.RUnlock()

	if exists {
		w.moveToBack(key)

		return chunk
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	w.Chunks[key] = &Chunk{Image: ebiten.NewImage(ChunkPixel, ChunkPixel)}
	w.genQueue <- key
	w.cacheQueue.PushBack(key)
	w.evictOldChunks()

	return w.Chunks[key]
}

// evictOldChunks clears old chunks.
func (w *World) evictOldChunks() {
	for w.cacheQueue.Len() > MaxChunks {
		oldest := w.cacheQueue.Front()
		if oldest != nil {
			key := oldest.Value.([2]int)
			delete(w.Chunks, key)
			w.cacheQueue.Remove(oldest)
		}
	}
}

// moveToBack moves a chunk to the back of the cache.
func (w *World) moveToBack(key [2]int) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for e := w.cacheQueue.Front(); e != nil; e = e.Next() {
		if e.Value == key {
			w.cacheQueue.MoveToBack(e)

			return
		}
	}
}

// Update updates the game state.
func (g *Game) Update() error {
	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		g.PlayerX -= 4
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		g.PlayerX += 4
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		g.PlayerY -= 4
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		g.PlayerY += 4
	}

	return nil
}

// Draw draws the game screen.
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 20, G: 20, B: 40, A: 255})

	playerChunkX := int(math.Floor(g.PlayerX / float64(ChunkPixel)))
	playerChunkY := int(math.Floor(g.PlayerY / float64(ChunkPixel)))

	for dx := -VisibleChunks / 2; dx <= VisibleChunks/2; dx++ {
		for dy := -VisibleChunks / 2; dy <= VisibleChunks/2; dy++ {
			cx := playerChunkX + dx
			cy := playerChunkY + dy

			chunk := g.World.GetChunk(cx, cy)

			screenX := float64(cx*ChunkPixel) - g.PlayerX + ScreenWidth/2
			screenY := float64(cy*ChunkPixel) - g.PlayerY + ScreenHeight/2

			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(Scale, Scale)
			op.GeoM.Translate(screenX, screenY)

			screen.DrawImage(chunk.Image, op)
		}
	}

	msg := fmt.Sprintf("Player: (%.1f, %.1f)\n", g.PlayerX, g.PlayerY)
	msg += fmt.Sprintf("TPS: %.1f, FPS: %.1f\n", ebiten.ActualTPS(), ebiten.ActualFPS())
	ebitenutil.DebugPrint(screen, msg)
}

// Layout takes the outside size (e.g., the window size) and returns the (logical) screen size.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenWidth, ScreenHeight
}

func main() {
	ebiten.SetWindowSize(ScreenWidth, ScreenHeight)
	ebiten.SetWindowTitle("Chunk Cache")

	game := &Game{
		World:   NewWorld(),
		PlayerX: 512,
		PlayerY: 512,
	}

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
