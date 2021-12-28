package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"sort"
)

type ByName []os.FileInfo

func (f ByName) Len() int           { return len(f) }
func (f ByName) Swap(i, j int)      { f[i], f[j] = f[j], f[i] }
func (f ByName) Less(i, j int) bool { return f[i].Name() < f[j].Name() }

func fileSize(file os.FileInfo) string {
	size := file.Size()

	if size == 0 {
		return "(empty)"
	}

	return fmt.Sprintf("(%vb)", size)
}

func printFile(out io.Writer, file os.FileInfo, prefix string, isLast bool) {

	var indent string
	if isLast {
		indent = prefix + "└───"
	} else {
		indent = prefix + "├───"
	}

	var resultString string
	if file.IsDir() {
		resultString = fmt.Sprintf("%v%v\n", indent, file.Name())
	} else {
		resultString = fmt.Sprintf("%v%v %v\n", indent, file.Name(), fileSize(file))
	}

	io.WriteString(out, resultString)
}

func deleteNotDirs(files []os.FileInfo) []os.FileInfo {
	var onlyDirs []os.FileInfo
	for _, file := range files {
		if file.IsDir() {
			onlyDirs = append(onlyDirs, file)
		}
	}

	return onlyDirs
}

func prettyPrint(out io.Writer, path string, printFiles bool, prefix string) error {
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
		return err
	}

	files, _ := file.Readdir(1000)
	sort.Sort(ByName(files))

	if printFiles == false {
		files = deleteNotDirs(files)
	}

	for i, file := range files {
		isLast := false
		if i+1 == len(files) {
			isLast = true
		}

		if file.IsDir() {
			printFile(out, file, prefix, isLast)

			var add string
			if isLast {
				add = "\t"
			} else {
				add = "│\t"
			}

			prettyPrint(out, fmt.Sprintf("%v/%v", path, file.Name()), printFiles, prefix+add)
		} else if printFiles {
			printFile(out, file, prefix, isLast)
		}

	}

	return nil
}

func dirTree(out io.Writer, path string, printFiles bool) error {
	return prettyPrint(out, path, printFiles, "")
}

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}
