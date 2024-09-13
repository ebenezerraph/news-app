# News App
 
## Description
This is a basic news web application built with Go, using this [tutorial](https://github.com/Freshman-tech/news-demo). It fetches news articles from a News API and renders it on the webpage.

## Live Demo
You can view a live demo of this application [here](https://ebenezerraph-news-app.onrender.com/).

## Features
- Fetches news from multiple sources using a News API
- Searches for news articles using keywords.
- Simple and responsive UI

## Improvements
Although I followed the tutorial step-by-step, I later decided to go off-course and make some improvements. These improvements include:
- Overall Code Structure - With the help of Claude, I modified, re-formatted and refractored the entire code structure to improve performance.
- Logs - Made the logs more informative for better debugging.
- Error Handling - Created a function to handle errors effectively, and return the right error messages to the client.
- Pagination - Limited the number of pages to 5, due to the maximum number of requests that can currently be made to the API.
- Filtered Articles - Added a function to filter articles that might have been removed, from the API search results.
- Template and UI - Made changes to the HTML template and page style, to improve user interface and experience.

## Technologies
- Backend:
     - Golang
- Frontend:
     - HTML
     - CSS
