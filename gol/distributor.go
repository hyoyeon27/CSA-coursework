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
	workersHeight := endY - startY
	height := p.ImageHeight

	newWorld := makeMatrix(workersHeight, width)

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
	imagePart := calculateNextState(p, startY, endY, width, newWorld)
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

	height := p.ImageHeight
	width := p.ImageWidth

	var mutex sync.Mutex

	var finishedTurns int

	fin := make(chan bool)

	// distributor divides the work between workers and interacts with other goroutines.
	// TODO: Create a 2D slice to store the world.
	world := make([][]byte, height)
	for y := range world {
		world[y] = make([]byte, width)
		for x := range world[y] {
			if <-c.ioInput > 0 {
				world[y][x] = 255
			} else {
				world[y][x] = 0
			}
		}
	}

	newWorld := world

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

	for turn := 0; turn < p.Turns; turn++ {
		if threads == 1 {
			newWorld = calculateNextState(p, 0, height, width, newWorld)
		} else {
			workerHeight := height / threads
			outFir := make([]chan [][]uint8, threads)
			for i := range outFir {
				outFir[i] = make(chan [][]uint8)
			}

			outSec := make([]chan [][]uint8, threads)
			for i := range outSec {
				outSec[i] = make(chan [][]uint8)
			}

			if height%threads == 0 { // when the thread can be divided
				for i := 0; i < threads; i++ {
					go worker(p, i*workerHeight, (i+1)*workerHeight, width, newWorld, outFir[i])
				}

				newWorld = makeMatrix(0, 0)

				for i := 0; i < threads; i++ {
					part := <-outFir[i]
					mutex.Lock()
					newWorld = append(newWorld, part...)
					mutex.Unlock()
				}

			} else { // when the thread cannot be divided by the thread(has remainders)
				for i := 0; i < threads; i++ {
					if i == (p.Threads - 1) { // if it is the last thread
						go worker(p, i*workerHeight, height, width, newWorld, outSec[i])
					} else { //else
						go worker(p, i*workerHeight, (i+1)*workerHeight, width, newWorld, outSec[i])
					}
				}

				newWorld = makeMatrix(0, 0)

				for i := 0; i < threads; i++ {
					part := <-outSec[i]
					mutex.Lock()
					newWorld = append(newWorld, part...)
					mutex.Unlock()
				}
			}
		}
	}

	finalWorld := newWorld

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
