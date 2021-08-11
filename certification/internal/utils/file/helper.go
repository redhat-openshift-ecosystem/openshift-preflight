package file

import (
	"compress/bzip2"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func DownloadFile(filename string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

func Unzip(bzipfile string, destination string) error {

	f, err := os.Open(bzipfile)
	if err != nil {
		return err
	}
	defer f.Close()

	in := bzip2.NewReader(f)

	out, err := os.Create(destination)

	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)

	if err != nil {
		return err
	}
	out.Close()
	return nil
}

func createArtifactsDir(artifactsDir string) (string, error) {
	if !strings.HasPrefix(artifactsDir, "/") {
		currentDir, err := os.Getwd()
		if err != nil {
			log.Error("unable to get current directory: ", err)
			return "", err
		}

		artifactsDir = filepath.Join(currentDir, artifactsDir)
	}

	err := os.MkdirAll(artifactsDir, 0777)
	if err != nil {
		log.Error("unable to create artifactsDir: ", err)
		return "", err
	}
	return artifactsDir, nil
}

func ArtifactPath(artifact string) string {
	artifactDir := viper.GetString("artifacts")

	artifactDir, err := createArtifactsDir(artifactDir)
	if err != nil {
		log.Fatal("could not retrieve artifact path")
		// Fatal does an os.Exit
	}
	return filepath.Join(artifactDir, artifact)
}
