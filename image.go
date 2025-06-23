package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

type Image struct {
	Path      string
	PngPath   string
	Filename  string
	Extension string
}

func (image *Image) ConvertToPNG() error {
	_ = os.MkdirAll("./temp", 0o755)

	filename := filepath.Base(image.Path + image.Filename)
	nameWithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))
	outputPath := filepath.Join("./temp", nameWithoutExt+".png")

	// "1000x1000>" resizes so long side = 1000px, maintains aspect ratio, only shrinks (doesn't upscale)
	cmd := exec.Command("magick", image.Path+image.Filename, "-resize", "1000x1000>", outputPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ImageMagick conversion failed: %v", err)
	}

	image.PngPath = outputPath

	return nil
}

func (image *Image) IsSupportedByOCR() bool {
	ocrSupported := []string{".jpg", ".jpeg", ".tif", ".png"}
	return slices.Contains(ocrSupported, image.Extension)
}
