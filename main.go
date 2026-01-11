package main

import (
	"fmt"
	"log"
	"net/http"
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

	// 静态文件服务
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.Handle("/data/icons/", http.StripPrefix("/data/icons/", http.FileServer(http.Dir(iconDir))))

	// 首页和 API 路由（需要认证）
	mux.Handle("/", middleware.AuthMiddleware(http.FileServer(http.Dir("static"))))
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

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
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
