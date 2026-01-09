package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"ntp/handlers"
	"ntp/models"
	"ntp/services"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// 确保数据目录存在
	dataDir := "data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("创建数据目录失败: %v", err)
	}

	// 确保图标目录存在
	iconDir := filepath.Join("static", "icons")
	if err := os.MkdirAll(iconDir, 0755); err != nil {
		log.Fatalf("创建图标目录失败: %v", err)
	}

	// 初始化数据库
	dbPath := filepath.Join(dataDir, "bookmarks.db")
	if err := models.InitDB(dbPath); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	defer models.CloseDB()

	// 初始化 repositories
	bookmarkRepo := models.NewBookmarkRepository(models.DB)
	groupRepo := models.NewGroupRepository(models.DB)
	searchEngineRepo := models.NewSearchEngineRepository(models.DB)

	// 初始化 services
	iconService := services.NewIconService(iconDir, "/static/icons")

	// 初始化 handlers
	bookmarkHandler := handlers.NewBookmarkHandler(bookmarkRepo, groupRepo, iconService)
	groupHandler := handlers.NewGroupHandler(groupRepo)
	searchEngineHandler := handlers.NewSearchEngineHandler(searchEngineRepo)
	fetchHandler := handlers.NewFetchHandler()

	// 设置路由
	mux := http.NewServeMux()

	// 静态文件服务
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.Handle("/", http.FileServer(http.Dir("static")))

	// API 路由 - 使用明确的路径避免冲突
	mux.HandleFunc("/api/bookmarks", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			bookmarkHandler.List(w, r)
		case "POST":
			bookmarkHandler.Create(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/bookmarks/search", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			bookmarkHandler.Search(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/bookmarks/export", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			bookmarkHandler.Export(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/bookmarks/import", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			bookmarkHandler.Import(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/bookmarks/reorder", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			bookmarkHandler.Reorder(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// 书签详情 API (带 ID)
	mux.HandleFunc("/api/bookmark/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "PUT":
			bookmarkHandler.Update(w, r)
		case "DELETE":
			bookmarkHandler.Delete(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// 分组 API
	mux.HandleFunc("/api/groups", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			groupHandler.List(w, r)
		case "POST":
			groupHandler.Create(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/groups/reorder", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			groupHandler.Reorder(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// 分组详情 API (带 ID)
	mux.HandleFunc("/api/group/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "PUT":
			groupHandler.Update(w, r)
		case "DELETE":
			groupHandler.Delete(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// 搜索引擎 API
	mux.HandleFunc("/api/search-engines", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			searchEngineHandler.List(w, r)
		case "POST":
			searchEngineHandler.Create(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/search-engines/reorder", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			searchEngineHandler.Reorder(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// 搜索引擎详情 API (带 ID)
	mux.HandleFunc("/api/search-engine/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "PUT":
			searchEngineHandler.Update(w, r)
		case "DELETE":
			searchEngineHandler.Delete(w, r)
		case "POST":
			searchEngineHandler.SetDefault(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// 自动获取网站元数据
	mux.HandleFunc("/api/fetch-metadata", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			fetchHandler.FetchMetadata(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// 上传图标
	mux.HandleFunc("/api/upload-icon", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			bookmarkHandler.UploadIcon(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// 搜索跳转
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		engineID := r.URL.Query().Get("engine")

		var engine *models.SearchEngine
		var err error

		if engineID != "" {
			// 根据 ID 获取引擎
			engine, err = searchEngineRepo.GetByID(atoi64(engineID))
			if err != nil || engine == nil {
				engine, _ = searchEngineRepo.GetDefault()
			}
		} else {
			// 获取默认引擎
			engine, _ = searchEngineRepo.GetDefault()
		}

		if engine == nil {
			http.Error(w, "没有可用的搜索引擎", http.StatusInternalServerError)
			return
		}

		searchURL := fmt.Sprintf(engine.URL, query)
		http.Redirect(w, r, searchURL, http.StatusFound)
	})

	// 启动服务器
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := "0.0.0.0:" + port
	log.Printf("服务器启动在 http://0.0.0.0:%s", port)
	if err := http.ListenAndServe(addr, enableCORS(mux)); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}

// enableCORS 启用 CORS 中间件
func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// respondJSON 返回 JSON 响应
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError 返回错误响应
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// atoi64 字符串转 int64
func atoi64(s string) int64 {
	var i int64
	fmt.Sscanf(s, "%d", &i)
	return i
}
