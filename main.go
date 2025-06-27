package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jonathanhecl/gollama"
	"github.com/jxskiss/mcli"
)

var (
	ctx     context.Context
	m       *gollama.Gollama
	errFile *os.File
	args    Args
	db      *BadgerCache
	c       Counter
)

func main() {
	go handleTermination()
	defer cleanup()

	mcli.SetOptions(mcli.Options{
		EnableFlagCompletionForAllCommands: true,
	})

	_, _ = mcli.Parse(&args)
	errFile, _ = os.OpenFile(args.GetErrFilePath(), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)

	var err error
	db, err = NewBadgerCache("./cache")
	if err != nil {
		fmt.Println(err)
		return
	}

	ctx = context.Background()
	m = gollama.New(args.Model)
	err = m.PullIfMissing(ctx)
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
	processImagesParallel(images)
}

func processImagesParallel(images []Image) {
	jobs := make(chan Image, len(images))
	var wg sync.WaitGroup
	var errMutex sync.Mutex

	writeError := func(format string, args ...interface{}) {
		errMutex.Lock()
		defer errMutex.Unlock()
		fmt.Fprintf(errFile, format, args...)
		fmt.Fprintf(os.Stderr, format, args...)
	}

	for i := 0; i < args.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for image := range jobs {
				processImage(image, writeError)
			}
		}()
	}

	for _, image := range images {
		jobs <- image
	}
	close(jobs)

	wg.Wait()
}

func processImage(image Image, writeError func(string, ...interface{})) {
	_, found := db.Get(image.GetFullPath())

	if !args.OverriteDescriptions && (found || image.hasDescription()) {
		if !found {
			_ = db.Set(image.GetFullPath(), "true")
			fmt.Printf(". Skipping...Image %s already has description.\n", image.Filename)
		} else {
			fmt.Printf(". Skipping...Image %s found in cache.\n", image.Filename)
		}
		return
	}

	start := time.Now()
	fmt.Printf("--- Started processing %s\n", image.Filename)
	defer func() {
		duration := time.Since(start)
		fmt.Printf("+ %s took %v ✅ Processed %d so far \n", image.Filename, duration, c.Value())
	}()

	err := image.ConvertToPNG()
	if err != nil {
		writeError("Error convertToPNG, err: %s\n", err.Error())
		return
	}

	var resText string
	if !args.SkipOCR {
		resText, err = image.extractText()
		if err != nil {
			writeError("Couldn't extract text for %s, err: %s\n", image.GetFullPath(), err.Error())
		}
		// fmt.Printf("Text for %s: %s\n", image.GetFullPath(), resText)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(args.Timeout)*time.Second)
	defer cancel()

	llmResponse, err := m.Chat(ctx, prompt, gollama.PromptImage{Filename: image.PngPath}, gollama.StructToStructuredFormat(LLMAnswer{}))
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			writeError("Chat timeout after %d seconds for image: %s\n", args.Timeout, image.GetFullPath())
		} else {
			writeError("Error chat for %s, err: %s\n", image.GetFullPath(), err.Error())
		}
		return
	}

	var item LLMAnswer
	_ = json.Unmarshal([]byte(llmResponse.Content), &item)

	resDescription := item.Description
	desc := resDescription + "\n" + resText
	err = image.addDescription(desc)
	if err != nil {
		writeError("Error adding description: %s\n", err.Error())
		return
	}

	_ = db.Set(image.GetFullPath(), desc)
	c.Inc()
}
