package main

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/alexflint/go-arg"
)

var wg sync.WaitGroup = sync.WaitGroup{}

var args struct {
	Inverted      bool   `arg:"-i,--inverted" help:"inverted sort" default:"false"`
	MaxDepth      int    `arg:"--depth" help:"max depth to go (0 = infinite)" default:"0"`
	All           bool   `arg:"-a,--all" help:"do not ignore entries starting with ."`
	Tree          bool   `arg:"-t,--tree" help:"Prints Tree of all indexed files"`
	Top           int    `arg:"--top" help:"N Top files" default:"0"`
	IncludeDirs   bool   `arg:"-d,--dirs" help:"include directories" default:"false"`
	HumanReadable bool   `arg:"-H, --" help:"human readable sizes" default:"false"`
	Verbose       bool   `arg:"-v, --verbose" help:"verbose mode" default:"false"`
	Folder        string `arg:"positional,required"`
}

func main() {
	start_time := time.Now()
	arg.MustParse(&args)

	if stat, err := os.Stat(args.Folder); errors.Is(err, os.ErrNotExist) || !stat.IsDir() {
		fmt.Printf("Folder path specified '%s' does not exists OR is not a directory.\n", args.Folder)
		os.Exit(1)
	}

	root := make_root_folder()

	wg.Add(1)
	go read_folder(&wg, args.Folder, root, 1, args.MaxDepth, args.All)
	wg.Wait()
	root.CalculateSize()
	index_duration := time.Since(start_time)

	var sorting_time time.Duration
	var printing_time time.Duration
	var file_count int
	if args.Tree {
		start_sorting := time.Now()
		root.Sort(args.Inverted)
		sorting_time = time.Since(start_sorting)

		start_printing := time.Now()
		fmt.Println(root.ToTree(0, args.HumanReadable))
		printing_time = time.Since(start_printing)

		file_count = root.Length(args.IncludeDirs)
	} else {
		start_sorting := time.Now()
		files := root.ToSlice(args.IncludeDirs)
		slices.SortFunc(files, func(a, b *File) int {
			if args.Inverted {
				return int(a.size - b.size)
			} else {
				return int(b.size - a.size)
			}
		})
		sorting_time = time.Since(start_sorting)

		start_printing := time.Now()
		for i, f := range files {
			if args.Top > 0 && i >= args.Top {
				break
			}

			fmt.Printf("%-*s %s\n", 12, f.Size(args.HumanReadable), f.FullPath())
		}
		printing_time = time.Since(start_printing)
		file_count = len(files)
	}

	if args.Verbose {
		fmt.Printf("## File Count %d\n", file_count)
		fmt.Printf("## Sorting Time %s\n", sorting_time)
		fmt.Printf("## Printing Time %s\n", printing_time)
		fmt.Printf("## Index Time %s\n", index_duration)
		fmt.Printf("## Total Times %s\n", time.Since(start_time))
	}
}
