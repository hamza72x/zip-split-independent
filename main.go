package main

import (
	"archive/zip"
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	col "github.com/hamza02x/go-color"
	hel "github.com/hamza02x/go-helper"
)

type xZipFile struct {
	SizeKB    float64
	FilePaths []string
}

var dir string
var splitSizeMB float64
var splitSizeKB float64
var zipDir string
var err error

func main() {

	flags()

	fmt.Println("Directory to be zipped:", dir)
	fmt.Printf("Split size: %dMB\n", int(splitSizeMB))

	zipFiles := getZipFiles()

	for i, zip := range zipFiles {

		zipName := fmt.Sprintf("%s/zip-%d.zip", zipDir, i+1)

		fmt.Printf("%s %s\n", col.Red("Creating"), zipName)

		if hel.FileExists(zipName) {

			reader := bufio.NewReader(os.Stdin)

			fmt.Printf("%s exists already, delete? (y/n)\n", zipName)

			text, _ := reader.ReadString('\n')

			text = strings.TrimSuffix(text, "\n")

			if text == "y" || text == "yes" {
				hel.FileRemoveIfExists(zipName)
			} else {
				fmt.Printf("Exiting since %s already exists", zipName)
				break
			}
		}

		err := makeZip(zipName, zip.FilePaths)

		if err != nil {
			fmt.Printf("Error creating %s, Error:\t%v", zipName, err)
			os.Exit(1)
		}

		fmt.Printf("%s %s\n", col.Green("Created"), zipName)
	}
}

func flags() {

	flag.StringVar(&dir, "d", "", "the directory which need to be zipped")
	flag.StringVar(&zipDir, "o", "zip-splits", "output zip directory (not zip file)")
	flag.Float64Var(&splitSizeMB, "s", 1, "split size (in MB)")
	flag.Parse()

	if dir == "" {
		flag.Usage()
		os.Exit(1)
	}

	dir, err = filepath.Abs(dir)

	if err != nil {
		fmt.Println("Error getting absolute path", err)
		os.Exit(1)
	}

	hel.DirCreateIfNotExists(zipDir)

	splitSizeKB = splitSizeMB * 1024.0
}

func getZipFiles() []xZipFile {
	var zipFies []xZipFile

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {

		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		var c = len(zipFies)
		var fileSizeKB = float64(info.Size()) / 1024.0
		var fileSizeMB = float64(info.Size()) / 1024.0 / 1024.0

		if fileSizeMB > splitSizeMB {
			fmt.Printf(
				"Minimum split size (%dMB) can't be lower than any of the file-size's (%dMB) maximum\n",
				int(splitSizeMB),
				int(fileSizeMB),
			)
			fmt.Println("Exiting")
			os.Exit(1)
		}

		if c == 0 {
			zipFies = append(zipFies, xZipFile{
				SizeKB:    fileSizeKB,
				FilePaths: []string{path},
			})
		} else {

			if (zipFies[c-1].SizeKB + fileSizeKB) > splitSizeKB {

				zipFies = append(zipFies, xZipFile{
					SizeKB:    fileSizeKB,
					FilePaths: []string{path},
				})

			} else {

				zipFies[c-1].SizeKB += fileSizeKB
				zipFies[c-1].FilePaths = append(zipFies[c-1].FilePaths, path)

			}

		}

		return nil
	})

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return zipFies
}

// makeZip compresses one or many files into a single zip archive file.
// Param 1: filename is the output zip file's name.
// Param 2: files is a list of files to add to the zip.
func makeZip(filename string, files []string) error {

	newZipFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer newZipFile.Close()

	zipWriter := zip.NewWriter(newZipFile)
	defer zipWriter.Close()

	// Add files to zip
	for _, file := range files {
		if err = addFileToZip(zipWriter, file); err != nil {
			return err
		}
	}
	return nil
}

// addFileToZip adding a file to zip writer
func addFileToZip(zipWriter *zip.Writer, filepath string) error {

	fileToZip, err := os.Open(filepath)

	if err != nil {
		return err
	}

	defer fileToZip.Close()

	// Get the file information
	info, err := fileToZip.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	// Using FileInfoHeader() above only uses the basename of the file. If we want
	// to preserve the folder structure we can overwrite this with the full path.

	headerName := strings.ReplaceAll(filepath, dir, ".")

	header.Name = headerName

	// Change to deflate to gain better compression
	// see http://golang.org/pkg/archive/zip/#pkg-constants
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, fileToZip)

	if err == nil {
		fmt.Printf("Added:\t%s\n", headerName)
	} else {
		fmt.Printf("Error Adding:\t%s\n, Error:\t%v", headerName, err)
	}

	return err
}
