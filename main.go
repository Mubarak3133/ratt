package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

func main() {
	targetPtr := flag.String("singletarget", "", "target to recon")
	targetFilePtr := flag.String("targetfile", "", "file containing targets")
	csvFilePtr := flag.String("csvfile", "", "csv file containing targets liken domain,ip,port")
	outputPtr := flag.String("output", "", "directory to store output")
	flag.Parse()

	if len(*targetPtr) > 0 && len(*targetFilePtr) > 0 && len(*csvFilePtr) > 0 {
		log.Fatalln("Please only specify one of singletarget, csvfile, or targetfile")
	} else if len(*targetPtr)+len(*targetFilePtr) == len(*csvFilePtr) {
		log.Fatalln("Please specify singletarget or targetfile or csvfile")
	}
	baseDir, _ := os.Getwd()
	if len(*outputPtr) > 0 {
		baseDir = *outputPtr
		fmt.Println(baseDir)
	}
	var client http.Client
	dialer := net.Dialer{
		Timeout:   time.Duration(30) * time.Second,
		KeepAlive: time.Duration(30) * time.Second,
	}

	defaultTransport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			CipherSuites:       nil,
			MaxVersion:         tls.VersionTLS13,
		},
		DialContext:           dialer.DialContext,
		MaxIdleConns:          100000,
		MaxIdleConnsPerHost:   2,
		IdleConnTimeout:       time.Duration(30) * time.Second,
		ResponseHeaderTimeout: time.Duration(30) * time.Second,
	}
	client = http.Client{
		Transport: defaultTransport,
		Timeout:   time.Duration(30) * time.Second,
	}
	wg := sync.WaitGroup{}

	if len(*targetPtr) > 0 { //single target recon
		if target, err := url.Parse(*targetPtr); err == nil {
			reconTarget := ReconResult{Url: *target, outputBaseDir: baseDir}
			reconTarget.StartRecon(client)
		} else {
			log.Fatalln(err)
		}
	} else if len(*targetFilePtr) > 0 {
		//multi target recon
		inputFile, err := os.Open(*targetFilePtr)
		if err != nil {
			fmt.Println(err)
		}
		defer inputFile.Close()
		scanner := bufio.NewScanner(inputFile)
		for scanner.Scan() {
			nextTarget := scanner.Text()
			fmt.Printf("Scanning %s\n", nextTarget)
			go func(targetToRecon string) {
				wg.Add(1)
				if targetToReconUrl, err := url.Parse(targetToRecon); err == nil {
					reconTarget := ReconResult{Url: *targetToReconUrl, outputBaseDir: baseDir}
					reconTarget.StartRecon(client)
				}
				wg.Done()
			}(nextTarget)
		}
	} else {
		concurrentGoroutines := make(chan struct{}, 1000)

		//csv file targets
		inputFile, err := os.Open(*csvFilePtr)
		if err != nil {
			fmt.Println(err)
		}
		defer inputFile.Close()
		scanner := bufio.NewScanner(inputFile)
		for scanner.Scan() {
			nextTarget := scanner.Text()
			go func(targetToRecon string) {
				concurrentGoroutines <- struct{}{}
				wg.Add(1)
				targetPieces := strings.Split(targetToRecon, ",")
				if len(targetPieces) == 3 {
					if targetToReconUrl, err := url.Parse(fmt.Sprintf("https://%s:%s", targetPieces[1], targetPieces[2])); err == nil {
						fmt.Printf("Scanning %s\n", targetToReconUrl)
						reconTarget := ReconResult{Url: *targetToReconUrl, domain: targetPieces[0], outputBaseDir: baseDir}
						reconTarget.StartRecon(client)
					}
				}
				wg.Done()
				<-concurrentGoroutines
			}(nextTarget)
		}
	}
	wg.Wait()
}
