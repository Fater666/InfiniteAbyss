package services

import (
	"database/sql"
	"time"

	"github.com/aiwuxian/project-abyss/internal/models"
	"github.com/aiwuxian/project-abyss/internal/storage"
	"github.com/google/uuid"
)

type MetaService struct {
	storage *storage.Storage
	config  models.GameConfig
}

func NewMetaService(storage *storage.Storage, config models.GameConfig) *MetaService {
	return &MetaService{
		storage: storage,
		config:  config,
	}
}

// CreateCharacter 创建新角色（手动创建）
func (ms *MetaService) CreateCharacter(char *models.Character) (*models.Character, error) {
	// 如果没有基础属性，使用默认值
	if char.BaseAttributes == nil || len(char.BaseAttributes) == 0 {
		char.BaseAttributes = map[string]int{
			"strength":     10,
			"dexterity":    10,
			"intelligence": 10,
			"charisma":     10,
			"perception":   10,
		}
	}

	char.ID = uuid.New().String()
	char.Level = 1
	char.XP = 0
	char.Traits = []string{}
	char.Inventory = []models.Item{}
	char.CreatedAt = time.Now()
	char.UpdatedAt = time.Now()

	if err := ms.storage.CreateCharacter(char); err != nil {
		return nil, err
	}

	return char, nil
}

// GetCharacter 获取角色
func (ms *MetaService) GetCharacter(id string) (*models.Character, error) {
	return ms.storage.GetCharacter(id)
}

// GetAllCharacters 获取所有角色
func (ms *MetaService) GetAllCharacters() ([]models.Character, error) {
	return ms.storage.GetAllCharacters()
}

// InitCharacterInWorld 初始化角色在新世界的状态
func (ms *MetaService) InitCharacterInWorld(characterID, worldID string, world *models.World) (*models.CharacterState, error) {
	// 尝试获取已有状态
	state, err := ms.storage.GetCharacterState(characterID, worldID)
	if err == nil {
		return state, nil // 已存在
	}

	if err != sql.ErrNoRows {
		return nil, err
	}

	// 获取角色信息
	char, err := ms.storage.GetCharacter(characterID)
	if err != nil {
		return nil, err
	}

	// 创建新状态
	state = &models.CharacterState{
		CharacterID: characterID,
		WorldID:     worldID,
		HP:          ms.config.DefaultHP,
		MaxHP:       ms.config.DefaultHP,
		SAN:         ms.config.DefaultSAN,
		MaxSAN:      ms.config.DefaultSAN,
		Attributes:  ms.calculateAttributes(char, world),
		Status:      []string{},
		Relations:   ms.initRelations(world),
	}

	if err := ms.storage.SaveCharacterState(state); err != nil {
		return nil, err
	}

	return state, nil
}

// calculateAttributes 根据角色基础属性、等级和世界类型计算属性
func (ms *MetaService) calculateAttributes(char *models.Character, world *models.World) map[string]int {
	// 从基础属性开始
	attrs := make(map[string]int)
	if char.BaseAttributes != nil && len(char.BaseAttributes) > 0 {
		for k, v := range char.BaseAttributes {
			attrs[k] = v
		}
	} else {
		// 如果没有基础属性，使用默认值
		attrs = map[string]int{
			"strength":     10,
			"dexterity":    10,
			"intelligence": 10,
			"charisma":     10,
			"perception":   10,
		}
	}

	// 根据等级加成
	levelBonus := char.Level - 1
	for k := range attrs {
		attrs[k] += levelBonus
	}

	// 根据世界类型调整
	switch world.Genre {
	case "horror":
		attrs["perception"] += 2
		attrs["strength"] -= 1
	case "fantasy":
		attrs["strength"] += 1
		attrs["charisma"] += 1
	case "urban":
		attrs["dexterity"] += 1
		attrs["intelligence"] += 1
	case "scifi":
		attrs["intelligence"] += 2
	case "romance", "school", "workplace":
		attrs["charisma"] += 2
	}

	return attrs
}

// initRelations 初始化与NPC的关系
func (ms *MetaService) initRelations(world *models.World) map[string]int {
	relations := make(map[string]int)
	for _, npc := range world.NPCs {
		relations[npc.ID] = npc.Relationship
	}
	return relations
}

// ApplyChanges 应用状态变化
func (ms *MetaService) ApplyChanges(characterID, worldID string, changes models.StateChanges) error {
	// 更新角色元信息
	char, err := ms.storage.GetCharacter(characterID)
	if err != nil {
		return err
	}

	char.XP += changes.XPGain

	// 处理道具
	for _, item := range changes.ItemsGained {
		char.Inventory = append(char.Inventory, item)
	}

	// 移除道具
	for _, itemID := range changes.ItemsLost {
		for i, item := range char.Inventory {
			if item.ID == itemID {
				char.Inventory = append(char.Inventory[:i], char.Inventory[i+1:]...)
				break
			}
		}
	}

	// 添加特质
	char.Traits = append(char.Traits, changes.TraitsGained...)

	char.UpdatedAt = time.Now()

	if err := ms.storage.UpdateCharacter(char); err != nil {
		return err
	}

	// 更新世界状态
	state, err := ms.storage.GetCharacterState(characterID, worldID)
	if err != nil {
		return err
	}

	state.HP += changes.HPChange
	if state.HP > state.MaxHP {
		state.HP = state.MaxHP
	}
	if state.HP < 0 {
		state.HP = 0
	}

	state.SAN += changes.SANChange
	if state.SAN > state.MaxSAN {
		state.SAN = state.MaxSAN
	}
	if state.SAN < 0 {
		state.SAN = 0
	}

	// 添加状态效果
	state.Status = append(state.Status, changes.StatusAdded...)

	// 移除状态效果
	for _, status := range changes.StatusRemoved {
		for i, s := range state.Status {
			if s == status {
				state.Status = append(state.Status[:i], state.Status[i+1:]...)
				break
			}
		}
	}

	// 更新关系
	for npcID, change := range changes.RelationChange {
		state.Relations[npcID] += change
	}

	return ms.storage.SaveCharacterState(state)
}

// GetCharacterState 获取角色在世界中的状态
func (ms *MetaService) GetCharacterState(characterID, worldID string) (*models.CharacterState, error) {
	return ms.storage.GetCharacterState(characterID, worldID)
}

// RestoreCharacterState 恢复角色状态（用于回退）
func (ms *MetaService) RestoreCharacterState(characterID, worldID string, snapshot *models.CharacterState) error {
	return ms.storage.SaveCharacterState(snapshot)
}
