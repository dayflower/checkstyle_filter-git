package main

import (
	"encoding/xml"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/phayes/checkstyle"
	"github.com/waigani/diffparser"
)

func findPatchByFile(d *diffparser.Diff, fileName string) *diffparser.DiffFile {
	for _, file := range d.Files {
		if file.NewName == fileName {
			return file
		}
	}

	return nil
}

func includedInChangedLineNumbers(p *diffparser.DiffFile, target int) bool {
	for _, hunk := range p.Hunks {
		for _, line := range hunk.NewRange.Lines {
			if line.Number > target {
				return false
			}

			if line.Number == target && line.Mode == diffparser.ADDED {
				return true
			}
		}
	}

	return false
}

func main() {
	commitIsh := os.Args[1]
	gitDiff, _ := exec.Command("git", "diff", "--no-color", commitIsh).Output()

	patches, _ := diffparser.Parse(string(gitDiff))

	body, _ := ioutil.ReadAll(os.Stdin)
	document := checkstyle.CheckStyle{}
	xml.Unmarshal(body, &document)

	basepath, _ := os.Getwd()

	files := []*checkstyle.File{}
	for _, fileElement := range document.File {
		file, _ := filepath.Rel(basepath, fileElement.Name)
		patch := findPatchByFile(patches, file)
		if patch != nil {
			errors := []*checkstyle.Error{}
			for _, errorElement := range fileElement.Error {
				if includedInChangedLineNumbers(patch, errorElement.Line) {
					errors = append(errors, errorElement)
				}
			}

			if len(errors) > 0 {
				fileElement.Error = errors
				files = append(files, fileElement)
			}
		}
	}

	document.File = files

	bytes, _ := xml.MarshalIndent(document, "", "   ")
	os.Stdout.Write(bytes)
}
