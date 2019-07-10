package main

import (
	"encoding/json"
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
	Completed bool
	Error     error
}

// flags for verbose

func main() {
	//url := "https://www.reddit.com/r/HadToHurt/comments/c9i6pj/ouch/ "
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

	var Listings []Listing

	err = json.Unmarshal(body, &Listings)

	if err != nil {
		log.Fatal(err)
	}

	videoUrl := Listings[0].Data.Children[0].Data.SecureMedia.RedditVideo.FallbackUrl
	videoId := strings.Split(videoUrl, "/")[3]
	videoFilePath := videoId + "_input.mp4"

	videoDownloaded := make(chan DownloadResult)
	go DownloadFile(videoFilePath, videoUrl, videoDownloaded)

	audioUrl := strings.Join([]string{"https://v.redd.it", videoId, "audio"}, "/")
	audioFilePath := videoId + "_input.mp3"

	audioDownloaded := make(chan DownloadResult)
	go DownloadFile(audioFilePath, audioUrl, audioDownloaded)

	select {
	case videoResult := <-videoDownloaded: audioResult := <-audioDownloaded
		if videoResult.Error != nil {
			log.Fatal(videoResult.Error)
		}

		if audioResult.Error != nil {
			log.Fatal(audioResult.Error)
		}

		if videoResult.Completed && audioResult.Completed {
			concatFile := concatFiles(videoFilePath, audioFilePath, videoId)

			log.Printf("Video download completed, %s", concatFile)
		}
	}
}

func concatFiles(videoFilePath string, audioFilePath string, videoId string) string {
	resultFile := videoId + ".mp4"
	args := []string{"-y", "-i", videoFilePath, "-i", audioFilePath, resultFile}

	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
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

func logg(a ...interface{}) {
	fmt.Print(a)
}

func DownloadFile(filepath string, url string, ch chan<- DownloadResult) {
	resp, err := http.Get(url)
	if err != nil {
		ch <- DownloadResult{false, err}
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		ch <- DownloadResult{false, err}
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)

	ch <- DownloadResult{true, err}
}
