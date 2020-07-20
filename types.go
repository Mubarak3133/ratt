package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"golang.org/x/net/html"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
)

type JSReconResult struct {
	Rr     ReconResult //Standard Recon Result data
	Parent string      //This is the root domain where we found this JS resource
}

func (jsRr *JSReconResult) parseJavascript() {

}

func (jsRr *JSReconResult) getOutputFolder() string {
	return jsRr.Parent + "js" + string(os.PathSeparator)
}

func (jsRr *JSReconResult) saveResults(inline bool) {
	outputPath := jsRr.getOutputFolder()
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		errDir := os.MkdirAll(outputPath, 0755)
		FatalCheck(errDir)
	}
	jsContent := []byte(jsRr.Rr.content)
	if inline {
		err := ioutil.WriteFile(outputPath+CreateInlineJSFileName(), jsContent, 0644)
		FatalCheck(err)
	} else {
		err := ioutil.WriteFile(outputPath+path.Base(jsRr.Rr.Url.Path), jsContent, 0644)
		FatalCheck(err)
	}

}

type ReconResult struct {
	Url     url.URL     //What resource are we getting
	domain  string      //internally used for if you wanted to hit a target by IP but a host header with a domain
	content string      //raw content of the page
	Title   string      //title of page (may be null in case of JS content)
	Headers http.Header //headers from calling resource
	paths   []string    //relative paths found within this resource
	urls    []string    //absolute urls found within this resource
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
	rr.fetchResource(client)
	if len(rr.content) != 0 {
		rr.parseResourceContent(client)
		rr.saveResults()
	}
}

func (rr *ReconResult) fetchResource(client http.Client) {
	var resp *http.Response
	var req *http.Request
	var err error
	req, err = http.NewRequest("GET", rr.Url.String(), nil)
	if err == nil {
		if len(rr.domain) > 0 {
			req.Host = rr.domain
		}
		req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.106 Safari/537.36")
		resp, err = client.Do(req)
		if err == nil {
			bodyBytes, _ := ioutil.ReadAll(resp.Body)
			rr.Headers = resp.Header
			_ = resp.Body.Close()
			rr.content = string(bodyBytes)
		} else {
			if strings.Contains(err.Error(), "IP SANs") {
				newUrl, err := url.Parse(fmt.Sprintf("%s://%s:%s", rr.Url.Scheme, rr.domain, rr.Url.Port()))
				if err == nil {
					fmt.Println("Trying new URL: " + newUrl.String())
					rr.Url = *newUrl
					rr.fetchResource(client)
				}
			} else {
				//fmt.Println(err)
			}
		}
	}
}

func (rr *ReconResult) getOutputFolder() string {
	currentPath, _ := os.Getwd()

	if len(rr.domain) > 0 {
		currentPath = currentPath + string(os.PathSeparator) + rr.domain + string(os.PathSeparator) + rr.Url.EscapedPath() + string(os.PathSeparator)
	} else {
		currentPath = currentPath + string(os.PathSeparator) + rr.Url.Hostname() + string(os.PathSeparator) + rr.Url.EscapedPath() + string(os.PathSeparator)
	}
	if len(rr.Url.RawQuery) > 0 {
		currentPath = currentPath + string(os.PathSeparator) + rr.Url.RawQuery + string(os.PathSeparator)
	}
	return currentPath
}

func (rr *ReconResult) saveResults() {
	outputPath := rr.getOutputFolder()
	//html content
	htmlContent := []byte(rr.content)
	err := ioutil.WriteFile(outputPath+"content.html", htmlContent, 0644)
	FatalCheck(err)
	//relative paths
	err = ioutil.WriteFile(outputPath+"relativePaths.txt", []byte(strings.Join(rr.paths, "\n")), 0644)
	FatalCheck(err)
	//absolute urls
	err = ioutil.WriteFile(outputPath+"absolutePaths.txt", []byte(strings.Join(rr.urls, "\n")), 0644)
	FatalCheck(err)
	//metadata
	reconMetadata, err := json.Marshal(rr)
	FatalCheck(err)
	err = ioutil.WriteFile(outputPath+"metadata.txt", reconMetadata, 0644)
	FatalCheck(err)
}

func (rr *ReconResult) parseResourceContent(client http.Client) {
	z := html.NewTokenizer(bytes.NewReader([]byte(rr.content)))

	for {
		tt := z.Next()
		switch {
		case tt == html.ErrorToken:
			// End of the document, we're done
			return
		case tt == html.StartTagToken:
			t := z.Token()
			switch {
			case t.Data == "script":
				inline := true
				var jsRr JSReconResult
				jsRr.Parent = rr.getOutputFolder()
				for _, attr := range t.Attr {
					if attr.Key == "src" {
						inline = false
						scriptUrl, _ := url.Parse(attr.Val)
						rr.Url.Hostname()
						if scriptUrl.IsAbs() {
							jsRr.Rr.Url = *scriptUrl
						} else {
							if strings.HasPrefix(attr.Val, "../") {
								scriptUrl, _ = url.Parse(attr.Val[3:])
							}
							jsRr.Rr.Url = url.URL{Scheme: rr.Url.Scheme, Host: rr.Url.Host, Path: scriptUrl.Path,
								RawPath: scriptUrl.RawPath, RawQuery: scriptUrl.RawQuery}
						}
						jsRr.Rr.fetchResource(client)
						break
					}
				}
				if inline {
					innerToken := z.Next()
					//just make sure it's actually a text token
					if innerToken == html.TextToken {
						//report the page title and break out of the loop
						jsRr.Rr.content = z.Token().Data
					}
				}
				jsRr.parseJavascript()
				jsRr.saveResults(inline)

			case t.Data == "a":
				for _, attr := range t.Attr {
					if attr.Key == "href" {
						scriptUrl, err := url.Parse(attr.Val)
						if err == nil {
							if scriptUrl.IsAbs() {
								rr.urls = append(rr.urls, attr.Val)
							} else {
								rr.paths = append(rr.paths, attr.Val)
							}
						}
					}
				}
			case t.Data == "title":
				innerToken := z.Next()
				//just make sure it's actually a text token
				if innerToken == html.TextToken {
					//report the page title and break out of the loop
					rr.Title = z.Token().Data
				}
			}
		}
	}
}
