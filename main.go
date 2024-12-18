package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io/fs"
	"math"
	"os"
	"path"
	"slices"
	"sync"
)

var (
	// dpiBleedRatio  = 50.0 / 12.0
	widthPlusBleed = 2.72
	widthNoBleed   = 2.48
	bleedWidth     = 0.24
)

var (
	supportedFileTypes = []string{".png", ".jpg", ".jpeg"}
	OW                 = false
)

func main() {
	inFileFolder := flag.String("i", ".", "File or folder to read in from")
	outFolder := flag.String("out", "./bleeder_out", "Folder to output to")
	overwrite := flag.Bool("r", false, "Overwrite files with the same name in output dir. If false, images are skipped.")
	flag.Parse()

	OW = *overwrite

	if err := os.MkdirAll(*outFolder, 0o700); err != nil {
		fmt.Printf("Failed to create dir: %s\n", err.Error())
		os.Exit(1)
	}

	if info, err := os.Stat(*inFileFolder); err == nil && info.IsDir() {
		// input path is directory
		entries, err := os.ReadDir(*inFileFolder)
		if err != nil {
			fmt.Printf("Failed to read input directory: %s\n", err.Error())
			os.Exit(1)
		}
		var wg sync.WaitGroup
		jobs := make(chan string, 100)
		go queueImageJobs(*inFileFolder, entries, jobs)
		for range 8 {
			wg.Add(1)
			go handleWriteImageWorker(jobs, *outFolder, &wg)
		}
		wg.Wait()
		fmt.Println("Done!")
	} else {
		// input path is not directory
		handleWriteImage(*inFileFolder, *outFolder)
	}
}

func queueImageJobs(inFolder string, entries []fs.DirEntry, jobs chan string) {
	for _, entry := range entries {
		if !entry.IsDir() && slices.Contains(supportedFileTypes, path.Ext(entry.Name())) {
			inFilePath := path.Join(inFolder, entry.Name())
			jobs <- inFilePath
		}
	}
	close(jobs)
}

func handleWriteImageWorker(jobs <-chan string, outFolderPath string, wg *sync.WaitGroup) {
	defer wg.Done()
	for inFilePath := range jobs {
		err := handleWriteImage(inFilePath, outFolderPath)
		if err != nil {
			fmt.Printf("Failed to handleWriteImage: %s\n", err.Error())
		}
	}
}

func handleWriteImage(inFilePath, outFolderPath string) error {
	inFile, err := os.Open(inFilePath)
	if err != nil {
		errors.Join()
		return fmt.Errorf("Failed to read image file from %s [err: %s]\n", inFilePath, err.Error())
	}

	img, format, err := image.Decode(inFile)
	newImg := createNewImage(img)
	outImgPath := path.Join(outFolderPath, path.Base(inFile.Name()))
	if _, err := os.Stat("/path/to/whatever"); !OW && !errors.Is(err, os.ErrNotExist) {
		fmt.Printf("Output file exists, skipping: %s\n", outImgPath)
		return nil
		// File already exists
	}
	fmt.Printf("Saving new image to %s\n", outImgPath)
	outImg, err := os.Create(outImgPath)
	if err != nil {
		fmt.Printf("Failed to create outImg: %s\n", err.Error())
		return err
	}
	fmt.Printf("Read img type: %s\n", format)

	// switch format {
	// case "png":
	png.Encode(outImg, newImg)
	return nil
}

func createNewImage(oldImg image.Image) image.Image {
	newBounds, oldBounds := calculateNewImageWithBounds(oldImg)

	newCard := image.NewRGBA(newBounds)

	black := color.RGBA{0, 0, 0, 255}

	draw.Draw(newCard, newBounds, &image.Uniform{C: black}, image.Point{}, draw.Src)
	// fmt.Printf("Drawing old card: oldBounds: %s, drawPoint: %s\n", oldBounds.String(), oldBounds.Min.String())
	draw.Draw(newCard, oldBounds, oldImg, image.Point{}, draw.Over)

	return newCard
}

func calculateNewImageWithBounds(img image.Image) (newCardBounds image.Rectangle, oldCardBounds image.Rectangle) {
	imgBounds := img.Bounds()
	cardWidth := imgBounds.Dx()
	cardHeight := imgBounds.Dy()

	bleedEdge := int(math.Round((float64(cardWidth)*bleedWidth)/widthNoBleed)) / 2

	// dpi := float64(bleedEdge) / bleedWidth

	// fmt.Printf("Card bleed: %dpx, dpi: %f\n", bleedEdge, dpi)

	newCardWidth := cardWidth + bleedEdge*2
	newCardHeight := cardHeight + bleedEdge*2
	newCardBounds = image.Rect(0, 0, newCardWidth, newCardHeight)
	oldCardBounds = image.Rect(bleedEdge, bleedEdge, newCardWidth-bleedEdge, newCardHeight-bleedEdge)
	// fmt.Printf("New image size: %dx%d, old image drawn: %s\n", newCardWidth, newCardHeight, oldCardBounds.String())
	return newCardBounds, oldCardBounds
}
