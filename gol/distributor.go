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

var mutex sync.Mutex

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
	mutex.Lock()
	fmt.Println("ENTERED COUNTING CELLS")
	cells := 0

	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			if world[y][x] == 255 {
				cells++
			}
		}
	}
	mutex.Unlock()
	return cells
}

func distributor(p Params, c distributorChannels, keyPresses <-chan rune) {
	height := p.ImageHeight
	width := p.ImageWidth

	c.ioCommand <- ioInput
	c.ioFilename <- fmt.Sprintf("%dx%d", width, height)

	var m sync.Mutex
	var cellsWorld [][]uint8

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

	// Make sure to send this event for all cells that are alive when the image is loaded in.
	flip := calculateAliveCells(p, world)
	for _, cell := range flip {
		c.events <- CellFlipped{0, cell}
	}

	//TODO: Ticker (STEP 3)
	go func() {
		t := time.NewTicker(2 * time.Second)
		for {
			select {
			case <-t.C: //when value passed down to the channel, alert events
				fmt.Println("FINISHEDTURNS IN GOROUT:", finishedTurns, "ALIVE CELLS:", countingCells(p, cellsWorld))
				c.events <- AliveCellsCount{finishedTurns, countingCells(p, cellsWorld)}

			case f := <-fin: //passing down the value to the channel
				if f == true { //if true, stop
					t.Stop()
				}
			}

		}
	}()

	//keypress (STEP 5)
	// TODO: CAN I add a channel or import library from the github <- ask
	go func() {
		keyboard := <-keyPresses
		for {
			if keyboard == 's' {
				c.ioCommand <- ioOutput
				c.ioFilename <- fmt.Sprintf("%dx%d", height, width)
				for y := 0; y < height; y++ {
					for x := 0; x < width; x++ {
						c.ioOutput <- world[y][x]
					}
				}
				c.events <- ImageOutputComplete{p.Turns, fmt.Sprintf("%dx%d", width, height)}
			} else if keyboard == 'q' {
				c.ioCommand <- ioOutput
				c.ioFilename <- fmt.Sprintf("%dx%d", height, width)
				for y := 0; y < height; y++ {
					for x := 0; x < width; x++ {
						c.ioOutput <- world[y][x]
					}
				}
				c.events <- ImageOutputComplete{p.Turns, fmt.Sprintf("%dx%d", width, height)}
				c.ioCommand <- ioCheckIdle
				<-c.ioIdle
				c.events <- StateChange{p.Turns, Quitting}
				close(c.events)
				//os.Exit(0)
			} else if keyboard == 'p' {
				c.events <- StateChange{finishedTurns, Paused}
				fmt.Println("Current turn being processed: ", finishedTurns)
				m.Lock()
				keyboard = <-keyPresses
				if keyboard == 'p' {
					c.events <- StateChange{finishedTurns, Executing}
					fmt.Println("Continuing")
				}
				m.Unlock()

			}
		}
	}()

	threads := p.Threads
	//var workersWorld [][]uint8
	//TODO: Workers and Threads
	//if threads == 1 {
	//	for turn := 0; turn < p.Turns; turn++ {
	//		fmt.Println("Turn: ", turn)
	//		chk := calculateAliveCells(p, world)
	//		for _, cell := range chk {
	//			c.events <- CellFlipped{turn + 1, cell}
	//		}
	//		mutex.Lock()
	//		newWorld = calculateNextState(p, 0, height, width, newWorld)
	//		cellsWorld = newWorld
	//		mutex.Unlock()
	//		finishedTurns++
	//		c.events <- TurnComplete{turn + 1}
	//	}
	//finalWorld = world
	//} else {
	workerHeight := height / threads
	out := make([]chan [][]uint8, threads)
	for i := range out {
		out[i] = make(chan [][]uint8)
	}

	// TODO: Execute all turns of the Game of Life.
	for turn := 0; turn < p.Turns; turn++ {
		//newPixelData = make([][]uint8, 0)
		chk := calculateAliveCells(p, world)
		for _, cell := range chk {
			c.events <- CellFlipped{turn + 1, cell}
		}
		if height%threads == 0 { // when the thread can be divided
			for i := 0; i < threads; i++ {
				go worker(p, i*workerHeight, (i+1)*workerHeight, width, newWorld, out[i])
			}
			m.Lock()
			newWorld = makeMatrix(0, 0)

			for j := 0; j < threads; j++ {
				output := <-out[j]
				newWorld = append(newWorld, output...)
			}
			m.Unlock()
		} else { // when the thread cannot be divided by the thread(has remainders)
			for i := 0; i < threads; i++ {
				if i == (threads - 1) { // if it is the last thread  //half of them working on 36 and others dividing the remainder evenly
					go worker(p, i*workerHeight, height, width, newWorld, out[i])
				} else { //else
					go worker(p, i*workerHeight, (i+1)*workerHeight, width, newWorld, out[i])
				}
			}
			m.Lock()
			newWorld = makeMatrix(0, 0)

			for j := 0; j < threads; j++ {
				output := <-out[j]
				newWorld = append(newWorld, output...)
			}
			m.Unlock()
		}
		//newWorld = newPixelData
		mutex.Lock()
		cellsWorld = newWorld
		mutex.Unlock()
		finishedTurns++
		fmt.Println("FINISHEDTURNS: ", finishedTurns)

		c.events <- TurnComplete{turn + 1}
	}
	//finalWorld = world
	//}
	finalWorld := newWorld

	// TODO: Report the final state using FinalTurnCompleteEvent.

	c.events <- FinalTurnComplete{
		CompletedTurns: p.Turns,
		Alive:          calculateAliveCells(p, finalWorld),
	}

	fin <- true

	//output (Step 3)
	c.ioCommand <- ioOutput
	c.ioFilename <- fmt.Sprintf("%dx%d", width, height)
	//Sending out the output
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			result := finalWorld[y][x]
			c.ioOutput <- result
		}
	}

	c.events <- ImageOutputComplete{p.Turns, fmt.Sprintf("%dx%d", width, height)}
	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{p.Turns, Quitting}

	close(c.events)
}
