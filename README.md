

# Go-Grab: Universal CLI Media Downloader

###### fun project 

**Go-Grab** is a high-performance TUI wrapper for `yt-dlp` built in **Go**. It replaces complex CLI flags with a beautiful, menu-driven interface to download media from **YouTube, Instagram, X (Twitter) and Reddit**.

![Demo](demo/demo.gif)



## Features

  * **Universal Support:** Works out-of-the-box with 1,000+ sites (YouTube, Reels, Reddit, etc.).
  * **Non-Blocking UI:** Uses **Goroutines** to decouple heavy download processes from the UI.
  * **Smart Parsing:** Real-time progress visualization via Regex parsing of raw `stdout` streams.

##  Prerequisites

You must have the download engine installed:

```bash
# MacOS / Linux / Windows (Winget)
brew install yt-dlp ffmpeg  # or 'sudo apt install' / 'winget install'
```

##  Installation

**Method 1: Install Globally (Recommended)**

```bash
git clone https://github.com/rohitkshirsagar19/go-grab.git
cd go-grab
go install .
```

*Now run `go-grab` from any terminal window.*

**Method 2: Run from Source**

```bash
go run main.go
```

## Under the Hood

  * **Architecture:** Event-driven [ELM Architecture](https://guide.elm-lang.org/architecture/) via Bubble Tea.
  * **Concurrency:** Spawns `yt-dlp` as a subprocess; streams output via **Pipes** and **Channels** to update the UI without freezing.

-----

*Created by [Rohit Kshirsagar](https://github.com/rohitkshirsagar19)*