package api

import (
	"log"
	"net/http"

	"github.com/aiwuxian/project-abyss/internal/models"
	"github.com/aiwuxian/project-abyss/internal/services"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	worldService  *services.WorldService
	storyService  *services.StoryService
	metaService   *services.MetaService
	llmService    *services.LLMService
	defaultConfig models.LLMConfig
}

func NewHandler(worldService *services.WorldService, storyService *services.StoryService,
	metaService *services.MetaService, llmService *services.LLMService) *Handler {
	return &Handler{
		worldService: worldService,
		storyService: storyService,
		metaService:  metaService,
		llmService:   llmService,
	}
}

// getCustomLLMService 从请求头获取自定义API配置并创建LLMService
func (h *Handler) getCustomLLMService(c *gin.Context) *services.LLMService {
	apiKey := c.GetHeader("X-Custom-API-Key")
	apiBase := c.GetHeader("X-Custom-API-Base")
	model := c.GetHeader("X-Custom-API-Model")

	// 如果没有自定义配置，返回默认服务
	if apiKey == "" {
		return h.llmService
	}

	// 创建自定义配置
	config := models.LLMConfig{
		Provider:    "openai",
		APIKey:      apiKey,
		APIBase:     apiBase,
		Model:       model,
		Temperature: 0.7,
		MaxTokens:   2000,
	}

	// 创建并返回新的LLMService实例
	return services.NewLLMService(config)
}

// CreateCharacter 创建角色（手动创建）
func (h *Handler) CreateCharacter(c *gin.Context) {
	var req struct {
		Name           string         `json:"name" binding:"required"`
		Gender         string         `json:"gender" binding:"required"`
		Age            int            `json:"age" binding:"required"`
		Appearance     string         `json:"appearance"`
		Personality    string         `json:"personality"`
		Background     string         `json:"background"`
		BaseAttributes map[string]int `json:"base_attributes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	char := &models.Character{
		Name:           req.Name,
		Gender:         req.Gender,
		Age:            req.Age,
		Appearance:     req.Appearance,
		Personality:    req.Personality,
		Background:     req.Background,
		BaseAttributes: req.BaseAttributes,
	}

	char, err := h.metaService.CreateCharacter(char)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, char)
}

// GenerateCharacter AI自动生成角色
func (h *Handler) GenerateCharacter(c *gin.Context) {
	var req struct {
		Name   string `json:"name" binding:"required"`
		Gender string `json:"gender" binding:"required"`
		Age    int    `json:"age" binding:"required"`
		Prompt string `json:"prompt"` // 可选的额外提示
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 使用自定义LLM配置（如果有）
	llmService := h.getCustomLLMService(c)

	char, err := llmService.GenerateCharacter(c.Request.Context(), req.Name, req.Gender, req.Age, req.Prompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 保存到数据库
	char, err = h.metaService.CreateCharacter(char)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, char)
}

// GetCharacter 获取角色信息
func (h *Handler) GetCharacter(c *gin.Context) {
	id := c.Param("id")

	char, err := h.metaService.GetCharacter(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
		return
	}

	c.JSON(http.StatusOK, char)
}

// ListCharacters 获取所有角色列表
func (h *Handler) ListCharacters(c *gin.Context) {
	characters, err := h.metaService.GetAllCharacters()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, characters)
}

// ParseSegment 解析小说段落，创建世界
func (h *Handler) ParseSegment(c *gin.Context) {
	var req struct {
		SegmentText string `json:"segment_text" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "段落文本不能为空"})
		return
	}

	// 使用自定义LLM配置（如果有）
	llmService := h.getCustomLLMService(c)

	// 创建临时的worldService使用自定义LLM
	worldService := services.NewWorldService(h.worldService.GetStorage(), llmService)

	world, err := worldService.CreateWorldFromSegment(c.Request.Context(), req.SegmentText)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, world)
}

// StartStory 开始新故事
func (h *Handler) StartStory(c *gin.Context) {
	var req struct {
		CharacterID string `json:"character_id" binding:"required"`
		WorldID     string `json:"world_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 使用自定义LLM配置（如果有）
	llmService := h.getCustomLLMService(c)

	// 创建临时的storyService使用自定义LLM
	storage, ruleEngine, metaService := h.storyService.GetDependencies()
	storyService := services.NewStoryService(storage, llmService, ruleEngine, metaService)

	story, scene, err := storyService.StartStory(c.Request.Context(), req.CharacterID, req.WorldID)
	if err != nil {
		log.Printf("❌ StartStory失败: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("✅ Story创建成功, ID: %s\n", story.ID)

	// 获取角色状态
	charState, err := h.metaService.GetCharacterState(req.CharacterID, req.WorldID)
	if err != nil {
		log.Printf("❌ GetCharacterState失败: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取角色状态失败: " + err.Error()})
		return
	}

	if charState == nil {
		log.Println("❌ charState为nil")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "角色状态不存在"})
		return
	}

	log.Printf("✅ 角色状态获取成功, HP: %d, SAN: %d\n", charState.HP, charState.SAN)

	c.JSON(http.StatusOK, gin.H{
		"story":      story,
		"scene":      scene,
		"char_state": charState,
	})
}

// TakeAction 执行行动
func (h *Handler) TakeAction(c *gin.Context) {
	var req struct {
		StoryID string        `json:"story_id" binding:"required"`
		Action  models.Action `json:"action" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 使用自定义LLM配置（如果有）
	llmService := h.getCustomLLMService(c)

	// 创建临时的storyService使用自定义LLM
	storage, ruleEngine, metaService := h.storyService.GetDependencies()
	storyService := services.NewStoryService(storage, llmService, ruleEngine, metaService)

	result, err := storyService.ProcessAction(c.Request.Context(), req.StoryID, req.Action)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 获取更新后的故事状态
	story, _ := storyService.GetStory(req.StoryID)

	c.JSON(http.StatusOK, gin.H{
		"result": result,
		"story":  story,
	})
}

// GetStory 获取故事状态
func (h *Handler) GetStory(c *gin.Context) {
	id := c.Param("id")

	story, err := h.storyService.GetStory(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "故事不存在"})
		return
	}

	// 获取场景和角色状态
	scene, _ := h.worldService.GetWorld(story.WorldID)
	charState, _ := h.metaService.GetCharacterState(story.CharacterID, story.WorldID)

	c.JSON(http.StatusOK, gin.H{
		"story":      story,
		"world":      scene,
		"char_state": charState,
	})
}

// UndoTurn 回退到上一个回合
func (h *Handler) UndoTurn(c *gin.Context) {
	var req struct {
		StoryID string `json:"story_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	story, err := h.storyService.UndoTurn(req.StoryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 获取更新后的角色状态
	charState, _ := h.metaService.GetCharacterState(story.CharacterID, story.WorldID)

	c.JSON(http.StatusOK, gin.H{
		"story":      story,
		"char_state": charState,
	})
}

// SaveGame 保存游戏
func (h *Handler) SaveGame(c *gin.Context) {
	var req struct {
		StoryID     string `json:"story_id" binding:"required"`
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	save, err := h.storyService.CreateSaveGame(req.StoryID, req.Name, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, save)
}

// ListSaves 列出存档
func (h *Handler) ListSaves(c *gin.Context) {
	characterID := c.Query("character_id")
	if characterID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "需要character_id参数"})
		return
	}

	saves, err := h.storyService.ListSaveGames(characterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"saves": saves})
}

// LoadGame 读取存档
func (h *Handler) LoadGame(c *gin.Context) {
	var req struct {
		StoryID string `json:"story_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	story, scene, charState, err := h.storyService.LoadStory(c.Request.Context(), req.StoryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"story":      story,
		"scene":      scene,
		"char_state": charState,
	})
}
