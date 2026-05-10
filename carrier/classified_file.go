package carrier

import (
	"fmt"
	"strings"
)

type ClassifiedFile struct {
	Category string `json:"category"`
	Name     string `json:"name"`
	Path     string `json:"path"`
}

func ExtractCategoryNames(category string) ([]string, error) {
	if category == "" {
		return nil, fmt.Errorf("Category is empty!")
	}

	return strings.Split(category, "/"), nil
}
