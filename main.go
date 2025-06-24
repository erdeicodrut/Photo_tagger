package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/jonathanhecl/gollama"
	"github.com/jxskiss/mcli"
)

var (
	ctx     context.Context
	m       *gollama.Gollama
	errFile *os.File
	args    Args
)

func main() {
	go handleTermination()
	defer cleanup()

	mcli.SetOptions(mcli.Options{
		EnableFlagCompletionForAllCommands: true,
	})

	_, _ = mcli.Parse(&args)
	errFile, _ = os.OpenFile(args.GetErrFilePath(), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
	defer errFile.Close()

	ctx = context.Background()
	m = gollama.New(args.Model)
	err := m.PullIfMissing(ctx)
	if err != nil {
		fmt.Fprintf(errFile, "Error getting model, err: %s\n", err.Error())
		return
	}

	path := args.DirPath
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Fprintf(errFile, "Path does not exist, err: %s\n", err.Error())
		return
	}

	processDir(path)
}

func processDir(path string) {
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}

	fmt.Println("Processing dir " + path)
	files, err := os.ReadDir(path)
	if err != nil {
		fmt.Fprintf(errFile, "Error reading path, err: %s\n", err.Error())
		return
	}
	if args.IsRecursive {
		for _, f := range files {
			if f.IsDir() {
				dirPath := filepath.Join(path, f.Name())
				processDir(dirPath)
			}
		}
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

		if !args.OverriteDescriptions && image.hasDescription() {
			fmt.Printf("Image %s already has description. Skipping...\n", image.GetFullPath())
			continue
		}

		fmt.Println("Processing image:", image.GetFullPath())

		err := image.ConvertToPNG()
		if err != nil {
			fmt.Fprintf(errFile, "Error convertToPNG, err: %s\n", err.Error())
			return
		}

		var resText string
		if !args.SkipOCR {
			resText, err := image.extractText()
			if err != nil {
				fmt.Fprintf(errFile, "Couldn't extract text for %s, err: %s\n", image.GetFullPath(), err.Error())
			}

			fmt.Printf("Text for %s: %s\n", image.GetFullPath(), resText)
		}

		type LLMAnswer struct {
			Description string `required:"true"`
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(args.Timeout)*time.Second)
		defer cancel()

		llmResponse, err := m.Chat(ctx, prompt, gollama.PromptImage{Filename: image.PngPath}, gollama.StructToStructuredFormat(LLMAnswer{}))
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				fmt.Fprintf(errFile, "Chat timeout after %d seconds for image: %s\n", args.Timeout, image.GetFullPath())
			} else {
				fmt.Fprintf(errFile, "Error chat for %s, err: %s\n", image.GetFullPath(), err.Error())
			}

			continue
		}

		var item LLMAnswer
		_ = json.Unmarshal([]byte(llmResponse.Content), &item)

		fmt.Println("Description:", item.Description)

		resDescription := item.Description

		desc := resDescription + "\n" + resText

		err = image.addDescription(desc)
		if err != nil {
			fmt.Fprintf(errFile, "Error adding description: %s\n", err.Error())
			return
		}

	}
}

func handleTermination() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c
	fmt.Println("\nReceived interrupt signal, cleaning up...")
	cleanup()
	os.Exit(0)
}

func cleanup() {
	os.RemoveAll("./temp")
}
