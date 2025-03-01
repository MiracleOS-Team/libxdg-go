package desktopFiles

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	basedir "github.com/MiracleOS-Team/libxdg-go/baseDir"
	"github.com/MiracleOS-Team/libxdg-go/icons"
	"gopkg.in/ini.v1"
)

type DesktopFile struct {
	Type              string
	Version           string
	Name              string
	Comment           string
	GenericName       string
	NoDisplay         bool
	Icon              string
	Hidden            bool
	OnlyShowIn        []string
	NotShowIn         []string
	DBusActivatable   bool
	Implements        []string
	ApplicationObject Application
	LinkObject        Link
	DirectoryObject   Directory
}

// DesktopEntry represents the structure of a .desktop file entry
// Application represents a desktop entry of type Application
type Application struct {
	TryExec              string   `json:"TryExec,omitempty"`              // Path to test if the program is installed
	Exec                 string   `json:"Exec,omitempty"`                 // Program to execute
	Path                 string   `json:"Path,omitempty"`                 // Working directory for the program
	Terminal             bool     `json:"Terminal,omitempty"`             // Whether to run in a terminal
	Actions              []string `json:"Actions,omitempty"`              // List of application actions
	MimeType             []string `json:"MimeType,omitempty"`             // Supported MIME types
	Categories           []string `json:"Categories,omitempty"`           // Categories for menus
	Keywords             []string `json:"Keywords,omitempty"`             // Additional search keywords
	StartupNotify        bool     `json:"StartupNotify,omitempty"`        // Whether startup notifications are supported
	StartupWMClass       string   `json:"StartupWMClass,omitempty"`       // WM class or name hint
	PrefersNonDefaultGPU bool     `json:"PrefersNonDefaultGPU,omitempty"` // Hint for using a discrete GPU
	SingleMainWindow     bool     `json:"SingleMainWindow,omitempty"`     // Hint for single-window applications
}

// Link represents a desktop entry of type Link
type Link struct {
	URL string `json:"URL"` // Target URL for the link
}

// Directory represents a desktop entry of type Directory
type Directory struct {
}

// Example of a locale selection function based on LC_MESSAGES
func getCurrentLocale() string {
	// Get the current LC_MESSAGES locale (using environment variable or similar approach)
	// For simplicity, we assume it's set in the environment variable "LC_MESSAGES"
	locale := os.Getenv("LC_MESSAGES")
	if locale == "" {
		locale = "en_US.UTF-8" // Default to English if no LC_MESSAGES is set
	}
	return locale
}

// Normalize the locale string (strip encoding, modifiers)
func normalizeLocale(locale string) string {
	// Strip the encoding and modifier parts
	// Locale format is lang_COUNTRY.ENCODING@MODIFIER
	re := regexp.MustCompile(`([a-zA-Z]{2,8})(_[a-zA-Z]{2,8})?(\.[a-zA-Z0-9_-]+)?(@[a-zA-Z0-9_-]+)?`)
	match := re.FindStringSubmatch(locale)
	if match != nil {
		// Return normalized locale of form lang_COUNTRY@MODIFIER (stripping encoding)
		return fmt.Sprintf("%s%s%s", match[1], match[2], match[4])
	}
	return locale
}

// TranslateFieldWithLocale attempts to find the appropriate localized value
func TranslateFieldWithLocale(key string, locale string, section *ini.Section) string {
	// Normalize the locale for matching (strip encoding and modifier parts)
	normalizedLocale := normalizeLocale(locale)

	// Try matching the full locale (lang_COUNTRY@MODIFIER)
	if val := section.Key(key + "[" + normalizedLocale + "]").String(); val != "" {
		return val
	}

	// Try matching lang_COUNTRY
	if parts := strings.Split(normalizedLocale, "@"); len(parts) > 1 {
		if val := section.Key(key + "[" + parts[0] + "]").String(); val != "" {
			return val
		}
	}

	// Try matching lang (fallback)
	if val := section.Key(key + "[" + strings.Split(normalizedLocale, "_")[0] + "]").String(); val != "" {
		return val
	}

	// Fallback to default (no locale)
	if val := section.Key(key).String(); val != "" {
		return val
	}

	return key // Return the original key if no match
}

func ParseIconString(value string) (string, error) {
	if strings.HasPrefix(value, "/") {
		return value, nil
	}
	if strings.Contains(value, "/") {
		return filepath.Join("/", value), nil
	}
	icon, err := icons.FindIconDefaults(value, 256, 1, "application-x-executable")
	return icon, err
}

// ReadDesktopFileWithLocale reads a .desktop file and prints key-value pairs with locale-based selection
func ReadDesktopFile(filePath string) (DesktopFile, error) {
	dfile := DesktopFile{}
	locale := getCurrentLocale()

	// Load the .desktop file
	cfg, err := ini.Load(filePath)
	if err != nil {
		return dfile, fmt.Errorf("failed to load .desktop file: %w", err)
	}

	// Iterate over sections and print key-value pairs with locale-based translation
	for _, section := range cfg.SectionStrings() {
		sectionObj := cfg.Section(section)
		for _, key := range sectionObj.KeyStrings() {
			if !strings.HasSuffix(key, "]") {
				if sectionObj.Name() == "Desktop Entry" {
					switch key {
					case "Type":
						dfile.Type = sectionObj.Key(key).String()
					case "Version":
						dfile.Version = sectionObj.Key(key).String()
					case "Name":
						dfile.Name = TranslateFieldWithLocale(key, locale, sectionObj)
					case "GenericName":
						dfile.GenericName = TranslateFieldWithLocale(key, locale, sectionObj)
					case "NoDisplay":
						dfile.NoDisplay, _ = sectionObj.Key(key).Bool()
					case "Comment":
						dfile.Comment = TranslateFieldWithLocale(key, locale, sectionObj)
					case "Icon":
						dfile.Icon, _ = ParseIconString(sectionObj.Key(key).String())
					case "Hidden":
						dfile.Hidden, _ = sectionObj.Key(key).Bool()
					case "OnlyShowIn":
						dfile.OnlyShowIn = sectionObj.Key(key).Strings(";")
					case "NotShowIn":
						dfile.NotShowIn = sectionObj.Key(key).Strings(";")
					case "DBusActivatable":
						dfile.DBusActivatable, _ = sectionObj.Key(key).Bool()
					case "TryExec":
						dfile.ApplicationObject.TryExec = sectionObj.Key(key).String()
					case "Exec":
						dfile.ApplicationObject.Exec = sectionObj.Key(key).String()
					case "Path":
						dfile.ApplicationObject.Path = sectionObj.Key(key).String()
					case "Terminal":
						dfile.ApplicationObject.Terminal, _ = sectionObj.Key(key).Bool()
					case "Actions":
						dfile.ApplicationObject.Actions = sectionObj.Key(key).Strings(";")
					case "MimeType":
						dfile.ApplicationObject.MimeType = sectionObj.Key(key).Strings(";")
					case "Implements":
						dfile.Implements = sectionObj.Key(key).Strings(";")
					case "Keywords":
						dfile.ApplicationObject.Keywords = []string{TranslateFieldWithLocale(key, locale, sectionObj)}
					case "StartupNotify":
						dfile.ApplicationObject.StartupNotify, _ = sectionObj.Key(key).Bool()
					case "StartupWMClass":
						dfile.ApplicationObject.StartupWMClass = sectionObj.Key(key).String()
					case "URL":
						dfile.LinkObject.URL = sectionObj.Key(key).String()
					case "PrefersNonDefaultGPU":
						dfile.ApplicationObject.PrefersNonDefaultGPU, _ = sectionObj.Key(key).Bool()
					case "SingleMainWindow":
						dfile.ApplicationObject.SingleMainWindow, _ = sectionObj.Key(key).Bool()

					}

				}

			}

		}
	}

	return dfile, nil
}

func ListAllApplications() ([]DesktopFile, error) {
	apps := make(map[string]DesktopFile)

	for _, dir := range basedir.GetXDGDirectory("dataDirs").([]string) {
		if _, err := os.Stat(dir + "/applications"); os.IsNotExist(err) {
			continue
		}
		slog.Info("Processing directory: ", dir+"/applications")
		app1, err := ListApplications(dir + "/applications")
		if err != nil {
			return nil, err
		}

		for nm, app := range app1 {
			apps[nm] = app
		}
		slog.Info("Finished processing directory: ", dir+"/applications")
	}

	fapps := []DesktopFile{}

	for _, app := range apps {
		fapps = append(fapps, app)
	}

	return fapps, nil
}

// ListApplications traverses a directory and parses .desktop files to list applications
func ListApplications(directory string) (map[string]DesktopFile, error) {
	var apps = make(map[string]DesktopFile)

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".desktop") {
			slog.Debug("Processing file: ", path)
			desktopFile, parseErr := ReadDesktopFile(path)
			if parseErr == nil && desktopFile.Type == "Application" && !desktopFile.NoDisplay && !desktopFile.Hidden {
				dName := strings.Replace(strings.Replace(info.Name(), directory, "", 1), "/", "-", -1)
				apps[dName] = desktopFile
			}
		} else if info.IsDir() && path != directory {
			slog.Debug("Processing subdirectory: ", path)
			tapps, err := ListApplications(path)
			if err == nil {
				for nm, app := range tapps {
					apps[info.Name()+"-"+nm] = app
				}
			}
			slog.Debug("Finished processing subdirectory: ", path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return apps, nil
}
