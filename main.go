package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Download struct {
	Url           string
	TargetPath    string
	TotalSections int
}

func (d Download) Do() error {
	r, err := d.getNewRequest("HEAD")
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return err
	}

	if resp.StatusCode > 299 {
		return errors.New(fmt.Sprintf("Cannot process download: response is %v", resp.StatusCode))
	}

	size, err := strconv.Atoi(resp.Header.Get("Content-Length"))
	if err != nil {
		return err
	}

	var sections = make([][2]int, d.TotalSections)
	eachSectionSize := size / d.TotalSections

	// Algorithm to set start and end positions(sizes) for sections
	for i := range sections {
		if i == 0 {
			sections[i][0] = 0
		} else {
			sections[i][0] = sections[i-1][1] + 1
		}

		if i < d.TotalSections-1 {
			sections[i][1] = sections[i][0] + eachSectionSize
		} else {
			sections[i][1] = size - 1
		}
	}

	var wg sync.WaitGroup

	for i, s := range sections {
		wg.Add(1)
		// when running function below concurrently, loop values might change, therefore we set them to new variables
		i := i
		s := s
		// Perform these concurrently
		go func() {
			defer wg.Done()
			d.downloadSection(i, s)
			if err != nil {
				panic(err)
			}

		}()

	}

	wg.Wait()
	err = d.mergeFiles(sections)
	if err != nil {
		return err
	}

	return nil
}

func (d Download) downloadSection(i int, s [2]int) error {
	r, err := d.getNewRequest("GET")
	if err != nil {
		return err
	}

	r.Header.Set("Range", fmt.Sprintf("bytes=%v-%v", s[0], s[1]))
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return err
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(fmt.Sprintf("section-%v.tmp", i), b, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

func (d Download) getNewRequest(method string) (*http.Request, error) {
	r, err := http.NewRequest(method, d.Url, nil)
	if err != nil {
		return nil, err
	}

	r.Header.Set("User-Agent", "My Download Manager")
	return r, nil
}

func (d Download) mergeFiles(sections [][2]int) error {
	f, err := os.OpenFile(d.TargetPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
	if err != nil {
		return err
	}

	defer f.Close()

	for i := range sections {
		b, err := ioutil.ReadFile(fmt.Sprintf("section-%v.tmp", i))
		if err != nil {
			return err
		}

		n, err := f.Write(b)
		if err != nil {
			return err
		}

		fmt.Printf("Merged %v bytes", n)

	}

	return nil
}

func main() {
	startTime := time.Now()
	url := getInput()
	d := Download{
		Url:           url,
		TargetPath:    "download.mp4",
		TotalSections: 10,
	}

	err := d.Do()

	if err != nil {
		return
	}

	fmt.Printf("Download Completed in %v seconds", time.Now().Sub(startTime).Seconds())
}

func getInput() string {
	fmt.Println("Welcome to My Download Manager")
	fmt.Println("Please Paste Your Video Url Below")

	reader := bufio.NewReader(os.Stdin)

	url, err := reader.ReadString('\n')
	if err != nil {
		return ""
	}

	trimmedUrl := strings.TrimSpace(url)

	return trimmedUrl
}
