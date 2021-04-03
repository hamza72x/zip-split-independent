package main

import (
	"archive/zip"
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	color "github.com/hamza02x/go-color"
	hel "github.com/hamza02x/go-helper"
)

type xZipFile struct {
	SizeKB    float64
	FilePaths []string
}

var (
	// flag vars
	dir         string
	zipName     string
	splitSizeMB float64
	zipDir      string
	//
	splitSizeKB float64
	err         error
)

func main() {

	flags()

	fmt.Println("Directory to be zipped:", dir)
	fmt.Printf("Split size: %dMB\n", int(splitSizeMB))

	zipFiles := getZipFiles(getFilePathSorted())

	for i, zip := range zipFiles {

		zipName := fmt.Sprintf("%s/%s-%d.zip", zipDir, zipName, i+1)

		fmt.Printf("%s %s\n", color.Red("Creating"), zipName)

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

		fmt.Printf("%s %s\n", color.Green("Created"), zipName)
	}
}

func flags() {

	flag.StringVar(&dir, "d", "", "the directory which need to be zipped")
	flag.StringVar(&zipDir, "o", "zip-splits", "output zip directory (not zip file)")
	flag.StringVar(&zipName, "n", "zip", "output zip file name prefix")
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

func getFilePathSorted() xFileInfos {

	var fileInfos xFileInfos

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {

		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

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

		fileInfos = append(fileInfos, &xFileInfo{
			Path:          path,
			PathKey:       getPathKey(path),
			FileSizeBytes: float64(info.Size()),
		})

		return nil
	})

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	sort.Sort(xByPathKey{fileInfos})

	return fileInfos
}

func getZipFiles(fileInfos xFileInfos) []xZipFile {

	var zipFies []xZipFile

	for _, f := range fileInfos {

		var c = len(zipFies)
		var fileSizeKB = f.FileSizeBytes / 1024.0

		if c == 0 {

			zipFies = append(zipFies, xZipFile{
				SizeKB:    fileSizeKB,
				FilePaths: []string{f.Path},
			})

		} else {

			if (zipFies[c-1].SizeKB + fileSizeKB) > splitSizeKB {

				zipFies = append(zipFies, xZipFile{
					SizeKB:    fileSizeKB,
					FilePaths: []string{f.Path},
				})

			} else {
				zipFies[c-1].SizeKB += fileSizeKB
				zipFies[c-1].FilePaths = append(zipFies[c-1].FilePaths, f.Path)
			}

		}

	}

	fmt.Printf("Total zip file will created - %s\n", color.Green(len(zipFies)))

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

// sorting stuff

const maxByte = 1<<8 - 1

func isDigit(d byte) bool {
	return '0' <= d && d <= '9'
}

func getPathKey(key string) string {
	sKey := make([]byte, 0, len(key)+8)
	j := -1
	for i := 0; i < len(key); i++ {
		b := key[i]
		if !isDigit(b) {
			sKey = append(sKey, b)
			j = -1
			continue
		}
		if j == -1 {
			sKey = append(sKey, 0x00)
			j = len(sKey) - 1
		}
		if sKey[j] == 1 && sKey[j+1] == '0' {
			sKey[j+1] = b
			continue
		}
		if sKey[j]+1 > maxByte {
			panic("PathKey: invalid key")
		}
		sKey = append(sKey, b)
		sKey[j]++
	}
	return string(sKey)
}

type xFileInfo struct {
	FileSizeBytes float64
	Path          string
	PathKey       string `datastore:"-"`
}

type xFileInfos []*xFileInfo

func (s xFileInfos) Len() int {
	return len(s)
}

func (s xFileInfos) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type xByPathKey struct{ xFileInfos }

func (s xByPathKey) Less(i, j int) bool {
	return s.xFileInfos[i].PathKey < s.xFileInfos[j].PathKey
}
