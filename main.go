package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
)

var logFile *os.File

func init() {
	logFile, err := os.OpenFile("/var/log/shear.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	checkErr(err, true)

	log.SetOutput(logFile)
}

func main() {
	whitelist := getWhitelist("/etc/shear/whitelist.txt")

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	checkErr(err, true)

	imageSummaries, err := cli.ImageList(context.Background(), types.ImageListOptions{})
	checkErr(err, true)

	for _, imageSummary := range imageSummaries {
		imageName := getImageName(imageSummary)
		if !whitelist[imageName] {
			response, err := cli.ImageRemove(context.Background(), imageSummary.ID, types.ImageRemoveOptions{Force: true, PruneChildren: true})
			checkErr(err, false)

			if len(response) > 0 {
				logImageDeleteResponse(response)
			}
		}
	}

	logFile.Close()
}

func getWhitelist(filepath string) map[string]bool {
	var whitelist = make(map[string]bool)

	file, err := os.Open(filepath)
	checkErr(err, true)

	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		whitelist[scanner.Text()] = true
	}

	return whitelist
}

func getImageName(imageSummary types.ImageSummary) (imageName string) {
	var imageRepoTag string

	if len(imageSummary.RepoTags) > 0 {
		imageRepoTag = imageSummary.RepoTags[0]
		imageName = strings.Split(imageRepoTag, ":")[0]
	} else if len(imageSummary.RepoDigests) > 0 {
		imageRepoTag = imageSummary.RepoDigests[0]
		imageName = strings.Split(imageRepoTag, "@")[0]
	} else {
		err := fmt.Errorf("cannot parse image %+v", imageSummary)
		checkErr(err, true)
	}

	return
}

func logImageDeleteResponse(response []types.ImageDeleteResponseItem) {
	for _, responseItem := range response {
		if responseItem.Deleted != "" {
			log.WithFields(log.Fields{
				"id": responseItem.Deleted,
			}).Info("image deleted")
		} else {
			log.WithFields(log.Fields{
				"id": responseItem.Untagged,
			}).Debug("image untagged")
		}
	}
}

func checkErr(err error, fatal bool) {
	if err != nil {
		if fatal {
			log.Fatal(err)
		} else {
			log.Error(err)
		}
	}
}
