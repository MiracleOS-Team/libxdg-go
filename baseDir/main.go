/*
	libxdg-go - An implementaion of various freedesktop specifications in go
    Copyright (C) 2025 MiracleOS Contributors

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU General Public License for more details.

    You should have received a copy of the GNU General Public License
    along with this program.  If not, see <https://www.gnu.org/licenses/>.

*/

package basedir

import (
	"fmt"
	"os"
	"strings"
)

// GetXDGDirectory returns either a string or a slice of strings depending on the directory type.
func GetXDGDirectory(dirType string) interface{} {
	switch dirType {
	case "data":
		return getEnvOrDefault("XDG_DATA_HOME", os.Getenv("HOME")+"/.local/share")
	case "config":
		return getEnvOrDefault("XDG_CONFIG_HOME", os.Getenv("HOME")+"/.config")
	case "state":
		return getEnvOrDefault("XDG_STATE_HOME", os.Getenv("HOME")+"/.local/state")
	case "cache":
		return getEnvOrDefault("XDG_CACHE_HOME", os.Getenv("HOME")+"/.cache")
	case "runtime":
		return getEnvOrDefault("XDG_RUNTIME_DIR", "")
	case "dataDirs":
		return getEnvOrDefaultList("XDG_DATA_DIRS", "/usr/local/share:/usr/share")
	case "configDirs":
		return getEnvOrDefaultList("XDG_CONFIG_DIRS", "/etc/xdg")
	default:
		return nil
	}
}

// getEnvOrDefault returns the value of an environment variable or a default if not set or empty.
func getEnvOrDefault(envVar, defaultValue string) string {
	value := os.Getenv(envVar)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvOrDefaultList returns a slice of strings by splitting an environment variable or using a default.
func getEnvOrDefaultList(envVar, defaultValue string) []string {
	value := os.Getenv(envVar)
	if value == "" {
		value = defaultValue
	}
	fmt.Println(strings.Split(value, ":"))
	return strings.Split(value, ":")
}
