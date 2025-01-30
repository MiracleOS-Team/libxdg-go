package desktopFiles

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// downloadURL downloads the content of a URL to a temporary file and returns the file path.
func downloadURL(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download URL %s: %v", url, err)
	}
	defer resp.Body.Close()

	// Create a temporary file
	tempFile, err := os.CreateTemp("", "downloaded-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %v", err)
	}
	defer tempFile.Close()

	// Write response body to the temporary file
	if _, err := io.Copy(tempFile, resp.Body); err != nil {
		return "", fmt.Errorf("failed to write to temporary file: %v", err)
	}

	return tempFile.Name(), nil
}

// validateAndExecute processes the Exec key according to the specification, then executes the command.
func ExecuteDesktopFile(dfile DesktopFile, urls []string, loc string) error {
	execCommand := dfile.ApplicationObject.Exec
	if execCommand == "" {
		return fmt.Errorf("exec key cannot be empty")
	}

	// Define valid field codes
	validFieldCodes := map[string]bool{
		"%f": true, "%F": true, "%u": true, "%U": true,
		"%i": true, "%c": true, "%k": true,
	}

	// Validate command for invalid field codes
	fieldCodeRegex := regexp.MustCompile(`%[a-zA-Z]`)
	matches := fieldCodeRegex.FindAllString(execCommand, -1)
	for _, match := range matches {
		if !validFieldCodes[match] {
			fmt.Printf("Warning: Invalid field code ignored: %s\n", match)
			execCommand = strings.ReplaceAll(execCommand, match, "")
		}
	}

	// Prepare replacements for field codes
	fieldCodeReplacements := map[string]string{
		"%f": "",
		"%F": "",
		"%u": "",
		"%U": strings.Join(urls, ""),
		"%i": fmt.Sprintf("--icon %s", dfile.Icon),
		"%c": dfile.Name,
		"%k": loc,
	}

	if len(urls) != 0 {
		fieldCodeReplacements["%u"] = urls[0]
	}

	// Lazy download handling for %f and %F
	urlFiles := map[string]string{} // Cache downloaded URLs
	var downloadedFiles []string

	for _, match := range matches {
		if match == "%f" || match == "%F" {
			for _, url := range urls {
				// Check if already downloaded
				if _, exists := urlFiles[url]; !exists {
					filePath, err := downloadURL(url)
					if err != nil {
						fmt.Printf("Warning: Failed to download URL %s: %v\n", url, err)
						continue
					}
					urlFiles[url] = filePath
					downloadedFiles = append(downloadedFiles, filePath)
				}
			}
		}
	}

	// Populate replacements for %f and %F
	if len(downloadedFiles) > 0 {
		fieldCodeReplacements["%f"] = downloadedFiles[0]
		fieldCodeReplacements["%F"] = strings.Join(downloadedFiles, " ")
	}

	// Split the command into arguments
	args := strings.Fields(execCommand)
	processedArgs := []string{}

	for _, arg := range args {
		// Unescape characters
		arg = strings.ReplaceAll(arg, "\\\\", "\\")
		arg = strings.ReplaceAll(arg, "\\\"", "\"")
		arg = strings.ReplaceAll(arg, "\\$", "$")

		// Expand field codes
		for code, replacement := range fieldCodeReplacements {
			arg = strings.ReplaceAll(arg, code, replacement)
		}

		processedArgs = append(processedArgs, arg)
	}

	if len(processedArgs) == 0 {
		return fmt.Errorf("no executable or arguments specified")
	}

	// Extract the executable and arguments
	executable := processedArgs[0]
	arguments := processedArgs[1:]

	// Check if the executable exists in PATH
	pathExecutable, err := exec.LookPath(executable)
	if err != nil {
		return fmt.Errorf("executable not found in PATH: %s", executable)
	}

	// Execute the command
	var cmd *exec.Cmd
	if dfile.ApplicationObject.Terminal {
		args = []string{"-e", strings.Join([]string{"\"", pathExecutable, "\""}, "")}
		args = append(args, arguments...)
		cmd = exec.Command("alacritty", args...)
	} else {
		cmd = exec.Command(pathExecutable, arguments...)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if dfile.ApplicationObject.Path == "" {
		dfile.ApplicationObject.Path = "/"
	}
	cmd.Dir = dfile.ApplicationObject.Path

	return cmd.Run()
}
