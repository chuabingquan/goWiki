package main

import (
	"errors"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
)

// Global
var templateRoot = "tmpl/"
var pageDataRoot = "data/"
var templates = template.Must(template.ParseFiles(templateRoot+"edit.html", templateRoot+"view.html"))
var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")
var linksPattern = regexp.MustCompile("\\[[a-zA-Z0-9]+\\]")

// Page data structure
type Page struct {
	Title    string
	Body     []byte // Byte slice
	HTMLBody template.HTML
}

// Persist page to storage
func (p *Page) save() error {
	// Create data/ directory in case it does not exist
	os.Mkdir("data", 0777)
	filename := p.Title + ".txt"
	return ioutil.WriteFile(pageDataRoot+filename, p.Body, 0600)
}

// Load pages
func loadPage(title string) (*Page, error) {
	filename := title + ".txt"
	body, err := ioutil.ReadFile(pageDataRoot + filename)
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

// Validate and Extract title
func getTitle(w http.ResponseWriter, r *http.Request) (string, error) {
	m := validPath.FindStringSubmatch(r.URL.Path)
	if m == nil {
		http.NotFound(w, r)
		return "", errors.New("Invalid Page Title")
	}
	return m[2], nil // Title is the second subexpression
}

// Parse Wiki content
func (p *Page) parseWiki() []byte {
	return linksPattern.ReplaceAllFunc(p.Body, replaceWikiLinks)
}

// Replace wiki links
func replaceWikiLinks(src []byte) []byte {
	stringSrc := string(src)
	linkName := stringSrc[1 : len(stringSrc)-1]
	return []byte("<a href='/view/" + linkName + "'>" + linkName + "</a>")
}

// Render Template
func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	if tmpl != "edit" {
		p.HTMLBody = template.HTML(p.parseWiki())
	}
	err := templates.ExecuteTemplate(w, tmpl+".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Root handler
func rootHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/view/FrontPage", http.StatusFound)
}

// Make handler
func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	// Closure
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract page title here from request
		// And call provided handler fn
		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, m[2])
	}
}

// View handler
func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
	}
	renderTemplate(w, "view", p)
}

// Edit handler
func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit", p)
}

// Save handler
func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	err := p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

// Main function
func main() {
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))
	http.ListenAndServe(":8080", nil)
}
