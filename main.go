package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jonathanhecl/gollama"
)

const (
	prompt = `
Provide a concise description of the image content for search purposes, including:
   - Main subjects/objects in the image
   - Setting/location type
   - Notable activities or scenes
   - Colors, style, or distinctive features
   - Don't miss any part of the image like objects and animals
For both use simple words and no derivations for smaller or larger, instead use a different word for size or other kind of attributes like that. Rembmber we are trying to optimise for search.
You can also add some simpler synonyms at the end of the description for better serachability.
don't use any markdown or other kind of forrmating, or newlines. give the text as clear as possible
	`
)

var (
	model  = "qwen2.5vl:3b"
	imgExt = []string{".heif", ".heic", ".jpg", ".jpeg", ".tif", ".png"}
)

func main() {
	ctx := context.Background()
	if len(os.Args) == 3 {
		model = os.Args[2]
	}
	m := gollama.New(model)
	err := m.PullIfMissing(ctx)
	if err != nil {
		fmt.Printf("Error getting model, err: %s", err.Error())
		return
	}

	path := os.Args[1]

	fmt.Println(path)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Printf("Path does not exist, err: %s", err.Error())
		return
	}

	files, err := os.ReadDir(path)
	if err != nil {
		fmt.Printf("Error reading path, err: %s", err.Error())
		return
	}

	images := []Image{}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		ext := filepath.Ext(file.Name())

		for _, v := range imgExt {
			if strings.EqualFold(ext, v) {
				images = append(images, Image{Path: path, Filename: file.Name(), Extension: strings.ToLower(ext)})
				break
			}
		}
	}

	for _, image := range images {
		fmt.Println("Processing image:", image.Filename)

		err := image.ConvertToPNG()
		if err != nil {
			fmt.Printf("Error convertToPNG, err: %s", err.Error())
			return
		}

		var resText string
		if image.IsSupportedByOCR() {
			resText = extractText(image.Path + image.Filename)
		} else {
			resText = extractText(image.PngPath)
		}
		fmt.Printf("Text for %s: %s\n", image.Filename, resText)

		type LLMAnswer struct {
			Description string `required:"true"`
		}

		resDesc, err := m.Chat(ctx, prompt, gollama.PromptImage{Filename: image.PngPath}, gollama.StructToStructuredFormat(LLMAnswer{}))
		if err != nil {
			fmt.Printf("Error chat, err: %s", err.Error())
			return
		}

		var item LLMAnswer
		_ = json.Unmarshal([]byte(resDesc.Content), &item)

		fmt.Println("Description:", item.Description)

	}
}

func extractText(imagePath string) string {
	type easyocrOutput struct {
		Text string `json:"text"`
	}

	output, err := exec.Command(
		"easyocr", "-l", "en", "-f", imagePath,
		"--paragraph", "True",
		"--gpu", "True",
		"--output_format", "json",
	).Output()
	if err != nil {
		return ""
	}
	if len(output) == 0 {
		return ""
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
		return ""
	}
	return strings.TrimSpace(strings.ReplaceAll(strings.Join(outputs, " "), "\n", " "))
}
