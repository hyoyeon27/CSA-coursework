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

func calculateNextState(p Params, world [][]byte) [][]byte {
	newWorld := make([][]byte, p.ImageHeight)
	for y := range newWorld {
		newWorld[y] = make([]byte, p.ImageWidth)
	}

	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			sum := (world[(y-1+p.ImageHeight)%p.ImageHeight][(x-1+p.ImageWidth)%p.ImageWidth])/255 + (world[(y-1+p.ImageHeight)%p.ImageHeight][(x+p.ImageWidth)%p.ImageWidth])/255 + (world[(y-1+p.ImageHeight)%p.ImageHeight][(x+1+p.ImageWidth)%p.ImageWidth])/255 +
				(world[(y+p.ImageHeight)%p.ImageHeight][(x-1+p.ImageWidth)%p.ImageWidth])/255 + (world[(y+p.ImageHeight)%p.ImageHeight][(x+1+p.ImageWidth)%p.ImageWidth])/255 +
				(world[(y+1+p.ImageHeight)%p.ImageHeight][(x-1+p.ImageWidth)%p.ImageWidth])/255 + (world[(y+1+p.ImageHeight)%p.ImageHeight][(x+p.ImageWidth)%p.ImageWidth])/255 + (world[(y+1+p.ImageHeight)%p.ImageHeight][(x+1+p.ImageWidth)%p.ImageWidth])/255
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

func distributor(p Params, c distributorChannels) {
	c.ioCommand <- ioInput
	c.ioFilename <- fmt.Sprintf("%dx%d", p.ImageWidth, p.ImageHeight)

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

	turn := 0

	// TODO: Execute all turns of the Game of Life.
	for a := 0; a < p.Turns; a++ {
		newWorld = calculateNextState(p, newWorld)
		fmt.Printf("Running!")
	}

	// TODO: Report the final state using FinalTurnCompleteEvent.

	c.events <- FinalTurnComplete{
		CompletedTurns: p.Turns,
		Alive:          calculateAliveCells(p, world),
	}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)

}
