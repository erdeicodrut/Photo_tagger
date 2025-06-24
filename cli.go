package main

import (
	"fmt"
	"time"
)

type Args struct {
	ErrFilePath          string `cli:"-e, --error, Error file path for error logs." default:"./errors"`
	Model                string `cli:"-m, --model, LLM model to be used. Should be a vision capable model." default:"qwen2.5vl:3b"`
	Timeout              int    `cli:"-t, --timeout, Timeout in seconds for per image processing task" default:"20"`
	Workers              int    `cli:"-w, --workers, Number of workers to be started for paralel processing." default:"4"`
	OverriteDescriptions bool   `cli:"-o, --overrite, Overrite existing descriptions. By default it skips images with descriptions."`
	IsRecursive          bool   `cli:"-r, --recursive, Do the processing recursively in subdirectories"`
	SkipOCR              bool   `cli:"-so, --skip-ocr, Skip OCR processing of images"`

	DirPath string `cli:"#R, directory, Directory to process images in"`
}

func (a *Args) GetErrFilePath() string {
	return fmt.Sprintf("%s_%s", a.ErrFilePath, time.Now().Format("20060102_150405"))
}
