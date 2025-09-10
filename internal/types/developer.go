package types

import "time"

type SystemInfo struct {
	AppName        string    `json:"app_name"`
	Version        string    `json:"version"`
	Environment    string    `json:"environment"`
	StartTime      time.Time `json:"start_time"`
	Uptime         string    `json:"uptime"`
	GoVersion      string    `json:"go_version"`
	Architecture   string    `json:"architecture"`
	OS             string    `json:"os"`
	PID            int       `json:"pid"`
	DBPath         string    `json:"db_path"`
	Port           string    `json:"port"`
}

type DatabaseStats struct {
	ProductCount   int64  `json:"product_count"`
	CategoryCount  int64  `json:"category_count"`
	OrderCount     int64  `json:"order_count"`
	UserCount      int64  `json:"user_count"`
	DatabaseSize   string `json:"database_size"`
}

type MemoryStats struct {
	Alloc        uint64  `json:"alloc"`
	TotalAlloc   uint64  `json:"total_alloc"`
	Sys          uint64  `json:"sys"`
	NumGC        uint32  `json:"num_gc"`
	Goroutines   int     `json:"goroutines"`
	AllocMB      float64 `json:"alloc_mb"`
	SysMB        float64 `json:"sys_mb"`
}