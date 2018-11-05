// Licensed under the  MIT License (MIT)
// Copyright (c) 2016 Peter Waller <p@pwaller.net>

package client

import (
	"archive/tar"
	"bufio"
	"fmt"
	"io"

	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/radityasurya/docker-show-context/pkg/docker"

	"github.com/spf13/viper"
)

// WriteCounter counts the bytes written to it.
type WriteCounter int

func (w *WriteCounter) Write(bs []byte) (int, error) {
	*w += WriteCounter(len(bs))
	return len(bs), nil
}

// Run the show context
func Run() {
	// Take a quick and dirty file count. This should be an over-estimate,
	// since it doesn't currently attempt to re-implement or reuse the
	// dockerignore logic.
	totalCount := 0
	totalStorage := int64(0)
	filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		totalCount++
		totalStorage += info.Size()
		return nil
	})

	r, err := docker.GetArchive(".", viper.GetString("dockerfile"))
	if err != nil {
		log.Fatalf("Failed to make context: %v", err)
	}
	defer r.Close()

	// Keep mappings of paths/extensions to bytes/counts/times.
	dirStorage := map[string]int64{}
	dirFiles := map[string]int64{}
	dirTime := map[string]int64{}
	extStorage := map[string]int64{}
	filesList := map[string]int64{}

	// Counts of amounts seen so far.
	currentCount := 0
	currentStorage := int64(0)

	// Update the progress indicator at some frequency.
	const updateFrequency = 50 // Hz
	ticker := time.NewTicker(time.Second / updateFrequency)
	defer ticker.Stop()
	tick := ticker.C

	start := time.Now()
	last := time.Now()

	// Keep a count of how many bytes of Tar file were seen.
	writeCounter := WriteCounter(0)
	tf := tar.NewReader(io.TeeReader(r, &writeCounter))

	cr := []byte("\r")
	showUpdate := func(w io.Writer) {
		os.Stderr.Write(cr) // always to Stderr.

		fmt.Fprintf(w,
			"  %v / %v (%.0f / %.0f MiB) "+
				"(%.1fs elapsed)",
			currentCount,
			totalCount,
			float64(currentStorage)/1024/1024,
			float64(totalStorage)/1024/1024,
			time.Since(start).Seconds(),
		)
	}

	fmt.Println()
	fmt.Println("Scanning local directory (in tar / on disk):")
entries:
	for {
		header, err := tf.Next()
		switch err {
		case io.EOF:
			showUpdate(os.Stdout)
			fmt.Println(" .. completed")
			fmt.Println()
			break entries
		default:
			log.Fatalf("Error reading archive: %v", err)
			return
		case nil:
		}

		duration := time.Since(last).Nanoseconds()
		last = time.Now()

		dir := filepath.Dir(header.Name)
		filename := header.Name
		size := header.FileInfo().Size()

		currentCount++
		currentStorage += size

		dirStorage[dir] += size
		dirTime[dir] += duration
		dirFiles[dir]++

		if !header.FileInfo().IsDir() {
			ext := filepath.Ext(strings.ToLower(header.Name))
			extStorage[ext] += size
			filesList[filename] += size
		}

		select {
		case <-tick:
			showUpdate(os.Stderr)
		default:
		}
	}

	fmt.Printf(
		"Excluded by .dockerignore: "+
			"%d files totalling %.2f MiB\n",
		totalCount-currentCount,
		float64(totalStorage-currentStorage)/1024/1024)
	fmt.Println()
	fmt.Println("Final .tar:")
	// Epilogue.
	fmt.Printf(
		"  %v files totalling %.2f MiB (+ %.2f MiB tar overhead)\n",
		currentCount,
		float64(currentStorage)/1024/1024,
		float64(int64(writeCounter)-currentStorage)/1024/1024,
	)
	fmt.Printf("  Took %.2f seconds to build\n", time.Since(start).Seconds())
	fmt.Println()

	// Produce Top-N.
	topDirStorage := SizeAscending(dirStorage)
	topDirFiles := SizeAscending(dirFiles)
	topDirTime := SizeAscending(dirTime)
	topExtStorage := SizeAscending(extStorage)
	topFilesList := SizeAscending(filesList)

	N := viper.GetInt("files-number")
	fmt.Printf("Top %d directories by time spent:\n", N)
	for i := 0; i < N && i < len(topDirTime); i++ {
		entry := &topDirTime[i]
		fmt.Printf("%5d ms: %v\n", entry.Size/1000/1000, entry.Path)
	}
	fmt.Println()

	fmt.Printf("Top %d directories by storage:\n", N)
	for i := 0; i < N && i < len(topDirStorage); i++ {
		entry := &topDirStorage[i]
		fmt.Printf("%7.2f MiB: %v\n", float64(entry.Size)/1024/1024, entry.Path)
	}
	fmt.Println()

	fmt.Printf("Top %d directories by file count:\n", N)
	for i := 0; i < N && i < len(topDirFiles); i++ {
		entry := &topDirFiles[i]
		fmt.Printf("%5d: %v\n", entry.Size, entry.Path)
	}
	fmt.Println()

	fmt.Printf("Top %d file extensions by storage:\n", N)
	for i := 0; i < N && i < len(topExtStorage); i++ {
		entry := &topExtStorage[i]
		fmt.Printf("%7.2f MiB: %v\n", float64(entry.Size)/1024/1024, entry.Path)
	}
	fmt.Println()

	fmt.Printf("Top %d files by size:\n", N)
	for i := 0; i < N && i < len(topFilesList); i++ {
		entry := &topFilesList[i]
		fmt.Printf("%7.2f MiB: %v\n", float64(entry.Size)/1024/1024, entry.Path)
	}

	output := viper.GetString("output")
	if output != "" {
		f, err := os.Create(output)
		if err != nil {
			log.Error(err)
		}

		defer f.Close()

		w := bufio.NewWriter(f)
		_, err = w.WriteString("List of files by size:\n")
		if err != nil {
			log.Error(err)
		}

		for i := 0; i < len(topFilesList); i++ {
			entry := &topFilesList[i]
			line := fmt.Sprintf("%7.2f MiB: %v\n", float64(entry.Size)/1024/1024, entry.Path)
			_, err = w.WriteString(line)
			if err != nil {
				log.Error(err)
			}
		}

		err = w.Flush()
		if err != nil {
			log.Error(err)
		}

		fmt.Println()
		fmt.Printf("\nThe complete logs could be found here: %s \n", output)
	}
}
