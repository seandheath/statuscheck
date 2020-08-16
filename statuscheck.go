package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/jessevdk/go-flags"
)

var opts = struct {
	Timeout time.Duration `short:"t" long:"timeout" description:"request timeout in seconds"`
	HTTPS   bool          `short:"s" description:"add https:// to urls"`
	Outfile string        `short:"o" long:"outfile" description:"file to write output to"`
	Path    string        `short:"p" long:"path" description:"path to append to each URL"`

	Args struct {
		INFILE string
	} `positional-args:"yes" required:"yes"`
}{
	Timeout: time.Duration(10) * time.Second,
	HTTPS:   false,
	Outfile: "",
	Path:    "",
}

func check(e error) {
	if e != nil {
		if flags.WroteHelp(e) {
			os.Exit(1)
		} else {
			panic(e)
		}
	}
}

func processURL(url string) []string {
	baseString := strings.TrimPrefix(url, "http://")
	baseString = strings.TrimPrefix(baseString, "https://")

	var urls []string

	// Add http://
	if baseString != "" {
		urls = append(urls, "http://"+url+opts.Path)
		// Add https://
		if opts.HTTPS {
			urls = append(urls, "https://"+url+opts.Path)
		}
	}

	return urls
}

func checkURL(url string, ch chan<- [2]string) {
	c := &http.Client{
		Timeout: opts.Timeout,
	}
	resp, err := c.Get(url)
	status := ""
	if err != nil {
		status = "error"
	} else {
		status = resp.Status
	}
	ch <- [2]string{status, url}
	return
}

func main() {
	_, err := flags.Parse(&opts)
	check(err)

	ifile, err := os.Open(opts.Args.INFILE)
	check(err)
	defer ifile.Close()

	var allURLs []string
	originalURLCount := 0

	scanner := bufio.NewScanner(ifile)
	for scanner.Scan() {
		url := scanner.Text()
		urls := processURL(url)
		allURLs = append(allURLs, urls...)
		originalURLCount++
	}
	check(scanner.Err())

	newURLCount := len(allURLs)
	if opts.Path != "" {
		fmt.Println("Appending path:", opts.Path)
	}
	fmt.Println("Generated", newURLCount, "URLs from original list of", originalURLCount)
	fmt.Println("Checking all URLs for status codes:")

	ch := make(chan [2]string)
	responses := make(map[string][]string)
	threads := 0

	// Start goroutines to check all URLs
	for _, url := range allURLs {
		go checkURL(url, ch)
		threads++
	}

	// Catch responses for all URLs
	bar := pb.StartNew(threads)
	for i := 0; i < threads; i++ {
		code := <-ch
		if code[0] != "error" {
			responses[code[0]] = append(responses[code[0]], code[1])
		}
		bar.Increment()
	}
	bar.Finish()

	output := os.Stdout
	if opts.Outfile != "" {
		ofile, err := os.Create(opts.Outfile)
		check(err)
		defer ofile.Close()
		output = ofile

	} else {
		fmt.Println("\n\nResponses:")
	}
	for status, urls := range responses {
		s := fmt.Sprintf("Status Code: %s\n\n", status)
		for _, url := range urls {
			s += url + "\n"
		}
		s += "\n"
		_, err := output.WriteString(s)
		check(err)
	}
	return
}
