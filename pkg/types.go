package pkg

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
	fmt.Println(jsRr.Parent + "js" + string(os.PathSeparator))
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

func (rr *ReconResult) StartRecon() {
	outputPath := rr.getOutputFolder()
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		errDir := os.MkdirAll(outputPath, 0755)
		FatalCheck(errDir)
	}
	rr.fetchResource()
	rr.parseResourceContent()
	rr.saveResults()
}

func (rr *ReconResult) fetchResource() {
	resp, _ := http.Get(rr.Url.String())
	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	rr.Headers = resp.Header
	_ = resp.Body.Close()
	rr.content = string(bodyBytes)
}

func (rr *ReconResult) getOutputFolder() string {
	currentPath, _ := os.Getwd()
	return currentPath + string(os.PathSeparator) + rr.Url.Hostname() + string(os.PathSeparator)
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
	err = ioutil.WriteFile(outputPath+"fullUrls.txt", []byte(strings.Join(rr.urls, "\n")), 0644)
	FatalCheck(err)
	//metadata
	reconMetadata, err := json.Marshal(rr)
	FatalCheck(err)
	err = ioutil.WriteFile(outputPath+"metadata.txt", reconMetadata, 0644)
	FatalCheck(err)
}

func (rr *ReconResult) parseResourceContent() {
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
						jsRr.Rr.fetchResource()
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
						scriptUrl, _ := url.Parse(attr.Val)
						if scriptUrl.IsAbs() {
							rr.urls = append(rr.urls, attr.Val)
						} else {
							rr.paths = append(rr.paths, attr.Val)
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
