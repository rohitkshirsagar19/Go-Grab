package main

import (
	"fmt"
	"log"
	"os/exec" // runing ext commands
)

func main() {

	videoURL := "https://youtu.be/-iRTJj5g4Ks?si=QCFGsSoidy-j4jQf" // test video

	fmt.Println("Attempting to download video ...")

	cmd := exec.Command("yt-dlp", "-f", "mp4", "-o", "test_video.mp4", videoURL)

	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Fatalf("Command failed: %v \n Output: %s", err, string(output))
	}

	fmt.Println("Success!")
	fmt.Println(string(output)) // printing yt-dlp logs

}
