package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Region represents a geographical region containing XML files.
type Region struct {
	ID       int
	Name     string
	XmlFiles []XMLFile
}

// XMLFile represents an XML file with its name and path.
type XMLFile struct {
	Name string
	Path string
}

// Reads the contents of the specified directory and returns a slice of os.DirEntry representing the files and subdirectories within it.
func ReadDirectory(path string) ([]os.DirEntry, error) {
	fmt.Printf("Reading directory: %s\n", path)
	files, err := os.ReadDir(path)

	if err != nil {
		return nil, err
	}

	return files, nil
}

// Reads the XML files from the specified directory and organizes them into a structured format.
// It returns a slice of Region structs, each containing the region's name and a list of XML files associated with that region.
func GetStructuredData(xmlFilesPath string) ([]Region, error) {

	// Read the main directory to get the list of regions
	regions, err := ReadDirectory(xmlFilesPath)
	if err != nil {
		fmt.Printf("read directory: %s", err)
		return nil, fmt.Errorf("read XML directory: %w", err)
	}

	if len(regions) == 0 {
		fmt.Println("No regions found.")
		return nil, fmt.Errorf("XML directory %s is empty", xmlFilesPath)
	}

	// Process each region directory and collect XML file information
	regionsFiles := []Region{}
	fileCount := 0
	refStruct := Region{
		ID:       len(regionsFiles) + 1,
		Name:     "00",
		XmlFiles: []XMLFile{}, // Initialize the XmlFiles slice
	}

	for _, region := range regions {
		if region.IsDir() {
			fmt.Printf("Processing directory: %s\n", region.Name())

			regionsStruct := Region{
				ID:       len(regionsFiles) + 1,
				Name:     region.Name(),
				XmlFiles: []XMLFile{}, // Initialize the XmlFiles slice
			}

			// Read XML files in the region directory
			files, err := ReadDirectory(xmlFilesPath + "/" + region.Name())
			if err != nil {
				fmt.Printf("Error reading directory for region %s: %v\n", region.Name(), err)
				return nil, fmt.Errorf("read region %s: %w", region.Name(), err)
			}
			if len(files) == 0 {
				fmt.Printf("No XML files found in region %s.\n", region.Name())
				continue
			}

			// Process each XML file and add it to the region struct
			for _, file := range files {
				if !file.IsDir() && strings.EqualFold(filepath.Ext(file.Name()), ".xml") {
					xmlFile := XMLFile{
						Name: file.Name(),
						Path: xmlFilesPath + "/" + region.Name() + "/" + file.Name(),
					}
					regionsStruct.XmlFiles = append(regionsStruct.XmlFiles, xmlFile)
					fileCount++
					fmt.Printf("Added XML file: %s\n", xmlFile.Path)
				}
			}

			// Append the region struct to the regionsFiles slice
			regionsFiles = append(regionsFiles, regionsStruct)

		} else if strings.EqualFold(filepath.Ext(region.Name()), ".xml") {
			fmt.Printf("Processing files in folder : %s\n", xmlFilesPath)

			// regionStruct := findRegionByName(regionsFiles, "00")

			// if regionStruct == nil {
			// 	regionStruct := Region{
			// 		ID:       len(regionsFiles) + 1,
			// 		Name:     "00",
			// 		XmlFiles: []XMLFile{}, // Initialize the XmlFiles slice
			// 	}

			// 	// Append the region struct to the regionsFiles slice
			// 	regionsFiles = append(regionsFiles, regionStruct)
			// }

			xmlFile := XMLFile{
				Name: region.Name(),
				Path: xmlFilesPath + "/" + region.Name(),
			}
			refStruct.XmlFiles = append(refStruct.XmlFiles, xmlFile)
			fileCount++
			fmt.Printf("Added XML file: %s\n", xmlFile.Path)
		}
	}

	regionsFiles = append(regionsFiles, refStruct)
	if fileCount == 0 {
		return nil, fmt.Errorf("no XML files found in %s", xmlFilesPath)
	}

	return regionsFiles, nil
}

func findRegionByName(regions []Region, name string) *Region {
	for i := range regions {
		if regions[i].Name == name {
			return &regions[i]
		}
	}
	return nil
}
