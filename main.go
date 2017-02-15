package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	var basedir string
	flag.StringVar(&basedir, "basedir", "./input", "base directory")
	var destdir string
	flag.StringVar(&destdir, "destdir", "./output", "dest directory")
	flag.Parse()

	os.Exit(run(basedir, destdir))
}

func run(basedir, destdir string) int {
	walkFn := func(path string, info os.FileInfo, err error) error {
		if !info.Mode().IsRegular() {
			return nil
		}
		fmt.Printf("path: %s\n", path)
		err = extractCacheBody(path, destdir)
		if err != nil {
			log.Printf("failed to extract cache body: err=%+v", err)
			return err
		}
		return nil
	}
	err := filepath.Walk(basedir, walkFn)
	if err != nil {
		log.Printf("failed to walk: err=%+v", err)
		return 1
	}
	return 0
}

const headerSize = 0x90
const keyMarker = "KEY: "

func extractCacheBody(path, destdir string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file: err=%+v", err)
	}
	defer file.Close()

	n, err := file.Seek(headerSize, io.SeekStart)
	if err != nil || n != headerSize {
		return fmt.Errorf("failed to skip header: err=%+v", err)
	}

	key := ""
	pos := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		//fmt.Printf("pos=%d, line=%s\n", pos, line)
		if strings.HasPrefix(line, keyMarker) {
			key = line[len(keyMarker):]
			fmt.Printf("key: %s\n", key)
		}
		if pos > 0 && line == "" {
			pos += len(line)
			break
		}
		pos += len(line) + 2 // CR LF
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}
	// fmt.Printf("pos: %d\n", pos)

	bodyStart := int64(headerSize + pos)
	n, err = file.Seek(bodyStart, io.SeekStart)
	if err != nil || n != bodyStart {
		return fmt.Errorf("failed to skip http header: err=%+v", err)
	}

	var outPath string
	if strings.HasSuffix(key, "/") {
		outPath = filepath.Join(destdir, key+"__index.html")
	} else {
		outPath = filepath.Join(destdir, key)
	}
	//log.Printf("outPath=%s", outPath)
	outDir := filepath.Dir(outPath)
	err = os.MkdirAll(outDir, 0777)
	if err != nil {
		return fmt.Errorf("failed to create dest dir: dir=%s, err=%+v", outDir, err)
	}

	outFile, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: file=%s, err=%+v", outPath, err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, file)
	if err != nil {
		return fmt.Errorf("failed to copy request body to output file: file=%s, err=%+v", outPath, err)
	}

	fmt.Printf("written: %s\n", outPath)
	return nil
}
