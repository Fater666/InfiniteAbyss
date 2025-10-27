package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aiwuxian/project-abyss/internal/models"
	"github.com/aiwuxian/project-abyss/internal/storage"
	"github.com/google/uuid"
)

type WorldService struct {
	storage *storage.Storage
	llm     *LLMService
}

func NewWorldService(storage *storage.Storage, llm *LLMService) *WorldService {
	return &WorldService{
		storage: storage,
		llm:     llm,
	}
}

// GetStorage 返回storage实例（用于创建临时服务）
func (ws *WorldService) GetStorage() *storage.Storage {
	return ws.storage
}

// CreateWorldFromSegment 从小说段落创建世界
func (ws *WorldService) CreateWorldFromSegment(ctx context.Context, segmentText string) (*models.World, error) {
	// 使用LLM解析段落
	world, err := ws.llm.ParseSegment(ctx, segmentText)
	if err != nil {
		return nil, fmt.Errorf("解析段落失败: %w", err)
	}

	// 生成原小说摘要（1000字内）
	if segmentText != "" {
		summary, err := ws.llm.GenerateOriginalSummary(ctx, segmentText)
		if err != nil {
			// 如果生成摘要失败，记录错误但不影响主流程
			log.Printf("⚠️ 生成原小说摘要失败: %v\n", err)
		} else {
			world.OriginalSummary = summary
		}
	}

	// 生成ID和时间戳
	world.ID = uuid.New().String()
	world.CreatedAt = time.Now()

	// 为每个NPC生成ID
	for i := range world.NPCs {
		world.NPCs[i].ID = uuid.New().String()
	}

	// 保存到数据库
	if err := ws.storage.CreateWorld(world); err != nil {
		return nil, fmt.Errorf("保存世界失败: %w", err)
	}

	return world, nil
}

// GetWorld 获取世界信息
func (ws *WorldService) GetWorld(worldID string) (*models.World, error) {
	return ws.storage.GetWorld(worldID)
}

// GenerateStartScene 为世界生成开场场景
func (ws *WorldService) GenerateStartScene(ctx context.Context, world *models.World, character *models.Character) (*models.Scene, error) {
	scene, err := ws.llm.GenerateScene(ctx, world, character)
	if err != nil {
		return nil, err
	}

	scene.ID = uuid.New().String()

	// 保存场景
	if err := ws.storage.CreateScene(scene); err != nil {
		return nil, fmt.Errorf("保存场景失败: %w", err)
	}

	return scene, nil
}
