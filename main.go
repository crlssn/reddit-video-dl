package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

type Listing struct {
	Data Data `json:"data"`
}

type Data struct {
	Children []Child `json:"children"`
}

type Child struct {
	Data ChildData `json:"data"`
}

type ChildData struct {
	SecureMedia SecureMedia `json:"secure_media"`
}

type SecureMedia struct {
	RedditVideo RedditVideo `json:"reddit_video"`
}

type RedditVideo struct {
	FallbackUrl string `json:"fallback_url"`
}

type DownloadResult struct {
	Success bool
	Error   error
}

var verboseOutput = flag.Bool("v", false, "Display encoding output")

func main() {
	flag.Parse()

	var url string
	fmt.Print("Enter Reddit URL: ")

	for {
		fmt.Scanln(&url)

		if strings.HasPrefix(url, "https://www.reddit.com") {
			break
		} else {
			fmt.Print("The URL is incorrect, please try again: ")
		}
	}

	if ! strings.HasSuffix(url, ".json") {
		url = url + ".json"
	}

	err, body := getBodyFromUrl(url)

	var Listings []Listing

	err = json.Unmarshal(body, &Listings)

	if err != nil {
		log.Fatal(err)
	}

	videoUrl := Listings[0].Data.Children[0].Data.SecureMedia.RedditVideo.FallbackUrl
	if len(videoUrl) < 1 {
		fmt.Print("No media file was found")
		os.Exit(0)
	}

	videoId := strings.Split(videoUrl, "/")[3]
	videoFilePath := videoId + "_input.mp4"

	videoCh := make(chan DownloadResult)
	go DownloadFile(videoFilePath, videoUrl, videoCh)

	audioUrl := strings.Join([]string{"https://v.redd.it", videoId, "audio"}, "/")
	audioFilePath := videoId + "_input.mp3"

	audioCh := make(chan DownloadResult)
	go DownloadFile(audioFilePath, audioUrl, audioCh)

	fmt.Println("Downloading files...")

	select {
	case videoDl := <-videoCh:
		audioDl := <-audioCh
		if videoDl.Error != nil {
			log.Fatal(videoDl.Error)
		}

		if audioDl.Error != nil {
			log.Fatal(audioDl.Error)
		}

		if videoDl.Success && audioDl.Success {
			concatFile := concatFiles(videoFilePath, audioFilePath, videoId)

			fmt.Printf("Video file is completed (file://%s)\n", concatFile)
		}
	}
}

func getBodyFromUrl(url string) (error, []byte) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.14; rv:68.0) Gecko/20100101 Firefox/68.0")
	res, err := client.Do(req)

	if err != nil {
		log.Fatal(err)
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		log.Fatal(err)
	}

	return err, body
}

func concatFiles(videoFilePath string, audioFilePath string, videoId string) string {
	fmt.Println("Concatenating audio and video...")

	wd, _ := os.Getwd()

	resultFile := wd + videoId + ".mp4"
	args := []string{"-y", "-i", videoFilePath, "-i", audioFilePath, resultFile}

	cmd := exec.Command("ffmpeg", args...)
	if *verboseOutput {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	err := cmd.Run()

	if err != nil {
		log.Fatal(err)
	}

	err = os.Remove(audioFilePath)

	if err != nil {
		log.Fatal(err)
	}

	err = os.Remove(videoFilePath)

	if err != nil {
		log.Fatal(err)
	}

	return resultFile
}

func DownloadFile(filepath string, url string, ch chan<- DownloadResult) {
	resp, err := http.Get(url)

	if err != nil {
		ch <- DownloadResult{false, err}
	}

	defer resp.Body.Close()

	file, err := os.Create(filepath)

	if err != nil {
		ch <- DownloadResult{false, err}
	}

	defer file.Close()

	_, err = io.Copy(file, resp.Body)

	ch <- DownloadResult{true, err}
}
