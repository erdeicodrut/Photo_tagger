package main

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
	imgExt       = []string{".hif", ".heif", ".heic", ".jpg", ".jpeg", ".tif", ".png"}
	ocrSupported = []string{".jpg", ".jpeg", ".png"}
)
