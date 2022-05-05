package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

var (
	ErrNoFiles  = errors.New("no files found")
	removeFiles = false
)

func main() {
	if len(os.Args) == 1 {
		fmt.Println("Usage: haml2erb [<path>]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	var answer string
	fmt.Fprintln(os.Stdout, "remove converted files? (y/n)")
	fmt.Scanln(&answer)

	if answer == "y" {
		removeFiles = true
	}

	if err := run(os.Args[1], os.Stdout); err != nil {
		log.Fatal(err)
	}
}

func run(root string, out io.Writer) error {
	//list haml files from specified dir
	doneCh := make(chan struct{})
	filesCh := make(chan string)

	files, err := WalkMatch(root, "*.haml")
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return ErrNoFiles
	}

	wg := sync.WaitGroup{}

	go func() {
		defer close(filesCh)
		for _, f := range files {
			filesCh <- f
		}
	}()

	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for f := range filesCh {
				haml, err := os.ReadFile(f)
				if err != nil {
					fmt.Fprintf(out, "reading file %s: %s [ERROR]\n", f, err)
					continue
				}
				fmt.Fprintf(out, "reading file %s [OK]\n", f)

				erb, err := haml2erb(string(haml))

				var unprocessableEntityError *ErrUnprocessableEntity
				if errors.As(err, &unprocessableEntityError) {
					fmt.Fprintf(out, "converting file %s [ERROR]\n", f)
					errFile, err := os.OpenFile("haml2erb-error.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
					if err != nil {
						log.Fatal(err)
					}

					errStr := `
============
%s
%s
============`

					errFile.WriteString(fmt.Sprintf(errStr, f, unprocessableEntityError.Error()))
					errFile.Close()

					continue

				} else if err != nil {
					fmt.Fprintf(out, "converting file %s: %s [ERROR]\n", f, err)
					continue

				}
				fmt.Fprintf(out, "converting file %s [OK]\n", f)

				fname := strings.ReplaceAll(f, ".haml", ".erb")
				if err := os.WriteFile(fname, []byte(erb), 0644); err != nil {
					fmt.Fprintf(out, "writing file %s: %s [ERROR]\n", fname, err)
					continue
				}
				fmt.Fprintf(out, "writing file %s [OK]\n", fname)

				if removeFiles {
					if err := os.Remove(f); err != nil {
						fmt.Fprintf(out, "removing file %s: %s [ERROR]\n", f, err)
						continue
					}
					fmt.Fprintf(out, "removing file %s [OK]\n", f)
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(doneCh)
	}()

	for {
		select {
		case <-doneCh:
			fmt.Fprintln(out, "DONE")
			return nil
		}
	}
}

func WalkMatch(root, pattern string) ([]string, error) {
	var matches []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if matched, err := filepath.Match(pattern, filepath.Base(path)); err != nil {
			return err
		} else if matched {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return matches, nil
}
