package icons

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Subdir struct {
	Type      string // Fixed, Scaled, Threshold
	PathName  string
	Size      int
	MinSize   int
	MaxSize   int
	Threshold int
	Scale     int
	Context   string
}

type Theme struct {
	Name     string
	Subdirs  []Subdir
	Parents  []string
	BasePath string
}

// DirectoryMatchesSize checks if the subdirectory matches the requested size and scale.
func directoryMatchesSize(subdir Subdir, iconSize, iconScale int) bool {
	if subdir.Scale != iconScale {
		return false
	}
	switch subdir.Type {
	case "Fixed":
		return subdir.Size == iconSize
	case "Scaled":
		return subdir.MinSize <= iconSize && iconSize <= subdir.MaxSize
	case "Threshold":
		return (subdir.Size-subdir.Threshold) <= iconSize && iconSize <= (subdir.Size+subdir.Threshold)
	default:
		return false
	}
}

// DirectorySizeDistance calculates the size distance for mismatched directories.
func directorySizeDistance(subdir Subdir, iconSize, iconScale int) int {
	if subdir.Type == "Fixed" {
		return abs(subdir.Size*subdir.Scale - iconSize*iconScale)
	}
	if subdir.Type == "Scaled" {
		if iconSize*iconScale < subdir.MinSize*subdir.Scale {
			return subdir.MinSize*subdir.Scale - iconSize*iconScale
		}
		if iconSize*iconScale > subdir.MaxSize*subdir.Scale {
			return iconSize*iconScale - subdir.MaxSize*subdir.Scale
		}
		return 0
	}
	if subdir.Type == "Threshold" {
		if iconSize*iconScale < (subdir.Size-subdir.Threshold)*subdir.Scale {
			return (subdir.Size-subdir.Threshold)*subdir.Scale - iconSize*iconScale
		}
		if iconSize*iconScale > (subdir.Size+subdir.Threshold)*subdir.Scale {
			return iconSize*iconScale - (subdir.Size+subdir.Threshold)*subdir.Scale
		}
		return 0
	}
	return 0
}

// LookupIcon attempts to find an icon file in the theme's directories.
func LookupIcon(iconName string, size, scale int, theme Theme) (string, error) {
	var closestFilename string
	minDistance := int(^uint(0) >> 1) // MaxInt
	extensions := []string{"png", "svg", "xpm"}

	for _, subdir := range theme.Subdirs {
		if subdir.Size == size && subdir.Scale == scale {
			for _, ext := range extensions {
				filename := filepath.Join(theme.BasePath, subdir.PathName, fmt.Sprintf("%s.%s", iconName, ext))
				if fileExists(filename) && directoryMatchesSize(subdir, size, scale) {
					return filename, nil
				}
				if fileExists(filename) {
					distance := directorySizeDistance(subdir, size, scale)
					if distance < minDistance {
						closestFilename = filename
						minDistance = distance
					}
				}
			}
		}

	}
	if closestFilename != "" {
		return closestFilename, nil
	}
	return "", errors.New("icon not found")
}

// FindIconHelper recursively searches for an icon in the theme and its parents.
func findIconHelper(icon string, size, scale int, theme Theme, themeMap map[string]Theme) (string, error) {
	filename, err := LookupIcon(icon, size, scale, theme)
	if err == nil {
		return filename, nil
	}
	for _, parentName := range theme.Parents {
		parentTheme, exists := themeMap[parentName]
		if !exists {
			continue
		}
		filename, err = findIconHelper(icon, size, scale, parentTheme, themeMap)
		if err == nil {
			return filename, nil
		}
	}
	return "", errors.New("icon not found in theme or parents")
}

// FindIcon implements the main logic to find an icon.
func FindIcon(icon string, size, scale int, theme Theme, themeMap map[string]Theme) (string, error) {
	filename, err := findIconHelper(icon, size, scale, theme, themeMap)
	if err == nil {
		return filename, nil
	}
	hicolorTheme, exists := themeMap["hicolor"]
	if !exists {
		return "", errors.New("hicolor theme not found")
	}
	filename, err = findIconHelper(icon, size, scale, hicolorTheme, themeMap)
	if err == nil {
		return filename, nil
	}
	return lookupFallbackIcon(icon)
}

// LookupFallbackIcon looks for an icon in fallback directories.
func lookupFallbackIcon(icon string) (string, error) {
	fallbackDirs := []string{"/usr/share/icons", "/usr/share/pixmaps"}
	extensions := []string{"png", "svg", "xpm"}

	for _, dir := range fallbackDirs {
		for _, ext := range extensions {
			filename := filepath.Join(dir, fmt.Sprintf("%s.%s", icon, ext))
			if fileExists(filename) {
				return filename, nil
			}
		}
	}
	return "", errors.New("fallback icon not found")
}

// Utility function to check if a file exists.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	return err == nil && !info.IsDir()
}

// Utility function for absolute value.
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
