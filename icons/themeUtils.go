package icons

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// parseIndexTheme parses the index.theme file and returns a Theme.
func parseIndexTheme(themeDir string) (Theme, error) {
	indexPath := filepath.Join(themeDir, "index.theme")
	file, err := os.Open(indexPath)
	if err != nil {
		return Theme{}, fmt.Errorf("failed to open index.theme: %w", err)
	}
	defer file.Close()

	var theme Theme
	theme.BasePath = themeDir
	currentSection := ""
	subdirs := make(map[string]Subdir)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Handle section headers
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = strings.Trim(line, "[]")
			continue
		}

		// Parse key-value pairs
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])

		if currentSection == "Icon Theme" {
			switch key {
			case "Name":
				theme.Name = value
			case "Inherits":
				theme.Parents = strings.Split(value, ",")
			case "Directories":
				dirNames := strings.Split(value, ",")
				for _, dir := range dirNames {
					subdirs[dir] = Subdir{Scale: 1, Type: "Threshold"} // Initialize subdirs
				}
			}
		} else if subdir, exists := subdirs[currentSection]; exists {
			switch key {
			case "Size":
				subdir.Size, _ = strconv.Atoi(value)
			case "MinSize":
				subdir.MinSize, _ = strconv.Atoi(value)
			case "MaxSize":
				subdir.MaxSize, _ = strconv.Atoi(value)
			case "Scale":
				subdir.Scale, _ = strconv.Atoi(value)
			case "Threshold":
				subdir.Threshold, _ = strconv.Atoi(value)
			case "Type":
				subdir.Type = value
			case "Context":
				subdir.Context = value
			}
			subdir.PathName = currentSection
			subdirs[currentSection] = subdir
		}
	}

	if err := scanner.Err(); err != nil {
		return Theme{}, fmt.Errorf("error reading index.theme: %w", err)
	}

	// Convert subdirs map to slice
	for _, subdir := range subdirs {
		theme.Subdirs = append(theme.Subdirs, subdir)
	}
	return theme, nil
}

// generateThemeMap traverses the icons directory to generate a map of themes.
func GenerateThemeMap(iconsDir string) (map[string]Theme, error) {
	themeMap := make(map[string]Theme)

	err := filepath.Walk(iconsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Check for index.theme file in the directory
			indexPath := filepath.Join(path, "index.theme")
			if _, err := os.Stat(indexPath); err == nil {
				theme, parseErr := parseIndexTheme(path)
				if parseErr != nil {
					return parseErr
				}
				themeMap[theme.Name] = theme
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to generate theme map: %w", err)
	}

	return themeMap, nil
}

func printTheme(name string, theme Theme) {
	fmt.Printf("Theme: %s\n", name)
	fmt.Printf("  BasePath: %s\n", theme.BasePath)
	fmt.Printf("  Parents: %v\n", theme.Parents)
	fmt.Printf("  Subdirs:\n")
	for _, subdir := range theme.Subdirs {
		fmt.Printf("    - Type: %s, Size: %d, MinSize: %d, MaxSize: %d, Scale: %d, Threshold: %d, Context: %s, Pathname: %s\n",
			subdir.Type, subdir.Size, subdir.MinSize, subdir.MaxSize, subdir.Scale, subdir.Threshold, subdir.Context, subdir.PathName)
	}
	fmt.Println()
}

// Utility function to print the theme map for debugging.
func printThemeMap(themeMap map[string]Theme) {
	for name, theme := range themeMap {
		printTheme(name, theme)
	}
}
