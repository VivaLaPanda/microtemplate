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

	"github.com/fsnotify/fsnotify"
)

// Various runtime flags
var rootPath = flag.String("path", "./templates", "Where to search for files and templates")
var watch = flag.Bool("watch", false, "Whether to automatically rebuild on changes")
var templatePrefix = flag.String("tprefix", "t_", "prefix for template files")
var outDir = flag.String("out", "./build/", "Where to search for files and templates")

var templates *template.Template
var watcher *fsnotify.Watcher
var absRootPath string

func applyTemplate(currentPath string, d os.DirEntry, err error) error {
	if err != nil {
		return err
	}

	if strings.Contains(currentPath, path.Join(*rootPath, *outDir)) {
		// We are in the output folder, so we can just skip
		return nil
	}

	// Convert the current path to the corresponding output dir path
	outPath := strings.Replace(currentPath, absRootPath, *outDir, -1)

	// If we're looking at a dir just make the path
	os.MkdirAll(filepath.Dir(outPath), os.FileMode(os.O_RDWR))
	if d.IsDir() {
		templates, err = templates.ParseGlob(filepath.Join(currentPath, "*.html"))
		if err != nil {
			return fmt.Errorf("failed to add subdir templates: %v", err)
		}

		if watcher != nil {
			watcher.Add(currentPath)
			if err != nil {
				return fmt.Errorf("failed to set watch on subdir: %v", err)
			}
		}

		return nil
	}

	// If we're looking at a file and it's marked as a template, don't output it
	if strings.Contains(filepath.Base(currentPath), *templatePrefix) {
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

func compileTemplates() error {
	var err error
	absRootPath, err = filepath.Abs(*rootPath)
	if err != nil {
		return fmt.Errorf("failed to ref rootPath: %v", err)
	}

	templates, err = template.ParseGlob(filepath.Join(absRootPath, "*.html"))
	if err != nil {
		return fmt.Errorf("failed to load templates. 'path' glob may be wrong: %v", err)
	}

	err = filepath.WalkDir(absRootPath, applyTemplate)
	if err != nil {
		return fmt.Errorf("failed to apply templates. Err: %v", err)
	}

	return nil
}

func main() {
	flag.Parse()

	err := compileTemplates()
	if err != nil {
		log.Fatalf("Couldn't compile templates: %v", err)
	}

	if *watch {
		watcher, err = fsnotify.NewWatcher()
		if err != nil {
			log.Fatalf("Failed to make watcher: %v", err)
		}
		defer watcher.Close()

		//
		done := make(chan bool)

		//
		go func() {
			for {
				select {
				// watch for events
				case event, ok := <-watcher.Events:
					if !ok {
						return
					}

					if event.Op&fsnotify.Write == fsnotify.Write {
						log.Println("Modified file:", event.Name)
					}
					err := compileTemplates()
					if err != nil {
						log.Printf("Couldn't compile templates: %v", err)
					}

					// watch for errors
				case err := <-watcher.Errors:
					log.Printf("Error while watching files: %v\n", err)
				}
			}
		}()

		// out of the box fsnotify can watch a single file, or a single directory
		if err := watcher.Add(*rootPath); err != nil {
			log.Fatalf("Failed to start watching path %s: %v", *rootPath, err)
		}

		<-done
	}
}
