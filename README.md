# News App
This is a basic news web application built with Go, using this [tutorial](https://github.com/Freshman-tech/news-demo). It fetches news articles from a News API and renders it on the webpage.

## Features
- Fetches news from multiple sources using a News API.
- Searches for news articles using keywords.
- Features a simple and responsive user interface.

## Improvements  
Although I followed the tutorial step by step, I later decided to deviate from it and implement some improvements. These improvements include:  
- Overall Code Structure: With the help of Claude, I modified, reformatted, and refactored the entire code structure to enhance performance.  
- Logs: I made the logs more informative to facilitate better debugging.  
- Error Handling: I created a function to handle errors effectively and return the appropriate error messages to the client.  
- Pagination: I limited the number of pages to 5 due to the maximum number of requests that can currently be made to the API.  
- Filtered Articles: I added a function to filter out articles that may have been removed from the API search results.  
- Template and UI: I made changes to the HTML template and page style to improve the user interface and experience.

## Technologies
[![technologies](https://skillicons.dev/icons?i=go,html,css&theme=light)](https://skillicons.dev)
