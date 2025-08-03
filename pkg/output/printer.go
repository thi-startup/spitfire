package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/olekukonko/tablewriter"
)

// OutputFormat represents different output formats
type OutputFormat string

const (
	TableFormat OutputFormat = "table"
	JSONFormat  OutputFormat = "json"
	YAMLFormat  OutputFormat = "yaml"
)

// Printer handles output formatting with a clean, generic interface
type Printer struct {
	format OutputFormat
	writer io.Writer
}

// NewPrinter creates a new output printer
func NewPrinter(format OutputFormat, writer io.Writer) *Printer {
	if writer == nil {
		writer = os.Stdout
	}
	return &Printer{
		format: format,
		writer: writer,
	}
}

// PrintTable prints data as a table with given headers
// This is the main generic interface - any command can use this
func (p *Printer) PrintTable(headers []string, rows [][]string) error {
	switch p.format {
	case TableFormat:
		return p.renderTable(headers, rows)
	case JSONFormat:
		return p.renderTableAsJSON(headers, rows)
	case YAMLFormat:
		return fmt.Errorf("YAML output not yet implemented")
	default:
		return fmt.Errorf("unsupported output format: %s", p.format)
	}
}

// PrintJSON prints arbitrary data as JSON
func (p *Printer) PrintJSON(data interface{}) error {
	encoder := json.NewEncoder(p.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// renderTable renders data as a formatted table
func (p *Printer) renderTable(headers []string, rows [][]string) error {
	if len(headers) == 0 {
		return fmt.Errorf("table headers are required")
	}

	table := tablewriter.NewWriter(p.writer)
	
	// Convert headers to []any for tablewriter API
	headerArgs := make([]any, len(headers))
	for i, h := range headers {
		headerArgs[i] = h
	}
	table.Header(headerArgs...)
	
	// Add rows using Bulk method
	if err := table.Bulk(rows); err != nil {
		return fmt.Errorf("failed to add table data: %w", err)
	}
	
	// Render the table
	if err := table.Render(); err != nil {
		return fmt.Errorf("failed to render table: %w", err)
	}

	return nil
}

// renderTableAsJSON converts table data to JSON format
func (p *Printer) renderTableAsJSON(headers []string, rows [][]string) error {
	// Convert table data to array of objects
	var objects []map[string]string
	
	for _, row := range rows {
		obj := make(map[string]string)
		for i, cell := range row {
			if i < len(headers) {
				obj[headers[i]] = cell
			}
		}
		objects = append(objects, obj)
	}
	
	return p.PrintJSON(objects)
}

// ParseOutputFormat parses string to OutputFormat
func ParseOutputFormat(format string) (OutputFormat, error) {
	switch format {
	case "table", "":
		return TableFormat, nil
	case "json":
		return JSONFormat, nil
	case "yaml", "yml":
		return YAMLFormat, nil
	default:
		return "", fmt.Errorf("invalid output format: %s. Valid values: 'table', 'json', 'yaml'", format)
	}
}

// PrintQuiet prints just names/IDs for quiet mode
func PrintQuiet(items []string, writer io.Writer) {
	if writer == nil {
		writer = os.Stdout
	}
	for _, item := range items {
		fmt.Fprintln(writer, item)
	}
}