package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"

	"github.com/aiwuxian/project-abyss/internal/api"
	"github.com/aiwuxian/project-abyss/internal/models"
	"github.com/aiwuxian/project-abyss/internal/services"
	"github.com/aiwuxian/project-abyss/internal/storage"
)

func main() {
	// 加载配置
	config, err := loadConfig("config.yml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化数据库
	store, err := storage.New(config.Database.Path)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer store.Close()

	// 初始化服务
	llmService := services.NewLLMService(config.LLM)
	ruleEngine := services.NewRuleEngine()
	metaService := services.NewMetaService(store, config.Game)
	worldService := services.NewWorldService(store, llmService)
	storyService := services.NewStoryService(store, llmService, ruleEngine, metaService)

	// 初始化API处理器
	handler := api.NewHandler(worldService, storyService, metaService, llmService)

	// 设置Gin路由
	r := gin.Default()

	// 静态文件
	r.Static("/web", "./web")
	r.GET("/", func(c *gin.Context) {
		c.Redirect(302, "/web/index.html")
	})

	// API路由
	apiGroup := r.Group("/api")
	{
		// 角色相关
		apiGroup.POST("/characters", handler.CreateCharacter)
		apiGroup.POST("/characters/generate", handler.GenerateCharacter)
		apiGroup.GET("/characters", handler.ListCharacters)
		apiGroup.GET("/characters/:id", handler.GetCharacter)

		// 世界相关
		apiGroup.POST("/worlds/parse", handler.ParseSegment)

		// 故事相关
		apiGroup.POST("/stories/start", handler.StartStory)
		apiGroup.GET("/stories/:id", handler.GetStory)
		apiGroup.POST("/stories/action", handler.TakeAction)
		apiGroup.POST("/stories/undo", handler.UndoTurn)

		// 存档相关
		apiGroup.POST("/saves", handler.SaveGame)
		apiGroup.GET("/saves", handler.ListSaves)
		apiGroup.POST("/saves/load", handler.LoadGame)
	}

	// 启动服务器
	addr := fmt.Sprintf("%s:%s", config.Server.Host, config.Server.Port)
	log.Printf("🎮 Project Abyss 启动成功！访问 http://localhost:%s", config.Server.Port)
	log.Printf("📖 准备开始你的无限流冒险...")

	if err := r.Run(addr); err != nil {
		log.Fatalf("启动服务器失败: %v", err)
	}
}

func loadConfig(path string) (*models.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config models.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
