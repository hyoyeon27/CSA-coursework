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

func calculateNextState(p Params, startY, endY, width int, world [][]byte) [][]uint8 {
	//height := endY - startY
	height := p.ImageHeight

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

func worker(p Params, startY, endY, width int, newWorld [][]byte, out chan<- [][]uint8) {
	imagePart := calculateNextState(p, startY, endY, width, newWorld)
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

	threads := p.Threads
	var workersWorld [][]uint8

	if p.Turns == 0 {
		workersWorld = newWorld
	} else {
		for turn := 0; turn < p.Turns; turn++ {
			if threads == 1 {
				newWorld = calculateNextState(p, 0, height, width, newWorld)
				workersWorld = newWorld
			} else {
				workerHeight := height / threads
				out := make([]chan [][]uint8, threads)
				for i := range out {
					out[i] = make(chan [][]uint8)
				}

				if p.ImageHeight%p.Threads == 0 { // when the thread can be divided
					for i := 0; i < threads; i++ {
						go worker(p, i*workerHeight, (i+1)*workerHeight, p.ImageWidth, newWorld, out[i])
					}
				} else { // when the thread cannot be divided by the thread(has remainders)
					for i := 0; i < p.Threads; i++ {
						if i == (p.Threads - 1) { // if it is the last thread
							go worker(p, i*workerHeight, workerHeight, p.ImageWidth, newWorld, out[i])
						} else { //else
							go worker(p, i*workerHeight, (i+1)*workerHeight, p.ImageWidth, newWorld, out[i])
						}
					}
				}

				workersWorld = makeMatrix(0, 0)

				for i := 0; i < threads; i++ {
					fmt.Println("I am appending!!")
					part := <-out[i]
					workersWorld = append(workersWorld, part...)
				}
			}
		}
	}

	finalWorld := workersWorld

	// TODO: Execute all turns of the Game of Life.

	// TODO: Report the final state using FinalTurnCompleteEvent.

	c.events <- FinalTurnComplete{
		CompletedTurns: p.Turns,
		Alive:          calculateAliveCells(p, finalWorld),
	}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{p.Turns, Quitting}

	close(c.events)
}

//func distributor(p Params, c distributorChannels) {
//	c.ioCommand <- ioInput
//	c.ioFilename <- fmt.Sprintf("%dx%d", p.ImageWidth, p.ImageHeight)
//
//	height := p.ImageHeight
//	width := p.ImageWidth
//
//	// distributor divides the work between workers and interacts with other goroutines.
//	// TODO: Create a 2D slice to store the world.
//	world := make([][]byte, p.ImageHeight)
//	for y := range world {
//		world[y] = make([]byte, p.ImageWidth)
//		for x := range world[y] {
//			if <-c.ioInput > 0 {
//				world[y][x] = 255
//			} else {
//				world[y][x] = 0
//			}
//		}
//	}
//
//	newWorld := world
//
//	threads := p.Threads
//	var workersWorld [][]uint8
//
//	if p.Turns == 0 {
//		workersWorld = newWorld
//	} else {
//		for turn := 0; turn < p.Turns; turn++ {
//			if threads == 1 {
//				newWorld = calculateNextState(p, 0, height, width, newWorld)
//				workersWorld = newWorld
//			} else {
//				workerHeight := height / threads
//				out := make([]chan [][]uint8, threads)
//
//				for i := range out {
//					out[i] = make(chan [][]uint8)
//				}
//
//				for i := 0; i < threads; i++ {
//					go worker(p, i*workerHeight, (i+1)*workerHeight, width, newWorld, out[i])
//				}
//
//				workersWorld = makeMatrix(0, 0)
//
//				for i := 0; i < threads; i++ {
//					part := <-out[i]
//					workersWorld = append(workersWorld, part...)
//				}
//			}
//		}
//	}
//	finalWorld := workersWorld
//
//	// TODO: Execute all turns of the Game of Life.
//
//	// TODO: Report the final state using FinalTurnCompleteEvent.
//
//	c.events <- FinalTurnComplete{
//		CompletedTurns: p.Turns,
//		Alive:          calculateAliveCells(p, finalWorld),
//	}
//
//	// Make sure that the Io has finished any output before exiting.
//	c.ioCommand <- ioCheckIdle
//	<-c.ioIdle
//
//	c.events <- StateChange{p.Turns, Quitting}
//
//	close(c.events)
//}
