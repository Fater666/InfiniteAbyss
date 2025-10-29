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

type StoryService struct {
	storage    *storage.Storage
	llm        *LLMService
	ruleEngine *RuleEngine
	meta       *MetaService
}

func NewStoryService(storage *storage.Storage, llm *LLMService,
	ruleEngine *RuleEngine, meta *MetaService) *StoryService {
	return &StoryService{
		storage:    storage,
		llm:        llm,
		ruleEngine: ruleEngine,
		meta:       meta,
	}
}

// GetDependencies 返回依赖项（用于创建临时服务）
func (ss *StoryService) GetDependencies() (*storage.Storage, *RuleEngine, *MetaService) {
	return ss.storage, ss.ruleEngine, ss.meta
}

// StartStory 开始新的故事
func (ss *StoryService) StartStory(ctx context.Context, characterID, worldID string) (*models.StoryState, *models.Scene, error) {
	// 获取世界信息
	world, err := ss.storage.GetWorld(worldID)
	if err != nil {
		return nil, nil, fmt.Errorf("获取世界失败: %w", err)
	}

	// 获取角色
	char, err := ss.storage.GetCharacter(characterID)
	if err != nil {
		return nil, nil, fmt.Errorf("获取角色失败: %w", err)
	}

	// 初始化角色状态
	if _, err := ss.meta.InitCharacterInWorld(characterID, worldID, world); err != nil {
		return nil, nil, fmt.Errorf("初始化角色状态失败: %w", err)
	}

	// 生成开场场景
	scene, err := ss.llm.GenerateScene(ctx, world, char)
	if err != nil {
		return nil, nil, fmt.Errorf("生成场景失败: %w", err)
	}
	scene.ID = uuid.New().String()

	if err := ss.storage.CreateScene(scene); err != nil {
		return nil, nil, fmt.Errorf("保存场景失败: %w", err)
	}

	// 选择起始剧情节点（选择第一个可玩节点）
	var startPlotNodeID string
	if len(world.PlotLines) > 0 {
		// 优先选择标记为可玩的节点，order最小的
		for _, node := range world.PlotLines {
			if node.IsPlayable {
				startPlotNodeID = node.ID
				break
			}
		}
		// 如果没有可玩节点，选择第一个
		if startPlotNodeID == "" {
			startPlotNodeID = world.PlotLines[0].ID
		}
	}

	// 创建故事状态
	story := &models.StoryState{
		ID:                uuid.New().String(),
		CharacterID:       characterID,
		WorldID:           worldID,
		SceneID:           scene.ID,
		CurrentPlotNodeID: startPlotNodeID,
		PlotProgress:      0.0,
		Turn:              0,
		Narrative:         []models.NarrativeLog{},
		Status:            "active",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// 添加开场叙事
	story.Narrative = append(story.Narrative, models.NarrativeLog{
		Turn:      0,
		Type:      "system",
		Content:   fmt.Sprintf("你进入了【%s】\n\n%s", scene.Name, scene.Description),
		Timestamp: time.Now(),
	})

	if err := ss.storage.CreateStoryState(story); err != nil {
		return nil, nil, fmt.Errorf("保存故事状态失败: %w", err)
	}

	return story, scene, nil
}

// ProcessAction 处理玩家行动
func (ss *StoryService) ProcessAction(ctx context.Context, storyID string, action models.Action) (*models.ActionResult, error) {
	// 获取故事状态
	story, err := ss.storage.GetStoryState(storyID)
	if err != nil {
		return nil, fmt.Errorf("获取故事状态失败: %w", err)
	}

	if story.Status != "active" {
		return nil, fmt.Errorf("故事已结束")
	}

	// 获取世界信息
	world, err := ss.storage.GetWorld(story.WorldID)
	if err != nil {
		return nil, fmt.Errorf("获取世界失败: %w", err)
	}

	// 获取场景
	scene, err := ss.storage.GetScene(story.SceneID)
	if err != nil {
		return nil, fmt.Errorf("获取场景失败: %w", err)
	}

	// 获取角色信息
	character, err := ss.storage.GetCharacter(story.CharacterID)
	if err != nil {
		return nil, fmt.Errorf("获取角色失败: %w", err)
	}

	// 获取角色状态
	charState, err := ss.meta.GetCharacterState(story.CharacterID, story.WorldID)
	if err != nil {
		return nil, fmt.Errorf("获取角色状态失败: %w", err)
	}

	// 计算检定难度
	difficulty := ss.ruleEngine.CalculateDifficulty(scene.Type, action.Type)

	// 选择合适的属性
	attribute := ss.selectAttribute(action.Type, charState.Attributes)

	// 执行检定
	diceRoll := ss.ruleEngine.Check(attribute, difficulty)

	log.Println("🎲 ========================================")
	log.Printf("🎲 [检定] 行动: %s\n", action.Content)
	log.Printf("🎲 属性加成: +%d | 目标难度: %d\n", attribute, difficulty)
	log.Printf("🎲 投掷结果: %d + %d = %d\n", diceRoll.Result, diceRoll.Modifier, diceRoll.Result+diceRoll.Modifier)
	if diceRoll.Critical {
		if diceRoll.Success {
			log.Println("🎲 ⭐⭐⭐ 大成功！⭐⭐⭐")
		} else {
			log.Println("🎲 💀💀💀 大失败！💀💀💀")
		}
	} else if diceRoll.Success {
		log.Println("🎲 ✅ 成功！")
	} else {
		log.Println("🎲 ❌ 失败...")
	}
	log.Println("🎲 ========================================")
	log.Println()

	// 生成叙事
	narrative, err := ss.llm.NarrateResult(ctx, world, character, scene, action, diceRoll, story.Narrative)
	if err != nil {
		narrative = fmt.Sprintf("你尝试了%s，结果%s", action.Content,
			map[bool]string{true: "成功", false: "失败"}[diceRoll.Success])
	}

	// 保存当前状态快照（用于回退）
	snapshot := models.StateSnapshot{
		Turn:      story.Turn,
		Narrative: append([]models.NarrativeLog{}, story.Narrative...),
		CharState: *charState,
		Timestamp: time.Now(),
	}
	story.Snapshots = append(story.Snapshots, snapshot)

	// 记录日志
	story.Turn++
	story.Narrative = append(story.Narrative, models.NarrativeLog{
		Turn:      story.Turn,
		Type:      "action",
		Content:   action.Content,
		Timestamp: time.Now(),
	})
	story.Narrative = append(story.Narrative, models.NarrativeLog{
		Turn:      story.Turn,
		Type:      "result",
		Content:   narrative,
		DiceRoll:  diceRoll,
		Timestamp: time.Now(),
	})

	// 计算状态变化
	changes := ss.calculateChanges(scene, action, diceRoll)

	log.Println("💫 [状态变化]")
	if changes.HPChange != 0 {
		log.Printf("   HP: %+d\n", changes.HPChange)
	}
	if changes.SANChange != 0 {
		log.Printf("   理智值: %+d\n", changes.SANChange)
	}
	if changes.XPGain > 0 {
		log.Printf("   经验值: +%d\n", changes.XPGain)
	}
	if len(changes.ItemsGained) > 0 {
		log.Printf("   获得道具: %d 个\n", len(changes.ItemsGained))
	}
	if len(changes.TraitsGained) > 0 {
		log.Printf("   获得特质: %v\n", changes.TraitsGained)
	}
	log.Println()

	// 应用变化
	if err := ss.meta.ApplyChanges(story.CharacterID, story.WorldID, changes); err != nil {
		return nil, fmt.Errorf("应用状态变化失败: %w", err)
	}

	// 评估剧情推进
	if story.CurrentPlotNodeID != "" {
		if err := ss.evaluatePlotProgress(ctx, story, action, narrative); err != nil {
			log.Printf("⚠️ 评估剧情推进失败: %v\n", err)
			// 不影响主流程，继续执行
		}
	}

	// 检查场景是否结束
	sceneEnd := ss.checkSceneEnd(scene, story, charState, changes)
	if sceneEnd {
		story.Status = "completed"
	}

	story.UpdatedAt = time.Now()
	if err := ss.storage.UpdateStoryState(story); err != nil {
		return nil, fmt.Errorf("更新故事状态失败: %w", err)
	}

	// 重新获取角色状态以获取最新数据
	charState, _ = ss.meta.GetCharacterState(story.CharacterID, story.WorldID)

	// 生成下一步选项
	var nextOptions []models.Option
	if !sceneEnd {
		nextOptions, err = ss.llm.GenerateOptions(ctx, world, scene, narrative, story.Narrative, charState)
		if err != nil {
			// 如果生成失败，提供默认选项
			nextOptions = ss.getDefaultOptions()
		}
	}

	return &models.ActionResult{
		Success:     diceRoll.Success,
		Narrative:   narrative,
		DiceRoll:    diceRoll,
		Changes:     changes,
		NextOptions: nextOptions,
		SceneEnd:    sceneEnd,
	}, nil
}

// selectAttribute 根据行动类型选择属性
func (ss *StoryService) selectAttribute(actionType string, attributes map[string]int) int {
	attrMap := map[string]string{
		"attack":      "strength",
		"move":        "dexterity",
		"sneak":       "dexterity",
		"talk":        "charisma",
		"persuade":    "charisma",
		"investigate": "perception",
		"use_item":    "intelligence",
	}

	attrName, ok := attrMap[actionType]
	if !ok {
		attrName = "intelligence"
	}

	return attributes[attrName]
}

// calculateChanges 计算状态变化
func (ss *StoryService) calculateChanges(scene *models.Scene, _ models.Action, diceRoll *models.DiceRoll) models.StateChanges {
	changes := models.StateChanges{}

	// 计算经验值
	changes.XPGain = ss.ruleEngine.CalculateXPGain(diceRoll.Target, diceRoll.Success)

	// 根据场景类型和结果计算HP/SAN变化
	if scene.Type == "combat" {
		if !diceRoll.Success {
			damage := ss.ruleEngine.CalculateDamage(5, diceRoll.Critical)
			changes.HPChange = -damage
		}
	}

	if scene.Type == "horror" || len(scene.Threats) > 0 {
		if !diceRoll.Success {
			changes.SANChange = -ss.ruleEngine.RollDice(6)
		}
	}

	// 大成功可能获得额外奖励
	if diceRoll.Critical && diceRoll.Success {
		changes.XPGain *= 2
		// 可能获得道具或特质
	}

	return changes
}

// checkSceneEnd 检查场景是否结束
func (ss *StoryService) checkSceneEnd(_ *models.Scene, story *models.StoryState,
	charState *models.CharacterState, _ models.StateChanges) bool {

	// 角色死亡
	if charState.HP <= 0 {
		return true
	}

	// 理智归零
	if charState.SAN <= 0 {
		return true
	}

	// 100回合强制失败
	if story.Turn >= 100 {
		log.Println("⏰ [超时] 已达到100回合限制，场景强制结束")
		return true
	}

	// 评估剧情进度判断是否完成
	world, err := ss.storage.GetWorld(story.WorldID)
	if err == nil && len(world.PlotLines) > 0 {
		// 找到当前节点
		var currentNode *models.PlotNode
		var currentNodeIndex int
		for i, node := range world.PlotLines {
			if node.ID == story.CurrentPlotNodeID {
				currentNode = &world.PlotLines[i]
				currentNodeIndex = i
				break
			}
		}

		if currentNode != nil {
			// 检查是否在最后一个节点且进度达到100%
			if currentNodeIndex == len(world.PlotLines)-1 && story.PlotProgress >= 1.0 {
				log.Println("✅ [完成] 已到达最终剧情节点并完成所有进度")
				return true
			}

			// 每5回合检查一次进度
			if story.Turn > 0 && story.Turn%5 == 0 {
				// 如果进度太低（低于0.2），提醒玩家
				if story.PlotProgress < 0.2 {
					log.Printf("⚠️ [进度提醒] 当前回合: %d, 进度: %.1f%%，请尽快推进剧情\n",
						story.Turn, story.PlotProgress*100)
				}
			}
		}
	}

	return false
}

// getDefaultOptions 获取默认选项
func (ss *StoryService) getDefaultOptions() []models.Option {
	return []models.Option{
		{
			ID:          "opt_1",
			Label:       "观察四周",
			Description: "仔细观察周围的环境",
			ActionType:  "investigate",
			Difficulty:  10,
			Risk:        "low",
		},
		{
			ID:          "opt_2",
			Label:       "向前移动",
			Description: "小心地向前探索",
			ActionType:  "move",
			Difficulty:  12,
			Risk:        "medium",
		},
		{
			ID:          "opt_3",
			Label:       "等待观望",
			Description: "保持警惕，等待时机",
			ActionType:  "custom",
			Difficulty:  8,
			Risk:        "low",
		},
	}
}

// GetStory 获取故事状态
func (ss *StoryService) GetStory(storyID string) (*models.StoryState, error) {
	return ss.storage.GetStoryState(storyID)
}

// UndoTurn 回退到上一个回合
func (ss *StoryService) UndoTurn(storyID string) (*models.StoryState, error) {
	story, err := ss.storage.GetStoryState(storyID)
	if err != nil {
		return nil, fmt.Errorf("获取故事状态失败: %w", err)
	}

	if len(story.Snapshots) == 0 {
		return nil, fmt.Errorf("无法回退：没有历史记录")
	}

	// 获取最后一个快照
	snapshot := story.Snapshots[len(story.Snapshots)-1]

	// 恢复状态
	story.Turn = snapshot.Turn
	story.Narrative = snapshot.Narrative
	story.Snapshots = story.Snapshots[:len(story.Snapshots)-1]
	story.UpdatedAt = time.Now()

	// 恢复角色状态
	if err := ss.meta.RestoreCharacterState(story.CharacterID, story.WorldID, &snapshot.CharState); err != nil {
		return nil, fmt.Errorf("恢复角色状态失败: %w", err)
	}

	if err := ss.storage.UpdateStoryState(story); err != nil {
		return nil, fmt.Errorf("更新故事状态失败: %w", err)
	}

	log.Println("⏪ [回退] 已回退到回合", story.Turn)

	return story, nil
}

// CreateSaveGame 创建存档
func (ss *StoryService) CreateSaveGame(storyID, name, description string) (*models.SaveGame, error) {
	story, err := ss.storage.GetStoryState(storyID)
	if err != nil {
		return nil, fmt.Errorf("获取故事状态失败: %w", err)
	}

	// 获取场景信息作为描述
	scene, _ := ss.storage.GetScene(story.SceneID)
	if description == "" && scene != nil {
		description = fmt.Sprintf("第%d回合 - %s", story.Turn, scene.Name)
	}

	save := &models.SaveGame{
		ID:          uuid.New().String(),
		Name:        name,
		StoryID:     storyID,
		CharacterID: story.CharacterID,
		WorldID:     story.WorldID,
		Turn:        story.Turn,
		Description: description,
		CreatedAt:   time.Now(),
	}

	if err := ss.storage.CreateSaveGame(save); err != nil {
		return nil, fmt.Errorf("创建存档失败: %w", err)
	}

	log.Printf("💾 [存档] 已创建存档: %s (回合 %d)\n", name, story.Turn)

	return save, nil
}

// ListSaveGames 列出角色的所有存档
func (ss *StoryService) ListSaveGames(characterID string) ([]models.SaveGame, error) {
	return ss.storage.GetSaveGamesByCharacter(characterID)
}

// LoadStory 读取故事
func (ss *StoryService) LoadStory(ctx context.Context, storyID string) (*models.StoryState, *models.Scene, *models.CharacterState, error) {
	story, err := ss.storage.GetStoryState(storyID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("获取故事状态失败: %w", err)
	}

	scene, err := ss.storage.GetScene(story.SceneID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("获取场景失败: %w", err)
	}

	charState, err := ss.meta.GetCharacterState(story.CharacterID, story.WorldID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("获取角色状态失败: %w", err)
	}

	log.Printf("📂 [读档] 已加载故事: %s (回合 %d)\n", story.ID, story.Turn)

	return story, scene, charState, nil
}

// evaluatePlotProgress 评估并更新剧情推进
func (ss *StoryService) evaluatePlotProgress(ctx context.Context, story *models.StoryState, action models.Action, narrative string) error {
	// 获取世界信息
	world, err := ss.storage.GetWorld(story.WorldID)
	if err != nil {
		return fmt.Errorf("获取世界失败: %w", err)
	}

	if len(world.PlotLines) == 0 {
		return nil // 没有剧情节点，不需要评估
	}

	// 找到当前节点
	var currentNode *models.PlotNode
	var currentNodeIndex int
	for i, node := range world.PlotLines {
		if node.ID == story.CurrentPlotNodeID {
			currentNode = &world.PlotLines[i]
			currentNodeIndex = i
			break
		}
	}

	if currentNode == nil {
		return fmt.Errorf("当前剧情节点不存在")
	}

	// 找到下一个节点
	var nextNode *models.PlotNode
	isLastNode := false
	if currentNodeIndex < len(world.PlotLines)-1 {
		nextNode = &world.PlotLines[currentNodeIndex+1]
	} else {
		// 已经是最后一个节点，创建一个虚拟的"完成"节点用于评估
		nextNode = &models.PlotNode{
			ID:          "completion",
			Name:        "场景完成",
			Description: "当前场景的所有主要剧情已经完成，场景可以结束了。",
			Location:    currentNode.Location,
			IsPlayable:  true,
		}
		isLastNode = true
	}

	// 调用LLM评估剧情推进
	newProgress, reached, err := ss.llm.EvaluatePlotProgress(ctx, currentNode, nextNode, action, narrative, story.PlotProgress)
	if err != nil {
		return err
	}

	story.PlotProgress = newProgress

	// 追加一条系统消息显示当前进度与目标
	progressMsg := fmt.Sprintf("剧情进度：%.0f%% / 100%%（当前：%s → 目标：%s）", story.PlotProgress*100, currentNode.Name, nextNode.Name)
	story.Narrative = append(story.Narrative, models.NarrativeLog{
		Turn:      story.Turn,
		Type:      "system",
		Content:   progressMsg,
		Timestamp: time.Now(),
	})

	// 如果到达下一个节点
	if reached {
		log.Printf("🎯 [剧情推进] 玩家从「%s」推进到「%s」\n", currentNode.Name, nextNode.Name)

		// 如果是最后一个节点，不切换节点ID，保持当前节点并标记完成
		if isLastNode {
			log.Println("🎯 [完成] 已到达最终节点并完成所有进度，场景准备结束")
			// 将进度设为1.0以确保场景结束
			story.PlotProgress = 1.0
		} else {
			// 更新当前节点
			story.CurrentPlotNodeID = nextNode.ID
			story.PlotProgress = 0.0 // 重置推进度

			// 添加剧情节点到达的系统消息
			story.Narrative = append(story.Narrative, models.NarrativeLog{
				Turn:      story.Turn,
				Type:      "system",
				Content:   fmt.Sprintf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━\n🎯 【剧情推进】%s\n━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n%s", nextNode.Name, nextNode.Description),
				Timestamp: time.Now(),
			})

			// 如果到达了新的最后一个节点，标记故事即将结束
			if currentNodeIndex+1 >= len(world.PlotLines)-1 {
				log.Println("📖 [剧情] 已到达最终剧情节点")
			}
		}
	}

	return nil
}
