package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io/fs"
	"math"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

var (
	// dpiBleedRatio  = 50.0 / 12.0
	// widthPlusBleed = 2.72
	widthNoBleed = 2.48
	bleedWidth   = 0.24
)

var (
	supportedFileTypes = []string{".png", ".jpg", ".jpeg"}
	cornerFix          = true
)

var (
	OW_OUT         = flag.Bool("w", false, "Overwrite output images with the same name that already exist.")
	OUT_FMT        = flag.String("fmt", "png", "Output format of cards. Supports [png, jpg, auto]. Auto keeps output format the same as input.")
	JPG_CORNER_FIX = flag.Bool("jcf", false, "Jpg Corner Fix - removes white artifacts from the corners of jpeg cards")
)

var (
	BLACK = color.RGBA{0, 0, 0, 255}
	RED   = color.RGBA{255, 0, 0, 255}
	GREEN = color.RGBA{0, 255, 0, 255}
	BLUE  = color.RGBA{0, 0, 255, 255}
)

func main() {
	inFileFolder := flag.String("i", ".", "File or folder to read in from")
	outFolder := flag.String("out", "./bleeder_out", "Folder to output to")
	flag.Parse()

	if err := os.MkdirAll(*outFolder, 0o700); err != nil {
		fmt.Printf("Failed to create output directory: %s\n", err.Error())
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
	} else {
		// input path is not directory
		handleWriteImage(*inFileFolder, *outFolder)
	}
	fmt.Printf("Done! See results in %s\n", *outFolder)
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
			err = errors.Join(fmt.Errorf("failed to read image file from %s", inFilePath), err)
			fmt.Println(err.Error())
		}
	}
}

func handleWriteImage(inFilePath, outFolderPath string) error {
	inFile, err := os.Open(inFilePath)
	if err != nil {
		return errors.Join(fmt.Errorf("failed to open image file [%s]", inFilePath), err)
	}

	img, newImgFormat, err := image.Decode(inFile)
	if err != nil {
		return err
	}
	// Need to re-write. Output file needs new name
	newImg := createCardWithBleed(img, newImgFormat)
	outImgBaseName := path.Base(inFile.Name())
	outImgPath := path.Join(outFolderPath, outImgBaseName)
	if _, err := os.Stat(outImgPath); *OW_OUT && !errors.Is(err, os.ErrNotExist) {
		fmt.Printf("Output file exists, skipping: [%s]\n", outImgBaseName)
		return nil
	}
	fmt.Printf("Adding bleed to image: [%s]\n", outImgBaseName)

	return saveImage(outImgPath, newImg, newImgFormat)
}

func saveImage(outPath string, imgOut image.Image, imgInFmt string) error {
	imgOutFmt := imgInFmt
	if *OUT_FMT != "auto" {
		imgOutFmt = *OUT_FMT
	}
	outPath = strings.TrimSuffix(outPath, filepath.Ext(outPath)) + "." + imgOutFmt
	imgOutFile, err := os.Create(outPath)
	if err != nil {
		return errors.Join(fmt.Errorf("Failed to create file: [%s]\n", outPath), err)
	}

	switch imgOutFmt {
	case "png":
		return png.Encode(imgOutFile, imgOut)
	case "jpg", "jpeg":
		return jpeg.Encode(imgOutFile, imgOut, &jpeg.Options{Quality: 100})
	}
	return fmt.Errorf("Failed to save image - unsupported format: %s", imgOutFmt)
}

func createCardWithBleed(cardImg image.Image, inImgFormat string) image.Image {
	// cardWithBleedBounds, cardNoBleedBounds := calculateBleedBounds(cardImg)
	bleedEdgePx := calculateBleedWidth(cardImg)

	cardWithBleedBounds := image.Rect(0, 0, cardImg.Bounds().Dx()+bleedEdgePx*2, cardImg.Bounds().Dy()+bleedEdgePx*2)
	// cardNoBleedBounds := image.Rect(bleedEdgePx, bleedEdgePx, cardImg.Bounds().Dx()+bleedEdgePx, cardWithBleedBounds.Dy()+bleedEdgePx)
	cardNoBleedBounds := cardImg.Bounds().Add(image.Point{bleedEdgePx, bleedEdgePx})

	cardWithBleed := image.NewRGBA(cardWithBleedBounds)

	// Draw black background
	draw.Draw(cardWithBleed, cardWithBleedBounds, &image.Uniform{C: BLACK}, image.Point{}, draw.Src)
	// Draw card onto image
	draw.Draw(cardWithBleed, cardNoBleedBounds, cardImg, image.Point{}, draw.Over)

	if cornerFix && (inImgFormat == "jpg" || inImgFormat == "jpeg") {
		fixCorners(cardWithBleed, bleedEdgePx, cardNoBleedBounds)
	}

	return cardWithBleed
}

// Overwrites the corners of the image to potentially fix issue if previously created with PNG
func fixCorners(img *image.RGBA, bleedWidth int, innerCardBounds image.Rectangle) {
	// bleedWidth := (cardWithBleedBounds.Dx() - innerCardBounds.Dx()) / 2 // Maybe pass this in instead of calculating
	rectWidth := int(float64(bleedWidth) * 0.75)

	blackImg := &image.Uniform{C: BLACK}

	// Top left
	origin := innerCardBounds.Min
	draw.Draw(img, image.Rect(origin.X, origin.Y, origin.X+rectWidth, origin.Y+bleedWidth), blackImg, image.Point{}, draw.Over)
	draw.Draw(img, image.Rect(origin.X, origin.Y, origin.X+bleedWidth, origin.Y+rectWidth), blackImg, image.Point{}, draw.Over)

	// Top right
	origin.X = innerCardBounds.Max.X
	draw.Draw(img, image.Rect(origin.X, origin.Y, origin.X-rectWidth, origin.Y+bleedWidth), blackImg, image.Point{}, draw.Over)
	draw.Draw(img, image.Rect(origin.X, origin.Y, origin.X-bleedWidth, origin.Y+rectWidth), blackImg, image.Point{}, draw.Over)

	// Bottom right
	origin.Y = innerCardBounds.Max.Y
	draw.Draw(img, image.Rect(origin.X, origin.Y, origin.X-rectWidth, origin.Y-bleedWidth), blackImg, image.Point{}, draw.Over)
	draw.Draw(img, image.Rect(origin.X, origin.Y, origin.X-bleedWidth, origin.Y-rectWidth), blackImg, image.Point{}, draw.Over)

	// Bottom left
	origin.X = innerCardBounds.Min.X
	draw.Draw(img, image.Rect(origin.X, origin.Y, origin.X+rectWidth, origin.Y-bleedWidth), blackImg, image.Point{}, draw.Over)
	draw.Draw(img, image.Rect(origin.X, origin.Y, origin.X+bleedWidth, origin.Y-rectWidth), blackImg, image.Point{}, draw.Over)
	// startPoint.Y += origCardBounds.Dy()
	// draw.Draw(img, image.Rect(0, 0, bleedWidth, bleedWidth), blackImg, startPoint, draw.Over)
}

func calculateBleedBounds(img image.Image) (cardWithBleedBounds image.Rectangle, origCardBounds image.Rectangle) {
	imgBounds := img.Bounds()
	cardWidth := imgBounds.Dx()
	cardHeight := imgBounds.Dy()

	bleedEdgeWidthPx := int(math.Round((float64(cardWidth)*bleedWidth)/widthNoBleed)) / 2

	// dpi := float64(bleedEdge) / bleedWidth

	// fmt.Printf("Card bleed: %dpx, dpi: %f\n", bleedEdge, dpi)

	newCardWidth := cardWidth + bleedEdgeWidthPx*2
	newCardHeight := cardHeight + bleedEdgeWidthPx*2
	cardWithBleedBounds = image.Rect(0, 0, newCardWidth, newCardHeight)
	origCardBounds = image.Rect(bleedEdgeWidthPx, bleedEdgeWidthPx, newCardWidth-bleedEdgeWidthPx, newCardHeight-bleedEdgeWidthPx)
	return cardWithBleedBounds, origCardBounds
}

func calculateBleedWidth(img image.Image) int {
	imgBounds := img.Bounds()
	cardWidth := imgBounds.Dx()

	return int(math.Round((float64(cardWidth)*bleedWidth)/widthNoBleed)) / 2
}
