package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
)

// remove useless punctuation from a string
func trimPunctuation(stringToTrim string) string {
	punctuation := []string { ",", ":", ";", ".", "!", "\"", "]", "[", "}", "{" }
	for _, p := range punctuation {
		stringToTrim = strings.ReplaceAll(stringToTrim, p, "")
	}
	return stringToTrim
}

// clean a string so it can be parsed
func cleanString(stringToClean string) string {

	stringToClean = strings.TrimSuffix(stringToClean, "\n")
	stringToClean = trimPunctuation(stringToClean)
	stringToClean = strings.TrimSpace(stringToClean)
	stringToClean = strings.ToLower(stringToClean)

	return stringToClean
}

// read the dictionary file and create a map from it
func readDictionary(dictionaryFileName string) map[string]int {

	// create our map which holds the word to count
	var ret map[string]int
	ret = make(map[string]int)

	// open words file or die horribly
	wf, err := os.Open(dictionaryFileName)
	if err != nil {
		log.Fatal(err)
	}

	// created a buffered reader so we can use the neato ReadString method
	wfReader := bufio.NewReader(wf)
	for {
		text, err := wfReader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				// break out of our loop if eof
				break
			}
			// die horribly if anything else happens, leaving a note
			log.Fatalf("Error reading words file [%s] Error [%s]", dictionaryFileName, err)
		}
		text = cleanString(text)
		ret[text] = 0
	}
	return ret
}

// read lines from a channel, clean and delimit them, if they are in a map, send them on to be counted
func countWords(wait * sync.WaitGroup, lineQueue chan string, wordsQueue chan string, wordmap map[string]int) {

	defer wait.Done()
	// loop while the channel is open, getting string from channel (name it item)
	for line := range lineQueue {
		line = cleanString(line)
		tokens := strings.Fields(line)
		for _, token := range(tokens) {
			_, found := wordmap[token]
			if found {
				wordsQueue <- token
			}
		}
	}
}

// count incoming words on a channel and place them in a map, send map when input channel closes
func updateCount(wordsQueue chan string, wordMapQueue chan map[string]int) {

	var ret map[string]int
	ret = make(map[string]int)

	defer close(wordMapQueue)

	for word := range wordsQueue {

		if ret[word] >= 1 {
			ret[word] = ret[word] + 1
		} else {
			ret[word] = 1
		}
	}
	wordMapQueue <- ret
}

// open a file and dump it's contents line by line to a channel
func dumpFileToChannel(fileName string, lineChannel chan string) {

	wf, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}

	// created a buffered reader so we can use the neato ReadString method
	fileReader := bufio.NewReader(wf)

	for {
		line, err := fileReader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				// break out of our loop if eof
				break
			} else {
				log.Fatal(err)
			}
		}

		lineChannel <- line
	}

	close(lineChannel)
}

// search a document using a dictionary file
func search(dictionaryFile string, documentFile string)  (map[string]int) {

	wordsMap := readDictionary(dictionaryFile)
	lineQueue := make(chan string, 1000)
	wordQueue := make(chan string, 1000)

	var wait sync.WaitGroup

	for i:=0; i<runtime.NumCPU(); i++ {
		wait.Add(1)
		go countWords(&wait, lineQueue, wordQueue, wordsMap)
	}

	wordMapQueue := make(chan map[string]int)

	go updateCount(wordQueue, wordMapQueue)

	go dumpFileToChannel(documentFile, lineQueue)

	wait.Wait() // wait for the threads counting the lines to finish

	close(wordQueue) // close the word count queue to the updateCount will finish

	ret := <-wordMapQueue
	return ret
}

func main() {

	var wordsFile = flag.String("dict", "","dictionary of words")
	var searchFile = flag.String("doc", "","document file to search")

	flag.Parse()

	log.Print("dict=", *wordsFile)

	if *wordsFile == "" {
		log.Printf("Missing Required Flag -dict")
		flag.Usage()
		os.Exit(1)
	}

	if *searchFile == "" {
		log.Printf("Missing Required Flag -doc")
		flag.Usage()
		os.Exit(1)
	}

	wm := search(*wordsFile, *searchFile)

	fmt.Print(wm)
}