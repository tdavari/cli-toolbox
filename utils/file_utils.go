package utils

import (
    "bufio"
    "os"
    "strings"
)

func ReadFileToList(fileName string) ([]string, error) {
    // Open the file
    file, err := os.Open(fileName)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    // Initialize an empty list to store lines
    lines := []string{}

    // Create a scanner to read the file line by line
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        // Strip leading and trailing whitespace from each line
        line := scanner.Text()
        line = strings.TrimSpace(line)
        lines = append(lines, line)
    }

    // Check for errors during scanning
    if err := scanner.Err(); err != nil {
        return nil, err
    }

    return lines, nil
}
