package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
)

var lines = flag.Int("lines", 0, "Lines per file")
var skipLines = flag.Int("skipLines", 0, "Lines to skip per file")
var split = flag.Bool("split", false, "Split single file")
var combine = flag.Bool("combine", false, "Combine into single file")
var walkSubDirectories = flag.Bool("subDir", true, "Inlcude sub directories in combine")
var globPattern = flag.String("pattern", "*", "Glob pattern to use")

func usage() {
	fmt.Fprintf(os.Stderr, "usage: combine-spliter [options] path [filename]\n")
	fmt.Fprintf(os.Stderr, "Flags:\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
	}

	name := flag.Arg(0)

	if *split {
		if *lines == 0 {
			fatalF("In split mode lines should be greater than 0")
		}
		splitFile(name, *lines, *skipLines)
		os.Exit(0)
	}

	if *combine {
		if flag.NArg() != 2 {
			fatalF("Argument missing: filename for combined file missing")
		}
		combineFile(name, flag.Arg(1), *lines, *skipLines, *walkSubDirectories, *globPattern)
		os.Exit(0)
	}

	fatalF("Specify mode with -split or -combine")
}

func failOnError(err error, format string, args ...interface{}) {
	if err != nil {
		fatalF(format, args...)
	}
}

func fatalF(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

func splitFile(name string, lines int, skipLines int) {
	dir, fileName := filepath.Split(name)

	file, err := os.Open(name)
	failOnError(err, "Unable to open file: %v", err)
	defer file.Close()

	scanner := bufio.NewScanner(file)
	line := 0
	count := 0
	skipedLines := 0
	var nfile *os.File
	for scanner.Scan() {
		if skipedLines < skipLines {
			scanner.Text()
			skipedLines++
			continue
		}
		if line%lines == 0 {
			if nfile != nil {
				err = nfile.Close()
				failOnError(err, "Unable to close file: %v", err)
			}
			count = line / lines
			nfile, err = os.OpenFile(path.Join(dir, fmt.Sprintf("%d-%s", count, fileName)), os.O_CREATE, 666)
			failOnError(err, "Unable to create file: %v", err)
		}
		_, err := fmt.Fprintln(nfile, scanner.Text())
		failOnError(err, "Unable to write: %v", err)
		line++
	}

	if nfile != nil {
		err = nfile.Sync()
		failOnError(err, "Unable to sync file: %v", err)
		err = nfile.Close()
		failOnError(err, "Unable to close file: %v", err)
	}

	if err := scanner.Err(); err != nil {
		fatalF("Reading error: %v", err)
	}
}

func combineFile(name string, nfile string, lines int, skipLines int, subDir bool, pattern string) {
	line := 0
	var wf *os.File
	filepath.Walk(name, func(fpath string, info os.FileInfo, err error) error {

		if info.IsDir() {
			if !subDir {
				return filepath.SkipDir
			}
			return nil
		}

		ok, err := path.Match(pattern, fpath)
		if err != nil {
			failOnError(err, "error: %v", err)
		}

		if !ok {
			return nil
		}
		skipedLines := 0
		rf, err := os.OpenFile(fpath, os.O_RDONLY, 666)
		failOnError(err, "Unable to read file: %v", err)
		scanner := bufio.NewScanner(rf)
		for scanner.Scan() {
			if skipedLines < skipLines {
				scanner.Text()
				skipedLines++
				continue
			}

			if lines == 0 {
				if wf == nil {
					wf, err = os.OpenFile(path.Join(name, nfile), os.O_CREATE, 666)
					failOnError(err, "Unable to create file: %v", err)
				}
			} else {
				if line%lines == 0 {
					if wf != nil {
						err = wf.Close()
						failOnError(err, "Unable to close file: %v", err)
					}
					count := line / lines
					wf, err = os.OpenFile(path.Join(name, fmt.Sprintf("%d-%s", count, nfile)), os.O_CREATE, 666)
					failOnError(err, "Unable to create file: %v", err)
				}
			}
			_, err := fmt.Fprintln(wf, scanner.Text())
			failOnError(err, "Unable to write: %v", err)
			line++
		}

		if err := scanner.Err(); err != nil {
			fatalF("Reading error: %v", err)
		}
		return nil
	})

}
