package gol

import (
	"fmt"
	"sync"
	"time"
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

func makeMatrix(height, width int) [][]uint8 {
	matrix := make([][]uint8, height)
	for i := range matrix {
		matrix[i] = make([]uint8, width)
	}
	return matrix
}

func calculateNextState(p Params, startY, endY, width int, world [][]byte) [][]uint8 {
	semiHeight := endY - startY
	height := p.ImageHeight

	newWorld := makeMatrix(semiHeight, width)

	for y := startY; y < endY; y++ {
		for x := 0; x < width; x++ {
			sum := (world[(y-1+height)%height][(x-1+width)%width])/255 + (world[(y-1+height)%height][(x+width)%width])/255 + (world[(y-1+height)%height][(x+1+width)%width])/255 +
				(world[(y+height)%height][(x-1+width)%width])/255 + (world[(y+height)%height][(x+1+width)%width])/255 +
				(world[(y+1+height)%height][(x-1+width)%width])/255 + (world[(y+1+height)%height][(x+width)%width])/255 + (world[(y+1+height)%height][(x+1+width)%width])/255
			if world[y][x] == 255 {
				if sum < 2 {
					newWorld[y-startY][x] = 0
				} else if sum == 2 || sum == 3 {
					newWorld[y-startY][x] = 255
				} else {
					newWorld[y-startY][x] = 0
				}
			} else {
				if sum == 3 {
					newWorld[y-startY][x] = 255
				}
			}
		}
	}
	return newWorld
}

func worker(p Params, startY, endY, width int, newWorld [][]byte, out chan<- [][]uint8) {
	imagePart := calculateNextState(p, startY, endY, p.ImageWidth, newWorld)
	out <- imagePart
}

func countingCells(p Params, world [][]uint8) int {
	cells := 0

	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			if world[y][x] == 255 {
				cells++
			}
		}
	}
	return cells
}

func distributor(p Params, c distributorChannels, keyPresses <-chan rune) {
	c.ioCommand <- ioInput
	c.ioFilename <- fmt.Sprintf("%dx%d", p.ImageWidth, p.ImageHeight)

	//height := p.ImageHeight
	width := p.ImageWidth

	var m sync.Mutex
	var finalWorld [][]uint8
	var newPixelData [][]uint8

	var finishedTurns int

	fin := make(chan bool)

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

	//newWorld := world

	//keypress (STEP 5)

	// TODO: CAN I add a channel or import library from the github <- ask
	go func() {
		keyboard := <-keyPresses
		switch keyboard {
		//case "s":
		//	//somethin
		}
	}()

	//Ticker (STEP 3)
	go func() {
		t := time.NewTicker(2 * time.Second)
		for {
			select {
			case <-t.C: //when value passed down to the channel, alert events
				c.events <- AliveCellsCount{finishedTurns, countingCells(p, world)}

			case f := <-fin: //passing down the value to the channel
				if f == true { //if true, stop
					t.Stop()
				}
			}

		}
	}()

	threads := p.Threads
	//var workersWorld [][]uint8

	if threads == 1 {
		for turn := 0; turn < p.Turns; turn++ {
			fmt.Sprintf("IF Turn: ", turn)
			world = calculateNextState(p, 0, p.ImageHeight, width, world)
			//result = calculateInitial(p, width, world)
			finishedTurns++
			c.events <- TurnComplete{turn + 1}
		}
		finalWorld = world
	} else {
		workerHeight := p.ImageHeight / threads
		out := make([]chan [][]uint8, threads)
		for i := range out {
			out[i] = make(chan [][]uint8)
		}

		// TODO: Execute all turns of the Game of Life.
		for turn := 0; turn < p.Turns; turn++ {
			newPixelData = make([][]uint8, 0)
			if p.ImageHeight%p.Threads == 0 { // when the thread can be divided
				for i := 0; i < p.Threads; i++ {
					go worker(p, i*workerHeight, (i+1)*workerHeight, p.ImageWidth, world, out[i])
				}
				m.Lock()
				for j := 0; j < p.Threads; j++ {
					result := <-out[j]
					newPixelData = append(newPixelData, result...)
				}
				m.Unlock()
			} else { // when the thread cannot be divided by the thread(has remainders)
				for i := 0; i < p.Threads; i++ {
					if i == (p.Threads - 1) { // if it is the last thread  //half of them working on 36 and others dividing the remainder evenly
						go worker(p, i*workerHeight, p.ImageHeight, p.ImageWidth, world, out[i])
					} else { //else
						go worker(p, i*workerHeight, (i+1)*workerHeight, p.ImageWidth, world, out[i])
					}
				}
				m.Lock()
				for j := 0; j < p.Threads; j++ {
					result := <-out[j]
					newPixelData = append(newPixelData, result...)
				}
				m.Unlock()
			}
			world = newPixelData
			finishedTurns++
			c.events <- TurnComplete{turn + 1}
		}
		finalWorld = world
	}

	// TODO: Report the final state using FinalTurnCompleteEvent.

	c.events <- FinalTurnComplete{
		CompletedTurns: p.Turns,
		Alive:          calculateAliveCells(p, finalWorld),
	}

	fin <- true

	//output (Step 3)
	c.ioCommand <- ioOutput
	c.ioFilename <- fmt.Sprintf("%dx%d", p.ImageWidth, p.ImageHeight)
	//Sending out the output
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			result := finalWorld[y][x]
			c.ioOutput <- result
		}
	}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{p.Turns, Quitting}

	close(c.events)
}
