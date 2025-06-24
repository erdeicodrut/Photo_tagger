package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/MaestroError/go-libheif"
	"golang.org/x/image/draw"
	"golang.org/x/image/tiff"
)

type Image struct {
	Path      string
	PngPath   string
	Filename  string
	Extension string
}

func (image *Image) GetFullPath() string {
	return filepath.Join(image.Path, image.Filename)
}

func (image *Image) IsOCRSupported() bool {
	return slices.Contains(ocrSupported, image.Extension)
}

func (image *Image) ConvertToPNG() error {
	_ = os.MkdirAll("./temp", 0o755)
	filename := filepath.Base(image.GetFullPath())
	nameWithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))
	outputPath := filepath.Join("./temp", nameWithoutExt+".png")

	img, err := loadImage(image.GetFullPath())
	if err != nil {
		return fmt.Errorf("failed to load image: %v", err)
	}

	// Resize so long side = 1000px, maintains aspect ratio, only shrinks (doesn't upscale)
	img = resizeToLongEdge(img, 1000, false)

	err = savePNG(img, outputPath)
	if err != nil {
		return fmt.Errorf("failed to save PNG: %v", err)
	}

	image.PngPath = outputPath
	return nil
}

func (image *Image) extractText() (string, error) {
	type easyocrOutput struct {
		Text string `json:"text"`
	}

	var imagePath string
	if image.IsOCRSupported() {
		imagePath = image.GetFullPath()
	} else {
		imagePath = image.PngPath
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(args.Timeout)*time.Second)
	defer cancel()

	output, err := exec.CommandContext(ctx,
		"easyocr", "-l", "en", "-f", imagePath,
		"--paragraph", "True",
		"--gpu", "True",
		"--output_format", "json",
	).Output()
	if err != nil {
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			return "", fmt.Errorf("OCR timeout after %d seconds for: %s\n", args.Timeout, imagePath)
		case err.(*exec.ExitError) != nil:
			return "", fmt.Errorf("OCR command failed for %s: %v\n", imagePath, err)
		default:
			return "", fmt.Errorf("OCR error for %s: %v\n", imagePath, err)
		}
	}

	if len(output) == 0 {
		return "", nil
	}
	var outputs []string
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 {
			continue
		}
		var item easyocrOutput
		_ = json.Unmarshal([]byte(line), &item)
		outputs = append(outputs, item.Text)
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return strings.TrimSpace(strings.ReplaceAll(strings.Join(outputs, " "), "\n", " ")), nil
}

func (image *Image) addDescription(description string) error {
	_, err := exec.Command(
		"exiftool", image.Path+image.Filename,
		fmt.Sprintf("-ImageDescription=%s", description),
		"-P",
		"-preserve",
		"-overwrite_original",
	).Output()
	return err
}

func (image *Image) hasDescription() bool {
	output, _ := exec.Command(
		"exiftool", image.Path+image.Filename,
	).Output()
	return strings.Contains(string(output), "Image Description")
}

func (image *Image) getExif() []byte {
	output, _ := exec.Command(
		"exiftool", image.Path+image.Filename,
	).Output()
	return output
}

func loadImage(filePath string) (image.Image, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".hif", ".heif", ".heic":
		return loadHEIF(filePath)
	case ".jpg", ".jpeg":
		return loadJPEG(filePath)
	case ".tif", ".tiff":
		return loadTIFF(filePath)
	case ".png":
		return loadPNG(filePath)
	default:
		file, err := os.Open(filePath)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		img, _, err := image.Decode(file)
		return img, err
	}
}

func loadHEIF(filePath string) (image.Image, error) {
	img, err := libheif.ReturnImageFromHeif(filePath)
	if err != nil {
		return nil, fmt.Errorf("HEIF decode error: %v", err)
	}
	return img, nil
}

func loadJPEG(filePath string) (image.Image, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return jpeg.Decode(file)
}

func loadTIFF(filePath string) (image.Image, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return tiff.Decode(file)
}

func loadPNG(filePath string) (image.Image, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return png.Decode(file)
}

func resizeToLongEdge(src image.Image, longEdge int, allowUpscale bool) image.Image {
	bounds := src.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	// If image is smaller and upscaling is disabled, return original
	if !allowUpscale && srcWidth <= longEdge && srcHeight <= longEdge {
		return src
	}

	var newWidth, newHeight int
	if srcWidth > srcHeight {
		newWidth = longEdge
		newHeight = int(float64(srcHeight) * float64(longEdge) / float64(srcWidth))
	} else {
		newHeight = longEdge
		newWidth = int(float64(srcWidth) * float64(longEdge) / float64(srcHeight))
	}

	dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

func savePNG(img image.Image, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, img)
}
