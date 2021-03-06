package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
)

type Job struct {
	filename string
	results  chan<- Result
}

type Result struct {
	filename string
	lino     int
	line     string
}

func commandLineFiles(files []string) []string {
	if runtime.GOOS == "windows" {
		args := make([]string, 0, len(files))
		for _, name := range files {
			if matches, err := filepath.Glob(name); err != nil {
				args = append(args, name)
			} else if matches != nil {
				args = append(args, matches...)
			}
		}
		return args
	}
	return files
}

var workers = runtime.NumCPU()

func minimum(first int, rest ...int) int {
	for _, x := range rest {
		if x < first {
			first = x
		}

	}
	return first
}
func grep(lineRx *regexp.Regexp, filenames []string) {
	jobs := make(chan Job, workers)
	results := make(chan Result, minimum(1000, len(filenames)))
	done := make(chan struct{}, workers)

	go addJobs(jobs, filenames, results) //execute in own goroutine
	for i := 0; i < workers; i++ {
		go doJobs(done, lineRx, jobs)
	}

	go awaitCompletion(done, results) //executes in its own goroutine
	processResults(results)           //blocks until the work is done
}

func addJobs(jobs chan<- Job, filenames []string, results chan<- Result) {
	for _, filename := range filenames {
		jobs <- Job{filename, results}
	}

	close(jobs)
}

func doJobs(done chan<- struct{}, lineRx *regexp.Regexp, jobs <-chan Job) {
	for job := range jobs {
		job.Do(lineRx)
	}

	done <- struct{}{}
}

func awaitCompletion(done <-chan struct{}, results chan Result) {
	for i := 0; i < workers; i++ {
		<-done
	}

	close(results)
}

func processResults(results <-chan Result) {
	for result := range results {
		fmt.Printf("%s: %d:%s\n", result.filename, result.lino, result.line)
	}
}

func (job Job) Do(lineRx *regexp.Regexp) {
	file, err := os.Open(job.filename)
	if err != nil {
		log.Printf("error: %s\n", err)
		return
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	for lino := 1; ; lino++ {
		line, err := reader.ReadBytes('\n')
		line = bytes.TrimRight(line, "\n\r")
		if lineRx.Match(line) {
			job.results <- Result{job.filename, lino, string(line)}
		}
		if err != nil {
			if err != io.EOF {
				log.Printf("error: %d: %s\n", lino, err)
			}
			break
		}
	}

}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	if len(os.Args) < 3 || os.Args[1] == "-h" || os.Args[1] == "--help" {

		fmt.Printf("Usage: %s <regexp> <files>\n", filepath.Base(os.Args[0]))
		os.Exit(1)

	}
	if lineRx, err := regexp.Compile(os.Args[1]); err != nil {
		log.Fatal("invalid regexp: %s\n", err)
	} else {
		grep(lineRx, commandLineFiles(os.Args[2:]))
	}

}
