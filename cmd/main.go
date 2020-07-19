package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/Static-Flow/ratt/pkg"
	"log"
	"net/url"
	"os"
)

func main() {
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
			reconTarget := pkg.ReconResult{Url: *target}
			reconTarget.StartRecon()
		} else {
			log.Fatalln(err)
		}
	} else {
		//multi target recon
		inputFile, err := os.Open(*targetFilePtr)
		if err != nil {
			fmt.Println(err)
		}
		defer inputFile.Close()
		scanner := bufio.NewScanner(inputFile)
		for scanner.Scan() {
			nextTarget := scanner.Text()
			go func(targetToRecon string) {
				if targetToReconUrl, err := url.Parse(targetToRecon); err == nil {
					reconTarget := pkg.ReconResult{Url: *targetToReconUrl}
					reconTarget.StartRecon()
				}
			}(nextTarget)
		}
	}
}
