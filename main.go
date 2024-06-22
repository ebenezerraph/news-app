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

var tpl = template.Must(template.ParseFiles("index.html"))
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

func (s *Search) CurrentPage() int {
	if s.NextPage == 1 {
		return s.NextPage
	}

	return s.NextPage - 1
}

func (s *Search) PreviousPage() int {
	return s.CurrentPage() - 1
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	buf := &bytes.Buffer{}
	err := tpl.Execute(buf, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	buf.WriteTo(w)
}

func searchHandler(newsapi *news.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		   buf := &bytes.Buffer{}
		   err := tpl.Execute(buf, nil)
		   if err != nil {
			  http.Error(w, "Error rendering template", http.StatusInternalServerError)
			  return
		   }
		   buf.WriteTo(w)
		   return
	    }
	    results, err := newsapi.FetchEverything(searchQuery, page)
	    if err != nil {
		   search := &Search{
			  Query:    searchQuery,
			  ErrorMsg: "Error fetching news. Please check your internet connection and try again later.",
		   }
		   if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			  search.ErrorMsg = "The request timed out. Please try again later."
		   }
		   buf := &bytes.Buffer{}
		   err := tpl.Execute(buf, search)
		   if err != nil {
			  http.Error(w, "Error rendering template", http.StatusInternalServerError)
			  return
		   }
		   buf.WriteTo(w)
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
		   TotalPages: int(math.Ceil(float64(results.TotalResults) / float64(newsapi.PageSize))),
		   Results:    results,
	    }

	    var filteredArticles []news.Article

		for _, article := range results.Articles {
			// Check for missing title or description
			if article.Title != "[Removed]" && article.Description != "[Removed]" {
			filteredArticles = append(filteredArticles, article)
			}
		}

		results.Articles = filteredArticles

	    if !search.IsLastPage() {
		   search.NextPage++
	    }
	    buf := &bytes.Buffer{}
	    err = tpl.Execute(buf, search)
	    if err != nil {
		   http.Error(w, "Error rendering template", http.StatusInternalServerError)
		   return
	    }
	    buf.WriteTo(w)
	    fmt.Printf("%+v", results)
	    fmt.Println("Search Query is: ", searchQuery)
	    fmt.Println("Page is: ", page)
	}
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

	myClient := &http.Client{Timeout: 10 * time.Second}
	newsapi := news.NewClient(myClient, apiKey, 20)

	fs := http.FileServer(http.Dir("assets"))

	mux := http.NewServeMux()
	mux.Handle("/assets/", http.StripPrefix("/assets/", fs))
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/search", searchHandler(newsapi))
	http.ListenAndServe(":"+port, mux)
}
