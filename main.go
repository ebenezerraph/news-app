package main

import (
	"bytes"
	"html/template"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/freshman-tech/news-demo/news"
	"github.com/joho/godotenv"
	"golang.org/x/net/html"
)

var tpl *template.Template

func add(a, b int) int {
	return a + b
}

func init() {
	var err error
	tpl = template.New("index.html").Funcs(template.FuncMap{
		"add": add,
	})
	tpl, err = tpl.ParseFiles("index.html")
	if err != nil {
		log.Fatalf("Error parsing template: %v", err)
	}
}

type Search struct {
	Query        string
	ErrorMsg     string
	Results      *news.Results
	CurrentPage  int
	TotalPages   int
	NextPage     int
	PreviousPage int
}

func (s *Search) IsLastPage() bool {
	return s.CurrentPage == s.TotalPages
}

var (
	imageCache sync.Map
	client     = &http.Client{Timeout: 2 * time.Second}
)

func (s *Search) IsImageValid(url string) bool {
	if url == "" {
		return false
	}

	if valid, ok := imageCache.Load(url); ok {
		return valid.(bool)
	}

	go func() {
		resp, err := client.Head(url)
		valid := err == nil && resp.StatusCode == http.StatusOK
		imageCache.Store(url, valid)
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()

	return true
}

type NewsAPIClient struct {
	*news.Client
}

func NewNewsAPIClient(apiKey string, pageSize int) *NewsAPIClient {
	return &NewsAPIClient{
		Client: news.NewClient(&http.Client{Timeout: 10 * time.Second}, apiKey, pageSize),
	}
}

type Server struct {
	newsapi *NewsAPIClient
}

func NewServer(apiKey string) *Server {
	return &Server{
		newsapi: NewNewsAPIClient(apiKey, 20),
	}
}

func (s *Server) fetchResults(query, page string) (*news.Results, error) {
	apiStart := time.Now()
	log.Printf("Search Query: %s, Page: %s", query, page)

	results, err := s.newsapi.FetchEverything(query, page)
	apiElapsed := time.Since(apiStart)
	log.Printf("API fetch took %s", apiElapsed)

	if err != nil {
		log.Printf("Error fetching results from API: %v", err)
		return nil, err
	}

	var filteredArticles []news.Article
	for i, article := range results.Articles {
		if article.Title != "[Removed]" && article.Description != "[Removed]" {
			isDuplicate := false
			for j := 0; j < i; j++ {
				prevArticle := results.Articles[j]
				if prevArticle.Source.Name == article.Source.Name &&
					(prevArticle.Title == article.Title || prevArticle.Description == article.Description) {
					isDuplicate = true
					break
				}
			}
			if !isDuplicate {
				filteredArticles = append(filteredArticles, article)
			}
		}
	}

	results.Articles = filteredArticles

	return results, nil
}

func (s *Server) indexHandler(w http.ResponseWriter, r *http.Request) {
	buf := &bytes.Buffer{}
	err := tpl.ExecuteTemplate(buf, "index.html", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	buf.WriteTo(w)
}

func getArticleCount(n *html.Node) int {
	if n.Type == html.ElementNode && n.Data == "div" {
		for _, a := range n.Attr {
			if a.Key == "id" && a.Val == "articleCount" {
				for _, attr := range n.Attr {
					if attr.Key == "data-count" {
						count, _ := strconv.Atoi(attr.Val)
						return count
					}
				}
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if count := getArticleCount(c); count != 0 {
			return count
		}
	}
	return 0
}

func (s *Server) searchHandler(w http.ResponseWriter, r *http.Request) {
	handlerStart := time.Now()

	u, err := url.Parse(r.URL.String())
	if err != nil {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	params := u.Query()
	searchQuery := params.Get("q")
	page := params.Get("page")
	if page == "" {
		page = "1"
	}
	if searchQuery == "" {
		s.indexHandler(w, r)
		return
	}

	results, err := s.fetchResults(searchQuery, page)
	if err != nil {
		s.handleSearchError(w, searchQuery, page, err)
		return
	}

	currentPage, err := strconv.Atoi(page)
	if err != nil {
		http.Error(w, "Invalid page number", http.StatusBadRequest)
		return
	}

	pageSize := s.newsapi.PageSize
	totalPages := int(math.Ceil(float64(results.TotalResults) / float64(pageSize)))
	if totalPages > 5 {
		totalPages = 5
	}

	nextPage := currentPage + 1
	previousPage := currentPage - 1
	if previousPage < 1 {
		previousPage = 1
	}

	search := &Search{
		Query:        searchQuery,
		Results:      results,
		CurrentPage:  currentPage,
		TotalPages:   totalPages,
		NextPage:     nextPage,
		PreviousPage: previousPage,
	}

	for _, article := range results.Articles {
		if article.URLToImage != "" {
			search.IsImageValid(article.URLToImage)
		}
	}

	templateStart := time.Now()
	buf := &bytes.Buffer{}
	err = tpl.Execute(buf, search)
	templateElapsed := time.Since(templateStart)
	log.Printf("Template rendering took %s", templateElapsed)

	doc, docErr := html.Parse(bytes.NewReader(buf.Bytes()))
	if docErr != nil {
		log.Printf("Error parsing rendered HTML: %v", err)
	} else {
		count := getArticleCount(doc)
		log.Printf("Number of articles displayed in template: %d", count)
	}

	if err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		return
	}
	buf.WriteTo(w)

	handlerElapsed := time.Since(handlerStart)
	log.Printf("Total handler time: %s", handlerElapsed)
}

func (s *Server) handleSearchError(w http.ResponseWriter, query, page string, err error) {
	search := &Search{
		Query:    query,
		ErrorMsg: "Error fetching news. Please check your internet connection and try again later.",
	}
	if page > "5" {
		search.ErrorMsg = "You can only get a maximum of 5 pages."
	}
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		search.ErrorMsg = "The request timed out. Please try again later."
	}
	buf := &bytes.Buffer{}
	if err := tpl.ExecuteTemplate(buf, "index.html", search); err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		return
	}

	buf.WriteTo(w)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file, falling back to system environment")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	apiKey := os.Getenv("NEWS_API_KEY")
	if apiKey == "" {
		log.Fatal("NEWS_API_KEY must be set in .env file or system environment")
	}

	server := NewServer(apiKey)

	fs := http.FileServer(http.Dir("assets"))

	mux := http.NewServeMux()
	mux.Handle("/assets/", http.StripPrefix("/assets/", fs))
	mux.HandleFunc("/", server.indexHandler)
	mux.HandleFunc("/search", server.searchHandler)

	log.Printf("Starting server on :%s\n", port)
	err = http.ListenAndServe(":"+port, mux)
	if err != nil {
		log.Fatal(err)
	}
}
