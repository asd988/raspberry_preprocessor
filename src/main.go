package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"
)

type Item struct {
	Id          string `json:"id"`
	DataVersion int    `json:"data_version"`
}

type outputPaths []string

func (v *outputPaths) String() string {
	return fmt.Sprintf("%v", *v)
}

func (v *outputPaths) Set(value string) error {
	*v = append(*v, value)
	return nil
}

func main() {
	var outputPaths outputPaths
	flag.Var(&outputPaths, "o", "The path to the output directory")

	versionsPath := flag.String("versions", ".rpb_cache/versions.json", "The path to the versions.json")
	ext := flag.String("ext", "txt,glsl,vsh,fsh,json,mcmeta", "extensions that the preprocessor will modify (comma separated)")
	vars := flag.String("vars", "{}", "Json object, with string-string pairs (` can be used for double quotes)")
	currentVersion := flag.String("v", "", "The minecraft version to use for processing")

	isFile := flag.Bool("f", false, "Whether to preprocess file(s) instead of directories")
	usePrecisePaths := flag.Bool("precise-paths", false, "When enabled you have to specify an output path for each input directory (eg. -o path1 -o path2)")

	flag.Parse()
	log.SetFlags(0)

	args := flag.Args()

	if len(outputPaths) == 0 && !*usePrecisePaths {
		outputPaths = append(outputPaths, ".rpb_cache/")
	}

	if *usePrecisePaths && len(args) != len(outputPaths) {
		log.Fatalf("precise paths are enabled so there should be the same amount of output paths as input paths, %d != %d", len(args), len(outputPaths))
	}

	if !*usePrecisePaths && len(outputPaths) != 1 {
		log.Fatal("there is only one output path")
	}

	for _, o := range outputPaths {
		s, err := os.Stat(o)
		if os.IsNotExist(err) {
			os.MkdirAll(o, 0644)
		} else if err != nil {
			log.Fatal(err)
		} else if !s.IsDir() {
			log.Fatal("output path must be directory")
		}
	}

	if *currentVersion == "" {
		log.Fatal("the current version must be provided")
	}

	*vars = strings.ReplaceAll(*vars, "`", "\"")

	// Read versions file
	data, err := os.ReadFile(*versionsPath)
	if err != nil {
		log.Fatal("error opening versions file: ", err)
	}

	var items []Item
	if err := json.Unmarshal([]byte(data), &items); err != nil {
		log.Fatal(err)
	}

	m := make(map[string]int, len(items))
	for _, item := range items {
		m[item.Id] = item.DataVersion
	}

	// Parse variables
	var variables map[string]string
	err = json.Unmarshal([]byte(*vars), &variables)
	if err != nil {
		log.Fatal(err)
	}

	// Process
	start := time.Now()

	exts := strings.Split(*ext, ",")

	g := new(errgroup.Group)
	sem := make(chan struct{}, runtime.NumCPU()*4)
	var failed atomic.Bool

	for i, dir := range args {
		if *isFile {
			panic("not implemented")
		} else {
			var outDir string
			if *usePrecisePaths {
				outDir = outputPaths[i]
			} else {
				base := filepath.Base(dir)
				outDir = filepath.Join(outputPaths[0], base)
			}
			fmt.Println(outDir, *usePrecisePaths)

			err := os.RemoveAll(outDir)
			if err != nil && !os.IsNotExist(err) {
				log.Fatal(err)
			}

			err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
				rel, _ := filepath.Rel(dir, path)
				outputPath := filepath.Join(outDir, rel)

				if d.IsDir() {
					os.MkdirAll(outputPath, 0644)
					return nil
				}

				sem <- struct{}{} // acquire slot
				g.Go(func() error {
					defer func() { <-sem }() // release slot

					if !slices.Contains(exts, filepath.Ext(path)[1:]) {
						err := os.Link(path, outputPath)
						if err != nil {
							log.Fatal(err)
						}
						return nil
					}

					ok := preprocessFile(path, outputPath, variables, m, m[*currentVersion])

					if !ok {
						failed.Store(true)
					}

					return nil
				})

				return nil
			})
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	if err := g.Wait(); err != nil {
		log.Fatal(err)
	}

	if failed.Load() {
		os.Exit(1)
	}

	fmt.Printf("Preprocessing successfully completed in %v\n", time.Since(start))
}
