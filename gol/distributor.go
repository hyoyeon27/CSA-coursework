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

//func calculateNextState(p Params, world [][]byte) [][]byte {
//	newWorld := make([][]byte, p.ImageHeight)
//	for y := range newWorld {
//		newWorld[y] = make([]byte, p.ImageWidth)
//	}
//
//	for y := 0; y < p.ImageHeight; y++ {
//		for x := 0; x < p.ImageWidth; x++ {
//			sum := (world[(y-1+p.ImageHeight)%p.ImageHeight][(x-1+p.ImageWidth)%p.ImageWidth])/255 + (world[(y-1+p.ImageHeight)%p.ImageHeight][(x+p.ImageWidth)%p.ImageWidth])/255 + (world[(y-1+p.ImageHeight)%p.ImageHeight][(x+1+p.ImageWidth)%p.ImageWidth])/255 +
//				(world[(y+p.ImageHeight)%p.ImageHeight][(x-1+p.ImageWidth)%p.ImageWidth])/255 + (world[(y+p.ImageHeight)%p.ImageHeight][(x+1+p.ImageWidth)%p.ImageWidth])/255 +
//				(world[(y+1+p.ImageHeight)%p.ImageHeight][(x-1+p.ImageWidth)%p.ImageWidth])/255 + (world[(y+1+p.ImageHeight)%p.ImageHeight][(x+p.ImageWidth)%p.ImageWidth])/255 + (world[(y+1+p.ImageHeight)%p.ImageHeight][(x+1+p.ImageWidth)%p.ImageWidth])/255
//			if world[y][x] == 255 {
//				if sum < 2 {
//					newWorld[y][x] = 0
//				} else if sum == 2 || sum == 3 {
//					newWorld[y][x] = 255
//				} else {
//					newWorld[y][x] = 0
//				}
//			} else {
//				if sum == 3 {
//					newWorld[y][x] = 255
//				}
//			}
//		}
//	}
//	return newWorld
//}

// ======================================================================================================================================
func makeMatrix(height, width int) [][]uint8 {
	matrix := make([][]uint8, height)
	for i := range matrix {
		matrix[i] = make([]uint8, width)
	}
	return matrix
}

//func makeImmutableMatrix(matrix [][]uint8) func(y, x int) uint8 {
//	return func(y, x int) uint8 {
//		return matrix[y][x]
//	}
//}

func calculateNextState(startY, endY, startX, endX int, world [][]byte) [][]uint8 {
	height := endY - startY
	width := endX - startX

	newWorld := make([][]byte, height)
	for y := range newWorld {
		newWorld[y] = make([]byte, width)
	}
	//radius := 2
	//midPoint := (5*5 + 1) / 2

	//filteredMatrix := makeMatrix(height, width)
	//filterValues := make([]int, 5*5)

	for y := startY; y < startY+height; y++ {
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

func worker(startY, endY, startX, endX int, newWorld [][]byte, out chan<- [][]uint8) {
	imagePart := calculateNextState(startY, endY, startX, endX, newWorld)
	out <- imagePart
}

//func filter(filepathIn, filepathOut string, threads int) {
//	//image.RegisterFormat("png", "PNG", png.Decode, png.DecodeConfig)
//	//image.RegisterFormat("jpeg", "jpeg", jpeg.Decode, jpeg.DecodeConfig)
//
//	//img := loadImage(filepathIn)
//	//bounds := img.Bounds()
//	height := bounds.Dy()
//	width := bounds.Dx()
//
//	immutableData := makeImmutableMatrix(getPixelData(img))
//	var newPixelData [][]uint8
//
//	if threads == 1 {
//		newPixelData = medianFilter(0, height, 0, width, immutableData)
//	} else {
//		workerHeight := height / threads
//		out := make([]chan [][]uint8, threads)
//		for i := range out {
//			out[i] = make(chan [][]uint8)
//		}
//
//		for i := 0; i < threads; i++ {
//			go worker(i*workerHeight, (i+1)*workerHeight, 0, width, immutableData, out[i])
//		}
//
//		newPixelData = makeMatrix(0, 0)
//
//		for i := 0; i < threads; i++ {
//			part := <-out[i]
//			newPixelData = append(newPixelData, part...)
//		}
//	}
//	//
//	//imout := image.NewGray(image.Rect(0, 0, width, height))
//	//imout.Pix = flattenImage(newPixelData)
//	//ofp, _ := os.Create(filepathOut)
//	//defer ofp.Close()
//	//err := png.Encode(ofp, imout)
//	//check(err)
//}

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

	var newPixelData [][]uint8
	threads := p.Threads

	for turn := 0; turn < p.Turns; turn++ {
		if threads == 1 {
			fmt.Println("threads == 1")
			newPixelData = calculateNextState(0, height, 0, width, newWorld)
		} else {
			fmt.Println("else")
			workerHeight := height / threads
			out := make([]chan [][]uint8, threads)

			for i := range out {
				out[i] = make(chan [][]uint8)
			}

			for i := 0; i < threads; i++ {
				go worker(i*workerHeight, (i+1)*workerHeight, 0, width, newWorld, out[i])
			}

			newPixelData = makeMatrix(0, 0)

			for i := 0; i < threads; i++ {
				part := <-out[i]
				newPixelData = append(newPixelData, part...)
			}
		}
	}
	workersWorld := newPixelData
	fmt.Println(workersWorld)

	// TODO: Execute all turns of the Game of Life.
	//for turn := 0; turn < p.Turns; turn++ {
	//	workersWorld = calculateNextState(p, workersWorld)
	//}

	// TODO: Report the final state using FinalTurnCompleteEvent.

	c.events <- FinalTurnComplete{
		CompletedTurns: p.Turns,
		Alive:          calculateAliveCells(p, workersWorld),
	}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{p.Turns, Quitting}

	//ticker := time.NewTicker(2 * time.Second)
	//for turn := 0; turn < p.Turns; turn++ {
	//
	//	AliveCellsCountChan := make(chan AliveCellsCount)
	//
	//	go func(turn int) {
	//		for {
	//			select {
	//			case <-ticker.C:
	//				AliveCellsCountChan <- AliveCellsCount{
	//					CompletedTurns: p.Turns,
	//					CellsCount:     binary.Size(calculateAliveCells(p, world))}
	//			}
	//		}
	//	}(turn)
	//	AliveCellsCount := <-AliveCellsCountChan
	//	c.events <- AliveCellsCount
	//
	//	c.events <- TurnComplete{
	//		CompletedTurns: p.Turns,
	//	}
	//}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}

//func distributor(p Params, c distributorChannels) {
//	c.ioCommand <- ioInput
//	c.ioFilename <- fmt.Sprintf("%dx%d", p.ImageWidth, p.ImageHeight)
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
//	// TODO: Execute all turns of the Game of Life.
//	for turn := 0; turn < p.Turns; turn++ {
//		newWorld = calculateNextState(p, newWorld)
//	}
//
//	// TODO: Report the final state using FinalTurnCompleteEvent.
//
//	c.events <- FinalTurnComplete{
//		CompletedTurns: p.Turns,
//		Alive:          calculateAliveCells(p, newWorld),
//	}
//
//	// Make sure that the Io has finished any output before exiting.
//	c.ioCommand <- ioCheckIdle
//	<-c.ioIdle
//
//	c.events <- StateChange{p.Turns, Quitting}
//
//	ticker := time.NewTicker(2 * time.Second)
//	for turn := 0; turn < p.Turns; turn++ {
//
//		AliveCellsCountChan := make(chan AliveCellsCount)
//
//		go func(turn int) {
//			for {
//				select {
//				case <-ticker.C:
//					AliveCellsCountChan <- AliveCellsCount{
//						CompletedTurns: p.Turns,
//						CellsCount:     binary.Size(calculateAliveCells(p, newWorld))}
//				}
//			}
//		}(turn)
//		AliveCellsCount := <-AliveCellsCountChan
//		c.events <- AliveCellsCount
//
//		c.events <- TurnComplete{
//			CompletedTurns: p.Turns,
//		}
//	}
//
//	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
//	close(c.events)
//}
