package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/akamensky/argparse"
	"github.com/tomnomnom/gahttp"
)

const version = "1.0.5"
const concurrency = 10

// Regex pattern to capture URLs from various sources
const regexStr = `(?:"|')(https?://[^\s"']+|//[^\s"']+|/[^\s"']+|[a-zA-Z0-9_\-/]+/[a-zA-Z0-9_\-/]+\.[a-zA-Z]{1,4})(?:"|')`

var founds []string

// Filters URLs to remove duplicates
func unique(strSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range strSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

// Downloads JavaScript files from URLs and parses their content
func downloadJSFile(urls []string, concurrency int) {
	pipeLine := gahttp.NewPipeline()
	pipeLine.SetConcurrency(concurrency)
	for _, u := range urls {
		pipeLine.Get(u, gahttp.Wrap(parseFile, gahttp.CloseBody))
	}
	pipeLine.Done()
	pipeLine.Wait()
}

// Parses content from the JavaScript files and matches URLs
func parseFile(req *http.Request, resp *http.Response, err error) {
	if err != nil {
		return
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	matchAndAdd(string(body))
}

// Extracts JavaScript URLs from HTML
func extractUrlFromJS(urls []string, baseUrl string) []string {
	urls = unique(urls)
	var cleaned []string
	for _, u := range urls {
		u = strings.ReplaceAll(u, "'", "")
		u = strings.ReplaceAll(u, "\"", "")
		if len(u) < 5 {
			continue
		}
		switch {
		case strings.HasPrefix(u, "http"), strings.HasPrefix(u, "https"):
			cleaned = append(cleaned, u)
		case strings.HasPrefix(u, "//"):
			cleaned = append(cleaned, "https:"+u)
		case strings.HasPrefix(u, "/"):
			cleaned = append(cleaned, baseUrl+u)
		}
	}
	return cleaned
}

// Matches URLs using regex and adds them to the list
func matchAndAdd(content string) []string {
	regExp, err := regexp.Compile(regexStr)
	if err != nil {
		log.Fatal(err)
	}
	links := regExp.FindAllString(content, -1)
	founds = append(founds, links...)
	return founds
}

// Appends the base URL to relative paths
func appendBaseUrl(urls []string, baseUrl string) []string {
	urls = unique(urls)
	var n []string
	for _, u := range urls {
		n = append(n, baseUrl+strings.TrimSpace(u))
	}
	return n
}

// Extracts URLs from HTML tags and attributes
func extractURLsFromHTML(baseUrl string) []string {
	resp, err := http.Get(baseUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	// Capture URLs from various HTML tags
	var urls []string
	doc.Find("script").Each(func(i int, s *goquery.Selection) {
		if src, exists := s.Attr("src"); exists {
			urls = append(urls, src)
		}
	})
	doc.Find("a, link, img, iframe").Each(func(i int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists {
			urls = append(urls, href)
		}
		if src, exists := s.Attr("src"); exists {
			urls = append(urls, src)
		}
	})
	// Capture URLs from inline JavaScript
	htmlJS := matchAndAdd(doc.Find("script").Text())
	urls = append(urls, extractUrlFromJS(htmlJS, baseUrl)...)

	return appendBaseUrl(unique(urls), baseUrl)
}

// Prepares the final output by cleaning up the URLs
func prepareResult(result []string) []string {
	for i := range result {
		result[i] = strings.ReplaceAll(result[i], "\"", "")
		result[i] = strings.ReplaceAll(result[i], "'", "")
	}
	return result
}

// Filters URLs based on the specified scope
func filterByScope(urls []string, scope string) []string {
	var scopedURLs []string
	for _, url := range urls {
		if strings.Contains(url, scope) {
			scopedURLs = append(scopedURLs, url)
		}
	}
	return scopedURLs
}

// Processes each URL by extracting and downloading JavaScript files
func processURL(baseUrl, scope string) {
	if !strings.HasPrefix(baseUrl, "http://") && !strings.HasPrefix(baseUrl, "https://") {
		baseUrl = "https://" + baseUrl
	}
	htmlUrls := extractURLsFromHTML(baseUrl)
	downloadJSFile(htmlUrls, concurrency)
	if scope != "" {
		founds = filterByScope(founds, scope)
	}
}

// Reads URLs from a specified file
func readURLsFromFile(filename string) []string {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Could not open file: %v\n", err)
	}
	defer file.Close()

	var urls []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		url := strings.TrimSpace(scanner.Text())
		if url != "" {
			urls = append(urls, url)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading file: %v\n", err)
	}
	return urls
}

func main() {
	parser := argparse.NewParser("goLinkFinder", "GoLinkFinder")
	domain := parser.String("d", "domain", &argparse.Options{Help: "Input a URL."})
	output := parser.String("o", "out", &argparse.Options{Help: "File name :  (e.g : output.txt)"})
	fileInput := parser.String("f", "file", &argparse.Options{Help: "Input file with URLs"})
	scope := parser.String("s", "scope", &argparse.Options{Help: "Scope for filtering URLs (e.g., example)"})

	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
		return
	}

	var urls []string
	if *fileInput != "" {
		urls = readURLsFromFile(*fileInput)
	} else if *domain != "" {
		urls = []string{*domain}
	} else {
		fmt.Println("Please provide a domain (-d) or a file (-f) with URLs.")
		return
	}

	for _, url := range urls {
		processURL(url, *scope)
	}

	founds = unique(founds)
	founds = prepareResult(founds)
	for _, found := range founds {
		fmt.Println(found)
	}

	if *output != "" {
		f, err := os.OpenFile("./"+*output, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Println(err)
		}
		defer f.Close()
		for _, found := range founds {
			if _, err := f.WriteString(found + "\n"); err != nil {
				log.Fatal(err)
			}
		}
	}
}
