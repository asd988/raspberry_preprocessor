package main

import (
	"encoding/json"
	"flag"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type Item struct {
	Id          string `json:"id"`
	DataVersion int    `json:"data_version"`
}

func main() {
	versionsPath := flag.String("versions", ".rpb_cache/versions.json", "The path to the versions.json")
	ext := flag.String("ext", "txt,glsl,vsh,fsh,json", "extensions that the preprocessor will modify (comma separated)")
	isFile := flag.Bool("f", false, "to preprocess file(s) instead of directories")
	vars := flag.String("vars", "{}", "json object, with string-string pairs (` can be used for double quotes)")
	outputPath := flag.String("o", ".rpb_cache/", "The path to the output directory")
	currentVersion := flag.String("v", "", "The minecraft version to use for processing")

	flag.Parse()
	log.SetFlags(0)

	args := flag.Args()

	s, err := os.Stat(*outputPath)
	if err != nil {
		log.Fatal(err)
	} else if !s.IsDir() {
		log.Fatal("output path must be directory")
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
	exts := strings.Split(*ext, ",")
	failed := false
	for _, dir := range args {
		if *isFile {
			panic("not implemented")
		} else {
			base := filepath.Base(dir)

			err := os.RemoveAll(filepath.Join(*outputPath, base))
			if err != nil && !os.IsNotExist(err) {
				log.Fatal(err)
			}

			err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
				if d.IsDir() {
					return nil
				}
				rel, _ := filepath.Rel(dir, path)
				outputPath := filepath.Join(*outputPath, base, rel)
				os.MkdirAll(filepath.Dir(outputPath), 0644)

				if !slices.Contains(exts, filepath.Ext(path)[1:]) {
					err := os.Link(path, outputPath)
					if err != nil {
						log.Fatal(err)
					}
					return nil
				}

				//fmt.Println("processing: ", rel)
				failed = !preprocessFile(path, outputPath, variables, m, m[*currentVersion]) || failed

				return nil
			})
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	if failed {
		os.Exit(1)
	}
}
