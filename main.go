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
	"sync"
	"time"
)

func main() {
	var client http.Client
	wg := sync.WaitGroup{}
	targetPtr := flag.String("singletarget", "", "target to recon")
	targetFilePtr := flag.String("targetfile", "", "file containing targets")
	flag.Parse()
	if len(*targetPtr) > 0 && len(*targetFilePtr) > 0 {
		log.Fatalln("Please on specify singletarget or targetfile")
	} else if len(*targetPtr) == len(*targetFilePtr) {
		log.Fatalln("Please specify singletarget or targetfile")
	}
	if len(*targetPtr) > 0 { //single target recon
		if target, err := url.Parse(*targetPtr); err == nil {
			reconTarget := ReconResult{Url: *target}
			reconTarget.StartRecon(client)
		} else {
			log.Fatalln(err)
		}
	} else {
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
					reconTarget := ReconResult{Url: *targetToReconUrl}
					reconTarget.StartRecon(client)
				}
				wg.Done()
			}(nextTarget)
		}
	}
	wg.Wait()
}
