package main

import (
	"context"
	"fmt"
	"os"
	Path "path"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/alexflint/go-arg"
)

type File struct {
	name     string
	path     string
	files    map[string]File
	isFolder bool
	size     int64
}

func (f File) Type() string {
	if f.isFolder {
		return "d"
	}
	return "f"
}

func read_folder(path string, ch chan<- File, depth int) {
	defer wg.Done()
	if args.MaxDepth > 0 && depth > args.MaxDepth {
		return
	}
	ch <- File{
		name:     path,
		path:     path,
		files:    make(map[string]File),
		isFolder: true,
		size:     0,
	}

	files, err := os.ReadDir(path)
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		if file.IsDir() {
			wg.Add(1)
			go read_folder(Path.Join(path, file.Name()), ch, depth+1)
		} else {
			if stat, err := os.Stat(Path.Join(path, file.Name())); err == nil {
				ch <- File{
					name:     file.Name(),
					path:     path,
					files:    nil,
					isFolder: false,
					size:     stat.Size(),
				}
			}
		}
	}
}

func handle_files(ctx context.Context, files_ch chan File, files map[string]File) {
	for {
		select {
		case file := <-files_ch:
			if file.isFolder {
				files[file.path] = file
			} else {
				files[Path.Join(file.path, file.name)] = file

				relative_path := strings.TrimPrefix(file.path, args.Folder)
				if relative_path != "" {
					for dir, _file := Path.Split(relative_path); dir != "" && _file != ""; dir, _file = Path.Split(dir) {
						en := files[Path.Join(args.Folder, dir)]
						en.size += file.size
						files[Path.Join(args.Folder, dir)] = en
					}
				}

				dir := files[file.path]
				dir.size += file.size
				files[file.path] = dir

				// Add to root folder
				dir = files[args.Folder]
				dir.size += file.size
				files[args.Folder] = dir
			}
		case <-ctx.Done():
			wg.Done()
			return
		}
	}
}

func format_bytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f%cB",
		float64(bytes)/float64(div), "KMGTPE"[exp])
}

var wg sync.WaitGroup = sync.WaitGroup{}

var args struct {
	Inverted      bool   `arg:"-i,--inverted" help:"inverted sort" default:"false"`
	MaxDepth      int    `arg:"--depth" help:"max depth to go (0 = infinite)" default:"10"`
	Top           int    `arg:"--top" help:"N Top files" default:"10"`
	IncludeDirs   bool   `arg:"-d,--dirs" help:"include directories" default:"false"`
	HumanReadable bool   `arg:"-H, --" help:"human readable sizes" default:"false"`
	Quiet         bool   `arg:"-q, --" help:"quiet mode" default:"false"`
	Folder        string `arg:"positional,required"`
}

func main() {
	start_time := time.Now()
	arg.MustParse(&args)

	ctx, cancel := context.WithCancel(context.Background())
	files := make(map[string]File)
	files_ch := make(chan File)
	wg.Add(1)

	go read_folder(args.Folder, files_ch, 1)
	go handle_files(ctx, files_ch, files)

	wg.Wait()
	wg.Add(1)
	cancel()
	wg.Wait()

	n_files := 0
	n_folders := 0
	sorted_files := make([]File, 0)
	for _, file := range files {
		if file.isFolder {
			n_folders++
		} else {
			n_files++
		}

		if !file.isFolder || args.IncludeDirs {
			sorted_files = append(sorted_files, file)
		}
	}
	sort.SliceStable(sorted_files, func(i, j int) bool {
		if args.Inverted {
			return sorted_files[i].size < sorted_files[j].size
		} else {
			return sorted_files[i].size > sorted_files[j].size
		}
	})

	N := 8
	for i, file := range sorted_files {
		if i >= args.Top {
			break
		}
		size := ""
		if args.HumanReadable {
			size = format_bytes(file.size)
		} else {
			size = fmt.Sprintf("%d", file.size)
			N = max(N, len(size))
		}
		name := Path.Join(file.path, file.name)
		if file.isFolder {
			name = file.path
		}

		fmt.Printf("%-*s %s %s\n", N, size, file.Type(), name)
	}
	if !args.Quiet {
		fmt.Printf("%s\n", strings.Repeat("-", 20))
		fmt.Printf("%d folders | %d files\n", n_folders, n_files)
		fmt.Printf("Took %s\n", time.Since(start_time))
	}
}
