package lib

import (
	"os"
	"regexp"
)

func Exists(path string) (bool, error) {
	_, err := os.Stat(path)

	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return false, err
}

/**
 * Parses url with the given regular expression and returns the
 * group values defined in the expression.
 *
 */
func GetParams(regEx, url string) (paramsMap map[string]string) {
	compRegEx := regexp.MustCompile(regEx)
	match := compRegEx.FindStringSubmatch(url)

	paramsMap = make(map[string]string)
	for i, name := range compRegEx.SubexpNames() {
		if i > 0 && i <= len(match) {
			paramsMap[name] = match[i]
		}
	}
	return paramsMap
}

// GetCmdOutput -
func GetCmdOutput(command string, args ...string) []string {
	lines := make([]string, 0)

	if len(args) > 0 {
		ShellEx(command, func(line string) {
			lines = append(lines, line)
		}, args...)
	} else {
		Shell(command, func(line string) {
			lines = append(lines, line)
		})
	}

	return lines
}

func Round(a float64) int {
	if a < 0 {
		return int(a - 0.5)
	}
	return int(a + 0.5)
}
