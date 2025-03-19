package main

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"slices"
	"strings"
	"sync"
)

func format_bytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f%cB",
		float64(bytes)/float64(div), "KMGTPE"[exp])
}

type File struct {
	name  string
	path  string
	size  int64
	isDir bool
}

func (f *File) FullPath() string {
	if f.isDir {
		return path.Join(f.path, f.name) + string(os.PathSeparator)
	}
	return path.Join(f.path, f.name)
}

func (f *File) Size(pretty bool) string {
	if pretty {
		return format_bytes(f.size)
	}
	return fmt.Sprintf("%d", f.size)
}

type FileEntry struct {
	file  File
	files []*FileEntry
	guard sync.Mutex
}

func make_root_folder() *FileEntry {
	return &FileEntry{
		file: File{
			name:  args.Folder,
			path:  args.Folder,
			isDir: true,
			size:  0,
		},
		files: make([]*FileEntry, 0),
		guard: sync.Mutex{},
	}
}

func (f *FileEntry) ToSlice(include_dirs bool) []*File {
	var files []*File
	if !f.IsDir() || include_dirs {
		files = append(files, &f.file)
	}

	if f.IsDir() {
		for _, file_entry := range f.files {
			files = append(files, file_entry.ToSlice(include_dirs)...)
		}
	}
	return files
}

func (f *FileEntry) Length(include_dirs bool) int {
	var length int
	if !f.IsDir() || include_dirs {
		length += 1
	}

	if f.IsDir() {
		for _, file_entry := range f.files {
			length += file_entry.Length(include_dirs)
		}
	}
	return length
}

func (f *FileEntry) IsDir() bool {
	return f.file.isDir
}

func (f *FileEntry) Sort(inverted bool) {
	if f.IsDir() {
		slices.SortFunc(f.files, func(i, j *FileEntry) int {
			if inverted {
				return int(i.file.size - j.file.size)
			}
			return int(j.file.size - i.file.size)
		})
		for _, inner_f := range f.files {
			inner_f.Sort(inverted)
		}
	}
}

func (f *FileEntry) CalculateSize() int64 {
	if f.IsDir() {
		for _, inner_f := range f.files {
			f.file.size += inner_f.CalculateSize()
		}
	}

	return f.file.size
}

func (f *FileEntry) ToTree(ident int, pretty bool) string {
	var buf bytes.Buffer
	size := fmt.Sprintf("%d", f.file.size)
	if pretty {
		size = format_bytes(f.file.size)
	}
	buf.WriteString(fmt.Sprintf("%s├── %s (%s)\n", strings.Repeat("│   ", ident), f.file.name, size))

	if f.IsDir() {
		for _, f := range f.files {
			buf.WriteString(f.ToTree(ident+1, pretty))
		}
	}
	return buf.String()
}

func read_folder(wg *sync.WaitGroup, folder_path string, folder *FileEntry, depth int, max_depth int, include_hidden bool) {
	defer wg.Done()
	if max_depth > 0 && depth > max_depth {
		return
	}

	files, err := os.ReadDir(folder_path)
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		if !include_hidden && strings.HasPrefix(file.Name(), ".") {
			continue
		}

		folder_entry := &FileEntry{
			file: File{
				name:  file.Name(),
				path:  folder_path,
				isDir: file.IsDir(),
			},
			guard: sync.Mutex{},
		}
		if folder_entry.IsDir() {
			folder.guard.Lock()
			folder.files = append(folder.files, folder_entry)
			folder.guard.Unlock()

			wg.Add(1)
			go read_folder(wg, folder_entry.file.FullPath(), folder_entry, depth+1, max_depth, include_hidden)
		} else {
			if stat, err := file.Info(); err == nil {
				folder.guard.Lock()
				folder_entry.file.size = stat.Size()
				folder.files = append(folder.files, folder_entry)
				folder.guard.Unlock()
			}
		}
	}
}
