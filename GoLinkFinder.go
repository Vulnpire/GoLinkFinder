package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/akamensky/argparse"
	"github.com/tomnomnom/gahttp"
)

const version = "1.1.0"
const concurrency = 10
const regexStr = `(?:"|')(https?://[^\s"']+|//[^\s"']+|/[^\s"']+|[a-zA-Z0-9_\-/]+/[a-zA-Z0-9_\-/]+\.[a-zA-Z]{1,4})(?:"|')`

var founds []string

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

func downloadJSFile(urls []string, concurrency int) {
	pipeLine := gahttp.NewPipeline()
	pipeLine.SetConcurrency(concurrency)
	for _, u := range urls {
		pipeLine.Get(u, gahttp.Wrap(parseFile, gahttp.CloseBody))
	}
	pipeLine.Done()
	pipeLine.Wait()
}

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

func matchAndAdd(content string) []string {
	regExp, err := regexp.Compile(regexStr)
	if err != nil {
		log.Fatal(err)
	}
	links := regExp.FindAllString(content, -1)
	founds = append(founds, links...)
	return founds
}

func appendBaseUrl(urls []string, baseUrl string) []string {
	urls = unique(urls)
	var n []string
	for _, u := range urls {
		n = append(n, baseUrl+strings.TrimSpace(u))
	}
	return n
}

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
	htmlJS := matchAndAdd(doc.Find("script").Text())
	urls = append(urls, extractUrlFromJS(htmlJS, baseUrl)...)

	return appendBaseUrl(unique(urls), baseUrl)
}

func prepareResult(result []string) []string {
	for i := range result {
		result[i] = strings.ReplaceAll(result[i], "\"", "")
		result[i] = strings.ReplaceAll(result[i], "'", "")
	}
	return result
}

func extractScopes(urls []string) []string {
	scopeMap := make(map[string]bool)
	for _, u := range urls {
		parsedUrl, err := url.Parse(u)
		if err == nil {
			parts := strings.Split(parsedUrl.Hostname(), ".")
			if len(parts) > 1 {
				scope := parts[len(parts)-2]
				scopeMap[scope] = true
			}
		}
	}
	var scopes []string
	for scope := range scopeMap {
		scopes = append(scopes, scope)
	}
	return scopes
}

func filterByScope(urls []string, scopes []string) []string {
	var scopedURLs []string
	for _, url := range urls {
		for _, scope := range scopes {
			if strings.Contains(url, scope) {
				scopedURLs = append(scopedURLs, url)
				break
			}
		}
	}
	return scopedURLs
}

func filterCompleteURLs(urls []string) []string {
	var filtered []string
	for _, url := range urls {
		if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
			filtered = append(filtered, url)
		}
	}
	return filtered
}

func processURL(baseUrl string, scopes []string) {
	if !strings.HasPrefix(baseUrl, "http://") && !strings.HasPrefix(baseUrl, "https://") {
		baseUrl = "https://" + baseUrl
	}
	htmlUrls := extractURLsFromHTML(baseUrl)
	downloadJSFile(htmlUrls, concurrency)
	if len(scopes) > 0 {
		founds = filterByScope(founds, scopes)
	}
}

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
	scope := parser.String("s", "scope", &argparse.Options{Help: "Scope for filtering URLs (e.g., example or 'all' for automatic)"})
	completeURLs := parser.Flag("c", "complete", &argparse.Options{Help: "Only output complete URLs starting with http:// or https://"})

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

	var scopes []string
	if *scope == "all" {
		scopes = extractScopes(urls)
	} else if *scope != "" {
		scopes = []string{*scope}
	}

	for _, url := range urls {
		processURL(url, scopes)
	}

	founds = unique(founds)
	founds = prepareResult(founds)

	if *completeURLs {
		founds = filterCompleteURLs(founds)
	}

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
