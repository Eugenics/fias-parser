package app

import (
	"fmt"
	"os"
	"strings"
)

func ReadXmlFile(path string) {
	fmt.Printf("Reading XML file: %s\n", path)
	file, err := os.Open(path)
	if err != nil {
		fmt.Printf("Error opening XML file %s: %v\n", path, err)
		return
	}
	defer file.Close()

	excludeTables := [...]string{
		"AS_ADDR_OBJ_DIVISION",
		"AS_APARTMENTS",
		"AS_APARTMENTS_PARAMS",
		"AS_CARPLACES",
		"AS_CARPLACES_PARAMS",
		"AS_CHANGE_HISTORY",
		"AS_HOUSES",
		"AS_HOUSES_PARAMS",
		"AS_MUN_HIERARCHY",
		"AS_REESTR_OBJECTS",
		"AS_ROOMS",
		"AS_ROOMS_PARAMS",
		"AS_STEADS",
		"AS_STEADS_PARAMS",
	}

	for _, excludeTable := range excludeTables {
		if strings.Contains(file.Name(), excludeTable) {
			return
		}
	}

	Handle(file)
}
