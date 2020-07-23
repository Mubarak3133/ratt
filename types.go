package main

import (
	"encoding/json"
	"fmt"
	"github.com/gocolly/colly"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const jsFolder = "js" + string(os.PathSeparator)

type CookiesToAdd struct {
	Cookies []http.Cookie
}

type ReconResult struct {
	Url           url.URL      //What resource are we getting
	outputBaseDir string       //Either os.getPwd() or the output directory flag value if set
	domain        string       //internally used for if you wanted to hit a target by IP but a host header with a domain
	Title         string       //title of page (may be null in case of JS content)
	Headers       http.Header  //headers from calling resource
	cookies       CookiesToAdd //cookies to add to the requests
	depth         int          //depth to crawl target
}

func (rr *ReconResult) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Url     string
		Title   string
		Headers http.Header
	}{
		Url:     rr.Url.String(),
		Title:   rr.Title,
		Headers: rr.Headers,
	})
}

func (rr *ReconResult) StartRecon(client http.Client) {
	outputPath := rr.getOutputFolder()
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		errDir := os.MkdirAll(outputPath, 0755)
		FatalCheck(errDir)
	}
	if _, err := os.Stat(outputPath + jsFolder); os.IsNotExist(err) {
		errDir := os.MkdirAll(outputPath+jsFolder, 0755)
		FatalCheck(errDir)
	}
	rr.reconIt(client)
}

func (rr *ReconResult) reconIt(client http.Client) {

	// Instantiate default collector
	c := colly.NewCollector(
		// MaxDepth is 2, so only the links on the scraped page
		// and links on those pages are visited
		colly.MaxDepth(rr.depth),
		colly.Async(),
	)
	c.SetClient(&client)
	jar, err := cookiejar.New(nil)
	FatalCheck(err)
	c.SetCookieJar(jar)
	c.ParseHTTPErrorResponse = true

	// Limit the maximum parallelism to 2
	// This is necessary if the goroutines are dynamically
	// created to control the limit of simultaneous requests.
	//
	// Parallelism can be controlled also by spawning fixed
	// number of go routines.
	c.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 5})

	c.OnHTML("script", func(element *colly.HTMLElement) {
		javascript := element.Attr("src")
		if len(javascript) != 0 {
			//fmt.Println("Found JS: "+javascript)
			element.Request.Visit(javascript)
		}
	})

	c.OnHTML("title", func(element *colly.HTMLElement) {
		rr.Title = element.Text
	})

	c.OnRequest(func(request *colly.Request) {
		for _, cookieToAdd := range rr.cookies.Cookies {
			err := c.SetCookies(request.URL.String(),
				[]*http.Cookie{{Name: cookieToAdd.Name,
					Value: cookieToAdd.Value}})
			FatalCheck(err)
		}
		fmt.Println("Visiting: " + request.URL.String())
	})

	c.OnScraped(func(response *colly.Response) {
		rr.saveResults(response)
	})

	c.OnHTML("form[action]", func(e *colly.HTMLElement) {
		link := e.Attr("action")
		// Print link

		foundUrl, err := url.Parse(link)
		if err == nil {
			if !foundUrl.IsAbs() {
				//fmt.Println(link)
				// Visit link found on page on a new thread
				e.Request.Visit(link)
			}
		} else {
			fmt.Println(err)
		}

	})

	// On every a element which has href attribute call callback
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		link = strings.TrimSpace(link)
		// Print link

		foundUrl, err := url.Parse(link)
		if err == nil {
			if !foundUrl.IsAbs() {
				//fmt.Println(link)
				// Visit link found on page on a new thread
				e.Request.Visit(link)
			}
		} else {
			fmt.Println(err)
		}

	})

	c.Visit(rr.Url.String())
	// Wait until threads are finished
	c.Wait()
}

func (rr *ReconResult) getOutputFolder() string {
	outputFolder := rr.outputBaseDir
	if len(rr.domain) > 0 {
		outputFolder = outputFolder + string(os.PathSeparator) + rr.domain + string(os.PathSeparator)
	} else {
		outputFolder = outputFolder + string(os.PathSeparator) + rr.Url.Hostname() + string(os.PathSeparator)
	}
	if len(rr.Url.RawQuery) > 0 {
		outputFolder = outputFolder + string(os.PathSeparator) + rr.Url.RawQuery + string(os.PathSeparator)
	}
	return outputFolder
}

func (rr *ReconResult) saveResults(response *colly.Response) {
	if strings.HasSuffix(response.Request.URL.Path, ".js") {
		outputPath := rr.getOutputFolder() + jsFolder + path.Base(response.Request.URL.Path)
		err := response.Save(outputPath)
		FatalCheck(err)
	} else {
		outputPath := filepath.FromSlash(rr.getOutputFolder() + response.Request.URL.Path)
		outputPath = strings.ReplaceAll(outputPath, string(os.PathSeparator)+string(os.PathSeparator), string(os.PathSeparator))
		fmt.Println(outputPath)
		if strings.HasSuffix(outputPath, string(os.PathSeparator)) {
			outputPath = outputPath + "content"
		}
		//var err error
		err := response.Save(outputPath + ".html")
		if err != nil {
			//if filepath.Clean(filepath.FromSlash(response.Request.URL.Path))
			//directory doesn't exist yet, lop off all but the base, make that dir then add the base as a file
			fmt.Println("New Directory: " + outputPath[:strings.LastIndex(outputPath, string(os.PathSeparator))])
			err = os.MkdirAll(outputPath[:strings.LastIndex(outputPath, string(os.PathSeparator))], 0755)
			FatalCheck(err)
			err := response.Save(outputPath + ".html")
			if err != nil {
				if !strings.Contains(err.Error(), "is a directory") {
					//err := response.Save(outputPath + string(os.PathSeparator) + "content.html")
					FatalCheck(err)
				}
			}

		}
		//metadata
		rr.Headers = *response.Headers
		reconMetadata, err := json.Marshal(rr)
		FatalCheck(err)
		err = ioutil.WriteFile(outputPath[:strings.LastIndex(outputPath, string(os.PathSeparator))+1]+"metadata.json", reconMetadata, 0644)
		FatalCheck(err)
	}

}
