package models

import "time"

// Character 角色元信息（跨世界继承）
type Character struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Gender         string         `json:"gender"`          // 性别：male/female
	Age            int            `json:"age"`             // 年龄
	Appearance     string         `json:"appearance"`      // 外貌描述
	Personality    string         `json:"personality"`     // 性格特点
	Background     string         `json:"background"`      // 背景故事
	BaseAttributes map[string]int `json:"base_attributes"` // 基础属性（不随世界改变）
	Level          int            `json:"level"`
	XP             int            `json:"xp"`
	Traits         []string       `json:"traits"`    // 特质列表
	Inventory      []Item         `json:"inventory"` // 道具列表
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// CharacterState 角色在特定世界中的状态
type CharacterState struct {
	CharacterID string         `json:"character_id"`
	WorldID     string         `json:"world_id"`
	HP          int            `json:"hp"`
	MaxHP       int            `json:"max_hp"`
	SAN         int            `json:"san"`        // 理智值
	MaxSAN      int            `json:"max_san"`    // 最大理智值
	Attributes  map[string]int `json:"attributes"` // 力量、敏捷、智力等
	Status      []string       `json:"status"`     // 状态效果
	Relations   map[string]int `json:"relations"`  // 与NPC的关系好感度
}

// Item 道具
type Item struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Type        string            `json:"type"` // weapon, consumable, key_item, etc.
	Properties  map[string]string `json:"properties"`
}

// World 世界概要
type World struct {
	ID              string     `json:"id"`
	SegmentText     string     `json:"segment_text"`     // 原始输入文本
	OriginalSummary string     `json:"original_summary"` // 原小说摘要（1000字内）
	Name            string     `json:"name"`
	Description     string     `json:"description"`
	Genre           string     `json:"genre"`      // 类型：horror, fantasy, urban, etc.
	Difficulty      int        `json:"difficulty"` // 1-10
	Goals           []string   `json:"goals"`      // 本世界的通关目标
	NPCs            []NPC      `json:"npcs"`       // 关键NPC
	PlotLines       []PlotNode `json:"plot_lines"` // 剧情时间线
	CreatedAt       time.Time  `json:"created_at"`
}

// PlotNode 剧情节点
type PlotNode struct {
	ID          string   `json:"id"`
	Order       int      `json:"order"`       // 顺序（1开始）
	Name        string   `json:"name"`        // 节点名称
	Description string   `json:"description"` // 节点描述
	Location    string   `json:"location"`    // 发生地点
	KeyNPCs     []string `json:"key_npcs"`    // 关键NPC名字
	Difficulty  int      `json:"difficulty"`  // 该节点难度1-10
	IsPlayable  bool     `json:"is_playable"` // 是否可作为起始点
}

// NPC 非玩家角色
type NPC struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Role         string   `json:"role"` // 角色定位：ally, enemy, neutral, boss
	Traits       []string `json:"traits"`
	Relationship int      `json:"relationship"` // 初始好感度
}

// Scene 场景/关卡
type Scene struct {
	ID          string   `json:"id"`
	WorldID     string   `json:"world_id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Type        string   `json:"type"`       // exploration, combat, social, puzzle
	Threats     []string `json:"threats"`    // 威胁/挑战
	Objectives  []string `json:"objectives"` // 场景目标
}

// StoryState 故事状态（一次游戏进程）
type StoryState struct {
	ID                string          `json:"id"`
	CharacterID       string          `json:"character_id"`
	WorldID           string          `json:"world_id"`
	SceneID           string          `json:"scene_id"`
	CurrentPlotNodeID string          `json:"current_plot_node_id"` // 当前所在剧情节点ID
	Turn              int             `json:"turn"`
	Narrative         []NarrativeLog  `json:"narrative"`     // 叙事日志
	Snapshots         []StateSnapshot `json:"snapshots"`     // 历史快照（用于回退）
	PlotProgress      float64         `json:"plot_progress"` // 向下一节点的推进度（0-1）
	Status            string          `json:"status"`        // active, completed, failed
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// StateSnapshot 状态快照（用于回退）
type StateSnapshot struct {
	Turn      int            `json:"turn"`
	Narrative []NarrativeLog `json:"narrative"`
	CharState CharacterState `json:"char_state"`
	Timestamp time.Time      `json:"timestamp"`
}

// NarrativeLog 叙事日志条目
type NarrativeLog struct {
	Turn      int       `json:"turn"`
	Type      string    `json:"type"` // action, result, dialogue, system
	Content   string    `json:"content"`
	DiceRoll  *DiceRoll `json:"dice_roll,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// DiceRoll 骰子检定结果
type DiceRoll struct {
	Type     string `json:"type"`   // D20, D6, etc.
	Result   int    `json:"result"` // 投掷结果
	Modifier int    `json:"modifier"`
	Target   int    `json:"target"` // 目标难度
	Success  bool   `json:"success"`
	Critical bool   `json:"critical"` // 大成功/大失败
}

// Action 玩家行动
type Action struct {
	Type       string            `json:"type"` // move, attack, talk, use_item, custom
	Content    string            `json:"content"`
	Target     string            `json:"target,omitempty"`
	Parameters map[string]string `json:"parameters,omitempty"`
}

// ActionResult 行动结果
type ActionResult struct {
	Success     bool         `json:"success"`
	Narrative   string       `json:"narrative"` // 结果描述
	DiceRoll    *DiceRoll    `json:"dice_roll,omitempty"`
	Changes     StateChanges `json:"changes"`      // 状态变化
	NextOptions []Option     `json:"next_options"` // 下一步可选行动
	SceneEnd    bool         `json:"scene_end"`    // 场景是否结束
}

// StateChanges 状态变化
type StateChanges struct {
	HPChange       int            `json:"hp_change,omitempty"`
	SANChange      int            `json:"san_change,omitempty"`
	XPGain         int            `json:"xp_gain,omitempty"`
	ItemsGained    []Item         `json:"items_gained,omitempty"`
	ItemsLost      []string       `json:"items_lost,omitempty"` // item IDs
	TraitsGained   []string       `json:"traits_gained,omitempty"`
	StatusAdded    []string       `json:"status_added,omitempty"`
	StatusRemoved  []string       `json:"status_removed,omitempty"`
	RelationChange map[string]int `json:"relation_change,omitempty"` // NPC_ID -> change
}

// Option 可选行动
type Option struct {
	ID          string `json:"id"`
	Label       string `json:"label"`       // 显示文本
	Description string `json:"description"` // 详细说明
	ActionType  string `json:"action_type"`
	Difficulty  int    `json:"difficulty,omitempty"` // 如需检定
	Risk        string `json:"risk,omitempty"`       // low, medium, high
}

// Config 配置
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	LLM      LLMConfig      `yaml:"llm"`
	Game     GameConfig     `yaml:"game"`
}

type ServerConfig struct {
	Port string `yaml:"port"`
	Host string `yaml:"host"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type LLMConfig struct {
	Provider    string  `yaml:"provider"`
	APIKey      string  `yaml:"api_key"`
	APIBase     string  `yaml:"api_base"`
	Model       string  `yaml:"model"`
	Temperature float32 `yaml:"temperature"`
	MaxTokens   int     `yaml:"max_tokens"`
}

type GameConfig struct {
	DefaultHP       int  `yaml:"default_hp"`
	DefaultSAN      int  `yaml:"default_san"`
	MaxTurnPerScene int  `yaml:"max_turn_per_scene"`
	EnableAdultMode bool `yaml:"enable_adult_mode"`
}

// SaveGame 存档
type SaveGame struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"` // 存档名称
	StoryID     string    `json:"story_id"`
	CharacterID string    `json:"character_id"`
	WorldID     string    `json:"world_id"`
	Turn        int       `json:"turn"`
	Description string    `json:"description"` // 存档描述（当前位置等）
	CreatedAt   time.Time `json:"created_at"`
}
