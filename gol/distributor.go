package gol

import (
	"fmt"
	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
}

func calculateAliveCells(p Params, world [][]byte) []util.Cell {
	var cells []util.Cell

	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			if world[y][x] == 255 {
				cells = append(cells, util.Cell{X: x, Y: y})
			}
		}
	}
	return cells
}

// ======================================================================================================================================
func makeMatrix(height, width int) [][]uint8 {
	matrix := make([][]uint8, height)
	for i := range matrix {
		matrix[i] = make([]uint8, width)
	}
	return matrix
}

func calculateNextState(startY, endY, width int, world [][]byte) [][]uint8 {
	height := endY - startY

	newWorld := make([][]byte, height)
	for y := range newWorld {
		newWorld[y] = make([]byte, width)
	}

	for y := startY; y < endY; y++ {
		for x := 0; x < width; x++ {
			sum := (world[(y-1+height)%height][(x-1+width)%width])/255 + (world[(y-1+height)%height][(x+width)%width])/255 + (world[(y-1+height)%height][(x+1+width)%width])/255 +
				(world[(y+height)%height][(x-1+width)%width])/255 + (world[(y+height)%height][(x+1+width)%width])/255 +
				(world[(y+1+height)%height][(x-1+width)%width])/255 + (world[(y+1+height)%height][(x+width)%width])/255 + (world[(y+1+height)%height][(x+1+width)%width])/255
			if world[y][x] == 255 {
				if sum < 2 {
					newWorld[y][x] = 0
				} else if sum == 2 || sum == 3 {
					newWorld[y][x] = 255
				} else {
					newWorld[y][x] = 0
				}
			} else {
				if sum == 3 {
					newWorld[y][x] = 255
				}
			}
		}
	}
	return newWorld
}

func worker(startY, endY, width int, newWorld [][]byte, out chan<- [][]uint8) {
	imagePart := calculateNextState(startY, endY, width, newWorld)
	out <- imagePart
}

//======================================================================================================================================

func distributor(p Params, c distributorChannels) {
	c.ioCommand <- ioInput
	c.ioFilename <- fmt.Sprintf("%dx%d", p.ImageWidth, p.ImageHeight)

	height := p.ImageHeight
	width := p.ImageWidth

	// distributor divides the work between workers and interacts with other goroutines.
	// TODO: Create a 2D slice to store the world.
	world := make([][]byte, p.ImageHeight)
	for y := range world {
		world[y] = make([]byte, p.ImageWidth)
		for x := range world[y] {
			if <-c.ioInput > 0 {
				world[y][x] = 255
			} else {
				world[y][x] = 0
			}
		}
	}

	newWorld := world

	//var newPixelData [][]uint8
	threads := p.Threads

	for turn := 0; turn < p.Turns; turn++ {
		if threads == 1 {
			newWorld = calculateNextState(0, height, width, newWorld)
		} else {
			workerHeight := height / threads
			out := make([]chan [][]uint8, threads)

			for i := range out {
				out[i] = make(chan [][]uint8)
			}

			for i := 0; i < threads; i++ {
				go worker(i*workerHeight, (i+1)*workerHeight, width, newWorld, out[i])
			}

			newWorld = makeMatrix(0, 0)

			for i := 0; i < threads; i++ {
				part := <-out[i]
				newWorld = append(newWorld, part...)
			}
		}
	}
	workersWorld := newWorld

	// TODO: Execute all turns of the Game of Life.

	// TODO: Report the final state using FinalTurnCompleteEvent.

	c.events <- FinalTurnComplete{
		CompletedTurns: p.Turns,
		Alive:          calculateAliveCells(p, workersWorld),
	}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{p.Turns, Quitting}

	close(c.events)
}
