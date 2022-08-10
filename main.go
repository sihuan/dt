package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: dt filename thread")
		os.Exit(1)
	}

	fp, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Println(err)
		return
	}

	if os.MkdirAll("dt_pximg", 0755) != nil {
		fmt.Println("Error creating dt_pximg directory")
		return
	}

	filename := os.Args[1]
	var extension = filepath.Ext(filename)
	var dt_path = filename[0 : len(filename)-len(extension)]
	fmt.Println("Downloading " + dt_path + "...")
	if os.MkdirAll(dt_path, 0755) != nil {
		fmt.Println("Error creating dt_path directory")
		return
	}

	buf := bufio.NewScanner(fp)

	type ur struct {
		url string
		r   int
	}

	urlChan := make(chan ur, 100)
	var wg sync.WaitGroup
	var ops uint64
	var d uint64
	thread, err := strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Println(err)
		return
	}

	go func() {
		i := 0
		for {
			if !buf.Scan() {
				break
			}
			line := buf.Text()
			urls := strings.Split(line, ",")
			for _, url := range urls {
				url = strings.Replace(url, "i.pximg.net", "210.140.92.145", -1)
				i += 1
				urlChan <- ur{url: url, r: i}
				break
			}
			// urlChan <- line
		}
		close(urlChan)
	}()

	for i := 0; i < thread; i++ {
		wg.Add(1)
		time.Sleep(time.Second * 2)
		go func() {
			tr := &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			client := &http.Client{Transport: tr}
			for urlr := range urlChan {
				url := urlr.url
				_, file := filepath.Split(url)
				_, err := os.Stat("dt_pximg/" + file)
				if err != nil {
					if os.IsNotExist(err) {
						downloader(client, url, "dt_pximg/"+file)
						atomic.AddUint64(&d, 1)
						time.Sleep(time.Millisecond * 100)
					} else {
						panic(err)
					}
				}
				var extension = filepath.Ext(file)
				var symLink = dt_path + "/" + fmt.Sprintf("%03d", urlr.r) + extension
				os.Symlink("../dt_pximg/"+file, symLink)
				atomic.AddUint64(&ops, 1)
				fmt.Print(atomic.LoadUint64(&ops), "Download: ", atomic.LoadUint64(&d), "\r")
			}
			wg.Done()
		}()
	}

	wg.Wait()

}

func downloader(client *http.Client, url string, file string) {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Referer", "https://app-api.pixiv.net/")
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Println(url, ":", resp.StatusCode)
		return
	}

	out, err := os.Create(file)
	if err != nil {
		panic(err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		panic(err)
	}
	// fmt.Println(url)
}
