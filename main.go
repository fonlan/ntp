package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"ntp/handlers"
	"ntp/i18n"
	"ntp/middleware"
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

	iconDir := filepath.Join(dataDir, "icons")
	if err := os.MkdirAll(iconDir, 0755); err != nil {
		log.Fatalf("创建图标目录失败: %v", err)
	}

	// 初始化数据库
	dbPath := filepath.Join(dataDir, "bookmarks.db")
	if err := models.InitDB(dbPath); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	defer models.CloseDB()

	// 初始化 repositories 和 services
	bookmarkRepo := models.NewBookmarkRepository(models.DB)
	groupRepo := models.NewGroupRepository(models.DB)
	searchEngineRepo := models.NewSearchEngineRepository(models.DB)
	iconService := services.NewIconService(iconDir, "/data/icons")

	// 初始化 handlers
	bookmarkHandler := handlers.NewBookmarkHandler(bookmarkRepo, groupRepo, iconService)
	groupHandler := handlers.NewGroupHandler(groupRepo)
	searchEngineHandler := handlers.NewSearchEngineHandler(searchEngineRepo)
	fetchHandler := handlers.NewFetchHandler()

	// 加载国际化翻译文件
	if err := i18n.LoadTranslations("i18n"); err != nil {
		log.Fatalf("加载翻译文件失败: %v", err)
	}

	// 设置路由
	mux := http.NewServeMux()

	// 静态文件服务
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.Handle("/data/icons/", http.StripPrefix("/data/icons/", http.FileServer(http.Dir(iconDir))))
	mux.Handle("/", http.FileServer(http.Dir("static")))

	// API 路由
	registerAPIRoutes(mux, bookmarkHandler, groupHandler, searchEngineHandler, fetchHandler)

	// 搜索跳转
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		engineID := r.URL.Query().Get("engine")

		var engine *models.SearchEngine
		var err error

		if engineID != "" {
			id, _ := strconv.ParseInt(engineID, 10, 64)
			engine, err = searchEngineRepo.GetByID(id)
			if err != nil || engine == nil {
				engine, _ = searchEngineRepo.GetDefault()
			}
		} else {
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

	handler := middleware.LocaleMiddleware(enableCORS(mux))
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}

// registerAPIRoutes 注册所有 API 路由
func registerAPIRoutes(mux *http.ServeMux, bh *handlers.BookmarkHandler, gh *handlers.GroupHandler, sh *handlers.SearchEngineHandler, fh *handlers.FetchHandler) {
	// 书签路由
	mux.HandleFunc("/api/bookmarks", methodRouter(map[string]http.HandlerFunc{
		"GET":  bh.List,
		"POST": bh.Create,
	}))
	mux.HandleFunc("/api/bookmarks/search", methodRouter(map[string]http.HandlerFunc{
		"GET": bh.Search,
	}))
	mux.HandleFunc("/api/bookmarks/export", methodRouter(map[string]http.HandlerFunc{
		"GET": bh.Export,
	}))
	mux.HandleFunc("/api/bookmarks/import", methodRouter(map[string]http.HandlerFunc{
		"POST": bh.Import,
	}))
	mux.HandleFunc("/api/bookmarks/reorder", methodRouter(map[string]http.HandlerFunc{
		"POST": bh.Reorder,
	}))
	mux.HandleFunc("/api/bookmark/", methodRouter(map[string]http.HandlerFunc{
		"PUT":    bh.Update,
		"DELETE": bh.Delete,
	}))

	// 分组路由
	mux.HandleFunc("/api/groups", methodRouter(map[string]http.HandlerFunc{
		"GET":  gh.List,
		"POST": gh.Create,
	}))
	mux.HandleFunc("/api/groups/reorder", methodRouter(map[string]http.HandlerFunc{
		"POST": gh.Reorder,
	}))
	mux.HandleFunc("/api/groups/", methodRouter(map[string]http.HandlerFunc{
		"PUT":    gh.Update,
		"DELETE": gh.Delete,
	}))

	// 搜索引擎路由
	mux.HandleFunc("/api/search-engines", methodRouter(map[string]http.HandlerFunc{
		"GET":  sh.List,
		"POST": sh.Create,
	}))
	mux.HandleFunc("/api/search-engines/reorder", methodRouter(map[string]http.HandlerFunc{
		"POST": sh.Reorder,
	}))
	mux.HandleFunc("/api/search-engine/", methodRouter(map[string]http.HandlerFunc{
		"PUT":    sh.Update,
		"DELETE": sh.Delete,
		"POST":   sh.SetDefault,
	}))

	// 其他 API
	mux.HandleFunc("/api/fetch-metadata", methodRouter(map[string]http.HandlerFunc{
		"POST": fh.FetchMetadata,
	}))
	mux.HandleFunc("/api/upload-icon", methodRouter(map[string]http.HandlerFunc{
		"POST": bh.UploadIcon,
	}))
}

// methodRouter 路由方法选择器
func methodRouter(handlers map[string]http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handler, ok := handlers[r.Method]
		if !ok {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handler(w, r)
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
