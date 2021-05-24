package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

// Various runtime flags
var rootPath = flag.String("path", "./templates", "Where to search for files and templates")
var outDir = flag.String("out", "./build/", "Where to search for files and templates")

var templates *template.Template
var absRootPath string

func applyTemplate(currentPath string, d os.DirEntry, err error) error {
	if strings.Contains(currentPath, path.Join(*rootPath, *outDir)) {
		// We are in the templates or output folder, so we can just skip
		return nil
	}

	// Convert the current path to the corresponding output dir path
	outPath := strings.Replace(currentPath, absRootPath, *outDir, -1)

	// If we're looking at a dir just make the path
	os.MkdirAll(filepath.Dir(outPath), os.FileMode(os.O_RDWR))
	if d.IsDir() {
		return nil
	}

	// Open the file for writing
	outfile, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("failed to write to output: %v", err)
	}
	defer outfile.Close()

	// Template/create the file
	err = templates.ExecuteTemplate(outfile, filepath.Base(currentPath), nil)

	return err
}

func main() {
	flag.Parse()

	var err error
	absRootPath, err = filepath.Abs(*rootPath)
	if err != nil {
		log.Fatalf("Failed to ref rootPath: %v\n", err)
	}

	templates, err = template.ParseGlob(filepath.Join(absRootPath, "*.html"))
	if err != nil {
		log.Fatalf("Failed to load templates. 'path' glob may be wrong: %v\n", err)
	}

	err = filepath.WalkDir(absRootPath, applyTemplate)
	if err != nil {
		log.Fatalf("Failed to apply templates. Err: %v\n", err)
	}
}
