package searchdb

import "time"

type Document struct {
	ID      string    `json:"id"`
	Path    string    `json:"path"`
	Name    string    `json:"name"`
	Content string    `json:"content"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
}

type Result struct {
	ID      string  `json:"id"`
	Path    string  `json:"path"`
	Name    string  `json:"name"`
	Score   float64 `json:"score"`
	Size    int64   `json:"size"`
	ModTime string  `json:"mod_time"`
}

type Response struct {
	Results    []Result `json:"results"`
	Total      uint64   `json:"total"`
	MaxScore   float64  `json:"max_score"`
	SearchTime string   `json:"search_time"`
}
