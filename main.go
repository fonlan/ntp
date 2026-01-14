package main

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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

	// 为现有搜索引擎填充图标（如果 icon_path 为空）
	initializeSearchEngineIcons(searchEngineRepo, iconService)

	// 初始化 handlers
	bookmarkHandler := handlers.NewBookmarkHandler(bookmarkRepo, groupRepo, iconService)
	groupHandler := handlers.NewGroupHandler(groupRepo)
	searchEngineHandler := handlers.NewSearchEngineHandler(searchEngineRepo, iconService)
	fetchHandler := handlers.NewFetchHandler()
	authHandler := handlers.NewAuthHandler()

	// 加载国际化翻译文件
	if err := i18n.LoadTranslations("i18n"); err != nil {
		log.Fatalf("加载翻译文件失败: %v", err)
	}

	// 设置路由
	mux := http.NewServeMux()

	// 登录页面（公开访问，无需认证）
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/login" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "static/login.html")
	})

	// 认证 API（公开访问，无需认证）
	mux.HandleFunc("/api/login", methodRouter(map[string]http.HandlerFunc{
		"POST": authHandler.Login,
	}))
	mux.HandleFunc("/api/logout", methodRouter(map[string]http.HandlerFunc{
		"POST": authHandler.Logout,
	}))
	mux.HandleFunc("/api/auth/check", methodRouter(map[string]http.HandlerFunc{
		"GET": authHandler.CheckAuth,
	}))

	// 静态文件服务（使用 Cache-Control 替代 Expires）
	mux.Handle("/static/", http.StripPrefix("/static/", cacheControlWrapper(http.FileServer(http.Dir("static")))))
	mux.Handle("/data/icons/", http.StripPrefix("/data/icons/", cacheControlWrapper(http.FileServer(http.Dir(iconDir)))))

	// 首页和 API 路由（需要认证）
	mux.Handle("/", middleware.AuthMiddleware(cacheControlWrapper(http.FileServer(http.Dir("static")))))
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

		// 使用 {q} 占位符替换，并对 query 进行 URL 编码，防止注入攻击
		searchURL := strings.ReplaceAll(engine.URL, "{q}", url.QueryEscape(query))
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
	// 书签路由（需要认证）
	mux.Handle("/api/bookmarks", middleware.APIAuthMiddleware(methodRouter(map[string]http.HandlerFunc{
		"GET":  bh.List,
		"POST": bh.Create,
	})))
	mux.Handle("/api/bookmarks/search", middleware.APIAuthMiddleware(methodRouter(map[string]http.HandlerFunc{
		"GET": bh.Search,
	})))
	mux.Handle("/api/bookmarks/export", middleware.APIAuthMiddleware(methodRouter(map[string]http.HandlerFunc{
		"GET": bh.Export,
	})))
	mux.Handle("/api/bookmarks/import", middleware.APIAuthMiddleware(methodRouter(map[string]http.HandlerFunc{
		"POST": bh.Import,
	})))
	mux.Handle("/api/bookmarks/reorder", middleware.APIAuthMiddleware(methodRouter(map[string]http.HandlerFunc{
		"POST": bh.Reorder,
	})))
	mux.Handle("/api/bookmark/", middleware.APIAuthMiddleware(methodRouter(map[string]http.HandlerFunc{
		"PUT":    bh.Update,
		"DELETE": bh.Delete,
	})))

	// 分组路由（需要认证）
	mux.Handle("/api/groups", middleware.APIAuthMiddleware(methodRouter(map[string]http.HandlerFunc{
		"GET":  gh.List,
		"POST": gh.Create,
	})))
	mux.Handle("/api/groups/reorder", middleware.APIAuthMiddleware(methodRouter(map[string]http.HandlerFunc{
		"POST": gh.Reorder,
	})))
	mux.Handle("/api/groups/", middleware.APIAuthMiddleware(methodRouter(map[string]http.HandlerFunc{
		"PUT":    gh.Update,
		"DELETE": gh.Delete,
	})))

	// 搜索引擎路由（需要认证）
	mux.Handle("/api/search-engines", middleware.APIAuthMiddleware(methodRouter(map[string]http.HandlerFunc{
		"GET":  sh.List,
		"POST": sh.Create,
	})))
	mux.Handle("/api/search-engines/reorder", middleware.APIAuthMiddleware(methodRouter(map[string]http.HandlerFunc{
		"POST": sh.Reorder,
	})))
	mux.Handle("/api/search-engine/", middleware.APIAuthMiddleware(methodRouter(map[string]http.HandlerFunc{
		"PUT":    sh.Update,
		"DELETE": sh.Delete,
		"POST":   sh.SetDefault,
	})))

	// 其他 API（需要认证）
	mux.Handle("/api/fetch-metadata", middleware.APIAuthMiddleware(methodRouter(map[string]http.HandlerFunc{
		"POST": fh.FetchMetadata,
	})))
	mux.Handle("/api/upload-icon", middleware.APIAuthMiddleware(methodRouter(map[string]http.HandlerFunc{
		"POST": bh.UploadIcon,
	})))
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

		// 安全响应头
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Content-Security-Policy 替代 X-Frame-Options
		// frame-ancestors 'self' 只允许同源页面嵌入（等同于 X-Frame-Options: SAMEORIGIN）
		w.Header().Set("Content-Security-Policy",
			"frame-ancestors 'self'; "+
				"default-src 'self'; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data: https: http:; "+
				"font-src 'self' data:")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// cacheControlWrapper 为静态文件添加缓存控制头，替代 Expires
func cacheControlWrapper(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 使用响应包装器来覆盖默认的 Expires 头
		wrapper := &cacheControlResponseWriter{ResponseWriter: w, Request: r}
		h.ServeHTTP(wrapper, r)
	})
}

// cacheControlResponseWriter 包装 ResponseWriter 以控制缓存头
type cacheControlResponseWriter struct {
	http.ResponseWriter
	Request    *http.Request
	headersSet bool
}

func (w *cacheControlResponseWriter) WriteHeader(statusCode int) {
	if !w.headersSet {
		// 根据文件类型设置不同的缓存策略
		path := w.Request.URL.Path

		var cacheControl string
		if strings.HasSuffix(path, ".html") {
			// HTML 文件：不缓存，确保始终获取最新版本
			cacheControl = "no-cache, no-store, must-revalidate"
		} else if strings.HasSuffix(path, ".css") || strings.HasSuffix(path, ".js") ||
			strings.HasSuffix(path, ".png") || strings.HasSuffix(path, ".jpg") ||
			strings.HasSuffix(path, ".jpeg") || strings.HasSuffix(path, ".gif") ||
			strings.HasSuffix(path, ".svg") || strings.HasSuffix(path, ".ico") ||
			strings.HasSuffix(path, ".woff") || strings.HasSuffix(path, ".woff2") {
			// 静态资源：长期缓存（1 年）
			cacheControl = "public, max-age=31536000, immutable"
		} else {
			// 其他文件：默认缓存策略（1 天）
			cacheControl = "public, max-age=86400"
		}

		// 设置 Cache-Control，移除 Expires
		w.Header().Set("Cache-Control", cacheControl)
		w.Header().Del("Expires")
		w.headersSet = true
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

// initializeSearchEngineIcons 为现有搜索引擎初始化图标
func initializeSearchEngineIcons(repo *models.SearchEngineRepository, iconService *services.IconService) {
	engines, err := repo.GetAll()
	if err != nil {
		log.Printf("获取搜索引擎列表失败: %v", err)
		return
	}

	for _, engine := range engines {
		needDownload := false

		// 检查是否需要下载图标
		if engine.IconPath == nil {
			// icon_path 为空，需要下载
			needDownload = true
		} else {
			// icon_path 不为空，检查文件是否存在
			iconFilePath := filepath.Join("data/icons", strings.TrimPrefix(*engine.IconPath, "/data/icons/"))
			if _, err := os.Stat(iconFilePath); os.IsNotExist(err) {
				// 文件不存在，需要重新下载
				log.Printf("搜索引擎 %s 的图标文件不存在，需要重新下载", engine.Name)
				needDownload = true
			}
		}

		if needDownload {
			iconPath, err := iconService.DownloadFavicon(engine.URL)
			if err != nil {
				log.Printf("下载搜索引擎 %s 图标失败: %v", engine.Name, err)
				continue
			}

			// 更新搜索引擎的 icon_path
			if iconPath != "" {
				engine.IconPath = &iconPath
				if err := repo.Update(&engine); err != nil {
					log.Printf("更新搜索引擎 %s 图标路径失败: %v", engine.Name, err)
				} else {
					log.Printf("成功为搜索引擎 %s 下载并保存图标", engine.Name)
				}
			}
		}
	}
}
