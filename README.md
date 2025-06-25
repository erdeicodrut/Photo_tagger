# Photo Tagger

A script to make an LLM give description to images so they are searchable by metadata in synology photos or other services like that.

### External dependencies

- EasyOCR
  https://github.com/JaidedAI/EasyOCR?tab=readme-ov-file#installation
- Exiftool
  https://exiftool.org/install.html
- Ollama
  https://ollama.com/download
  Make sure you have OLLAMA_HOST set in current env.
  If you are running it locally:
  `export OLLAMA_HOST=http://127.0.0.1:11434`
  otherwise your ollama server IP instead of `127.0.0.1`

Optional:

- sshfs
  can be used if you actually want to use it on a remote machine like a synology nas
  `mkdir ~/remote`
  `sshfs -o rw <user>@<ip>:./ ~/remote -v`

These are used as CLI dependencies so make sure they are accessible in the current env.

### Running

`go build .`
`./photos_tagger -h`
