package main

import (
	"encoding/csv"
	"fmt"
	"html"
	"html/template"
	"io"
	"net/http"
	"strings"

	"github.com/antchfx/xmlquery"
)

// Global cache for tracking compiled results across export tasks
var currentCSVData [][]string

func main() {
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/process", handleProcess)
	http.HandleFunc("/export", handleExport)

	fmt.Println("Server launching at http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
	}
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, fmt.Sprintf("Template loading error: %v", err), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func handleProcess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. Ingest XML Input Content
	xmlData := r.FormValue("xml_text")
	file, _, err := r.FormFile("xml_file")
	if err == nil {
		defer file.Close()
		fileBytes, err := io.ReadAll(file)
		if err == nil {
			xmlData = string(fileBytes)
		}
	}

	if strings.TrimSpace(xmlData) == "" {
		w.Write([]byte("<div class='alert error'>Please provide XML data.</div>"))
		return
	}

	doc, err := xmlquery.Parse(strings.NewReader(xmlData))
	if err != nil {
		w.Write([]byte(fmt.Sprintf("<div class='alert error'>Invalid XML: %v</div>", err)))
		return
	}

	// 2. Process Configuration Field Mapping Lists
	r.ParseForm()
	keys := r.Form["keys[]"]
	xpaths := r.Form["xpaths[]"]

	// Define our flat Key-Value structure layout for CSV and View Outputs
	currentCSVData = [][]string{{"Key Name", "Extracted Value"}}

	for i, xp := range xpaths {
		key := "Untitled"
		if i < len(keys) && strings.TrimSpace(keys[i]) != "" {
			key = strings.TrimSpace(keys[i])
		}

		xpTrimmed := strings.TrimSpace(xp)
		if xpTrimmed != "" {
			nodes, err := xmlquery.QueryAll(doc, xpTrimmed)
			if err == nil {
				for _, node := range nodes {
					// Every match gets serialized directly as an individual Key-Value row
					currentCSVData = append(currentCSVData, []string{key, node.InnerText()})
				}
			}
		}
	}

	// 3. Render out the interactive HTML Table Grid view snippet
	var htmlResponse strings.Builder
	htmlResponse.WriteString("<div style='margin-bottom: 15px;'><a href='/export' class='btn btn-success'>📥 Download Key-Value CSV</a></div>")
	htmlResponse.WriteString("<div class='table-container'><table><thead><tr>")
	htmlResponse.WriteString("<th>Key Name</th><th>Extracted Value</th>")
	htmlResponse.WriteString("</tr></thead><tbody>")

	for i, row := range currentCSVData {
		if i == 0 {
			continue // Skip headers row in loop
		}
		htmlResponse.WriteString("<tr>")
		htmlResponse.WriteString(fmt.Sprintf("<td style='font-weight:600; color:#2c3e50;'>%s</td>", html.EscapeString(row[0])))
		htmlResponse.WriteString(fmt.Sprintf("<td>%s</td>", html.EscapeString(row[1])))
		htmlResponse.WriteString("</tr>")
	}
	htmlResponse.WriteString("</tbody></table></div>")

	w.Write([]byte(htmlResponse.String()))
}

func handleExport(w http.ResponseWriter, r *http.Request) {
	if len(currentCSVData) <= 1 {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename=key_value_extracted_data.csv")
	w.Header().Set("Content-Type", "text/csv")

	writer := csv.NewWriter(w)
	writer.WriteAll(currentCSVData)
	writer.Flush()
}
