package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// JSON writes data as JSON to stdout
func JSON(data interface{}) error {
	return JSONTo(os.Stdout, data)
}

// JSONTo writes data as JSON to the given writer
func JSONTo(w io.Writer, data interface{}) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// JSONCompact writes data as compact JSON to stdout
func JSONCompact(data interface{}) error {
	return JSONCompactTo(os.Stdout, data)
}

// JSONCompactTo writes data as compact JSON to the given writer
func JSONCompactTo(w io.Writer, data interface{}) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(data)
}

// Output writes data in the specified format
func Output(format string, data interface{}) error {
	switch format {
	case "json":
		return JSON(data)
	case "table", "":
		return Table(data)
	default:
		return fmt.Errorf("unknown output format: %s", format)
	}
}
