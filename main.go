package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/freshman-tech/news-demo/news"
	"github.com/joho/godotenv"
)

var tpl *template.Template

func init() {
	tpl = template.Must(template.ParseFiles("index.html"))
}

type Search struct {
	Query      string
	NextPage   int
	TotalPages int
	Results    *news.Results
	ErrorMsg   string
}

func (s *Search) IsLastPage() bool {
	return s.NextPage >= s.TotalPages
}

func (s *Search) CurrentPage() int {
	if s.NextPage == 1 {
		return s.NextPage
	}
	return s.NextPage - 1
}

func (s *Search) PreviousPage() int {
	return s.CurrentPage() - 1
}

func (s *Search) IsImageValid(url string) bool {
	if url == "" {
		return false
	}
	resp, err := http.Head(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
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

func (s *Server) indexHandler(w http.ResponseWriter, r *http.Request) {
	buf := &bytes.Buffer{}
	err := tpl.Execute(buf, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	buf.WriteTo(w)
}

func (s *Server) searchHandler(w http.ResponseWriter, r *http.Request) {
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

	apiStart := time.Now()
	results, err := s.newsapi.FetchEverything(searchQuery, page)
	apiElapsed := time.Since(apiStart)
	log.Printf("API fetch took %s", apiElapsed)

	if err != nil {
		s.handleSearchError(w, searchQuery, err)
		return
	}

	nextPage, err := strconv.Atoi(page)
	if err != nil {
		http.Error(w, "Invalid page number", http.StatusBadRequest)
		return
	}

	search := &Search{
		Query:      searchQuery,
		NextPage:   nextPage,
		TotalPages: int(math.Ceil(float64(results.TotalResults) / float64(s.newsapi.PageSize))),
		Results:    s.filterArticles(results),
	}

	if !search.IsLastPage() {
		search.NextPage++
	}

	templateStart := time.Now()
	buf := &bytes.Buffer{}
	err = tpl.Execute(buf, search)
	templateElapsed := time.Since(templateStart)
	log.Printf("Template rendering took %s", templateElapsed)

	if err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		return
	}
	buf.WriteTo(w)

	totalElapsed := time.Since(apiStart)
	log.Printf("Total handler time: %s", totalElapsed)

	fmt.Printf("Search Query: %s, Page: %s\n", searchQuery, page)
}

func (s *Server) handleSearchError(w http.ResponseWriter, query string, err error) {
	search := &Search{
		Query:    query,
		ErrorMsg: "Error fetching news. Please check your internet connection and try again later.",
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

func (s *Server) filterArticles(results *news.Results) *news.Results {
	var filteredArticles []news.Article
	for _, article := range results.Articles {
		if article.Title != "[Removed]" && article.Description != "[Removed]" {
			filteredArticles = append(filteredArticles, article)
		}
	}
	results.Articles = filteredArticles
	return results
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	apiKey := os.Getenv("NEWS_API_KEY")
	if apiKey == "" {
		log.Fatal("Env: apiKey must be set")
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
