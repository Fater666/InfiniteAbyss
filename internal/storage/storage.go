package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aiwuxian/project-abyss/internal/models"
	_ "modernc.org/sqlite"
)

type Storage struct {
	db *sql.DB
}

func New(dbPath string) (*Storage, error) {
	// 确保目录存在
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	s := &Storage{db: db}
	if err := s.initSchema(); err != nil {
		return nil, fmt.Errorf("初始化数据库结构失败: %w", err)
	}

	return s, nil
}

func (s *Storage) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS characters (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		gender TEXT DEFAULT 'male',
		age INTEGER DEFAULT 20,
		appearance TEXT,
		personality TEXT,
		background TEXT,
		base_attributes TEXT, -- JSON object
		level INTEGER DEFAULT 1,
		xp INTEGER DEFAULT 0,
		traits TEXT, -- JSON array
		inventory TEXT, -- JSON array
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS worlds (
		id TEXT PRIMARY KEY,
		segment_text TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		genre TEXT,
		difficulty INTEGER DEFAULT 5,
		goals TEXT, -- JSON array
		npcs TEXT, -- JSON array
		plot_lines TEXT, -- JSON array
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS character_states (
		character_id TEXT,
		world_id TEXT,
		hp INTEGER,
		max_hp INTEGER,
		san INTEGER,
		max_san INTEGER,
		attributes TEXT, -- JSON object
		status TEXT, -- JSON array
		relations TEXT, -- JSON object
		PRIMARY KEY (character_id, world_id),
		FOREIGN KEY (character_id) REFERENCES characters(id),
		FOREIGN KEY (world_id) REFERENCES worlds(id)
	);

	CREATE TABLE IF NOT EXISTS scenes (
		id TEXT PRIMARY KEY,
		world_id TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		type TEXT,
		threats TEXT, -- JSON array
		objectives TEXT, -- JSON array
		FOREIGN KEY (world_id) REFERENCES worlds(id)
	);

	CREATE TABLE IF NOT EXISTS story_states (
		id TEXT PRIMARY KEY,
		character_id TEXT NOT NULL,
		world_id TEXT NOT NULL,
		scene_id TEXT,
		turn INTEGER DEFAULT 0,
		narrative TEXT, -- JSON array
		snapshots TEXT, -- JSON array
		status TEXT DEFAULT 'active',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (character_id) REFERENCES characters(id),
		FOREIGN KEY (world_id) REFERENCES worlds(id),
		FOREIGN KEY (scene_id) REFERENCES scenes(id)
	);

	CREATE TABLE IF NOT EXISTS save_games (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		story_id TEXT NOT NULL,
		character_id TEXT NOT NULL,
		world_id TEXT NOT NULL,
		turn INTEGER,
		description TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (story_id) REFERENCES story_states(id),
		FOREIGN KEY (character_id) REFERENCES characters(id),
		FOREIGN KEY (world_id) REFERENCES worlds(id)
	);

	CREATE INDEX IF NOT EXISTS idx_story_character ON story_states(character_id);
	CREATE INDEX IF NOT EXISTS idx_story_world ON story_states(world_id);
	CREATE INDEX IF NOT EXISTS idx_story_status ON story_states(status);
	`

	_, err := s.db.Exec(schema)
	return err
}

func (s *Storage) Close() error {
	return s.db.Close()
}

// Character operations
func (s *Storage) CreateCharacter(char *models.Character) error {
	traitsJSON, _ := json.Marshal(char.Traits)
	inventoryJSON, _ := json.Marshal(char.Inventory)
	baseAttrsJSON, _ := json.Marshal(char.BaseAttributes)

	_, err := s.db.Exec(`
		INSERT INTO characters (id, name, gender, age, appearance, personality, background, base_attributes, level, xp, traits, inventory, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, char.ID, char.Name, char.Gender, char.Age, char.Appearance, char.Personality, char.Background, baseAttrsJSON,
		char.Level, char.XP, traitsJSON, inventoryJSON, char.CreatedAt, char.UpdatedAt)

	return err
}

func (s *Storage) GetCharacter(id string) (*models.Character, error) {
	var char models.Character
	var traitsJSON, inventoryJSON, baseAttrsJSON string

	err := s.db.QueryRow(`
		SELECT id, name, gender, age, appearance, personality, background, base_attributes, level, xp, traits, inventory, created_at, updated_at
		FROM characters WHERE id = ?
	`, id).Scan(&char.ID, &char.Name, &char.Gender, &char.Age, &char.Appearance, &char.Personality, &char.Background, &baseAttrsJSON,
		&char.Level, &char.XP, &traitsJSON, &inventoryJSON, &char.CreatedAt, &char.UpdatedAt)

	if err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(traitsJSON), &char.Traits)
	json.Unmarshal([]byte(inventoryJSON), &char.Inventory)
	json.Unmarshal([]byte(baseAttrsJSON), &char.BaseAttributes)

	return &char, nil
}

func (s *Storage) UpdateCharacter(char *models.Character) error {
	traitsJSON, _ := json.Marshal(char.Traits)
	inventoryJSON, _ := json.Marshal(char.Inventory)
	baseAttrsJSON, _ := json.Marshal(char.BaseAttributes)

	_, err := s.db.Exec(`
		UPDATE characters 
		SET name=?, gender=?, age=?, appearance=?, personality=?, background=?, base_attributes=?, level=?, xp=?, traits=?, inventory=?, updated_at=?
		WHERE id=?
	`, char.Name, char.Gender, char.Age, char.Appearance, char.Personality, char.Background, baseAttrsJSON,
		char.Level, char.XP, traitsJSON, inventoryJSON, time.Now(), char.ID)

	return err
}

// GetAllCharacters 获取所有角色列表
func (s *Storage) GetAllCharacters() ([]models.Character, error) {
	rows, err := s.db.Query(`
		SELECT id, name, gender, age, appearance, personality, background, base_attributes, level, xp, traits, inventory, created_at, updated_at
		FROM characters
		ORDER BY created_at DESC
	`)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var characters []models.Character
	for rows.Next() {
		var char models.Character
		var traitsJSON, inventoryJSON, baseAttrsJSON string

		err := rows.Scan(&char.ID, &char.Name, &char.Gender, &char.Age, &char.Appearance, &char.Personality, &char.Background, &baseAttrsJSON,
			&char.Level, &char.XP, &traitsJSON, &inventoryJSON, &char.CreatedAt, &char.UpdatedAt)

		if err != nil {
			continue
		}

		json.Unmarshal([]byte(traitsJSON), &char.Traits)
		json.Unmarshal([]byte(inventoryJSON), &char.Inventory)
		json.Unmarshal([]byte(baseAttrsJSON), &char.BaseAttributes)

		characters = append(characters, char)
	}

	return characters, nil
}

// World operations
func (s *Storage) CreateWorld(world *models.World) error {
	goalsJSON, _ := json.Marshal(world.Goals)
	npcsJSON, _ := json.Marshal(world.NPCs)
	plotLinesJSON, _ := json.Marshal(world.PlotLines)

	_, err := s.db.Exec(`
		INSERT INTO worlds (id, segment_text, name, description, genre, difficulty, goals, npcs, plot_lines, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, world.ID, world.SegmentText, world.Name, world.Description,
		world.Genre, world.Difficulty, goalsJSON, npcsJSON, plotLinesJSON, world.CreatedAt)

	return err
}

func (s *Storage) GetWorld(id string) (*models.World, error) {
	var world models.World
	var goalsJSON, npcsJSON, plotLinesJSON string

	err := s.db.QueryRow(`
		SELECT id, segment_text, name, description, genre, difficulty, goals, npcs, plot_lines, created_at
		FROM worlds WHERE id = ?
	`, id).Scan(&world.ID, &world.SegmentText, &world.Name, &world.Description,
		&world.Genre, &world.Difficulty, &goalsJSON, &npcsJSON, &plotLinesJSON, &world.CreatedAt)

	if err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(goalsJSON), &world.Goals)
	json.Unmarshal([]byte(npcsJSON), &world.NPCs)
	json.Unmarshal([]byte(plotLinesJSON), &world.PlotLines)

	return &world, nil
}

// CharacterState operations
func (s *Storage) SaveCharacterState(state *models.CharacterState) error {
	attributesJSON, _ := json.Marshal(state.Attributes)
	statusJSON, _ := json.Marshal(state.Status)
	relationsJSON, _ := json.Marshal(state.Relations)

	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO character_states 
		(character_id, world_id, hp, max_hp, san, max_san, attributes, status, relations)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, state.CharacterID, state.WorldID, state.HP, state.MaxHP,
		state.SAN, state.MaxSAN, attributesJSON, statusJSON, relationsJSON)

	return err
}

func (s *Storage) GetCharacterState(characterID, worldID string) (*models.CharacterState, error) {
	var state models.CharacterState
	var attributesJSON, statusJSON, relationsJSON string

	err := s.db.QueryRow(`
		SELECT character_id, world_id, hp, max_hp, san, max_san, attributes, status, relations
		FROM character_states WHERE character_id = ? AND world_id = ?
	`, characterID, worldID).Scan(&state.CharacterID, &state.WorldID,
		&state.HP, &state.MaxHP, &state.SAN, &state.MaxSAN,
		&attributesJSON, &statusJSON, &relationsJSON)

	if err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(attributesJSON), &state.Attributes)
	json.Unmarshal([]byte(statusJSON), &state.Status)
	json.Unmarshal([]byte(relationsJSON), &state.Relations)

	return &state, nil
}

// Scene operations
func (s *Storage) CreateScene(scene *models.Scene) error {
	threatsJSON, _ := json.Marshal(scene.Threats)
	objectivesJSON, _ := json.Marshal(scene.Objectives)

	_, err := s.db.Exec(`
		INSERT INTO scenes (id, world_id, name, description, type, threats, objectives)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, scene.ID, scene.WorldID, scene.Name, scene.Description,
		scene.Type, threatsJSON, objectivesJSON)

	return err
}

func (s *Storage) GetScene(id string) (*models.Scene, error) {
	var scene models.Scene
	var threatsJSON, objectivesJSON string

	err := s.db.QueryRow(`
		SELECT id, world_id, name, description, type, threats, objectives
		FROM scenes WHERE id = ?
	`, id).Scan(&scene.ID, &scene.WorldID, &scene.Name, &scene.Description,
		&scene.Type, &threatsJSON, &objectivesJSON)

	if err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(threatsJSON), &scene.Threats)
	json.Unmarshal([]byte(objectivesJSON), &scene.Objectives)

	return &scene, nil
}

// StoryState operations
func (s *Storage) CreateStoryState(story *models.StoryState) error {
	narrativeJSON, _ := json.Marshal(story.Narrative)
	snapshotsJSON, _ := json.Marshal(story.Snapshots)

	_, err := s.db.Exec(`
		INSERT INTO story_states (id, character_id, world_id, scene_id, turn, narrative, snapshots, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, story.ID, story.CharacterID, story.WorldID, story.SceneID,
		story.Turn, narrativeJSON, snapshotsJSON, story.Status, story.CreatedAt, story.UpdatedAt)

	return err
}

func (s *Storage) UpdateStoryState(story *models.StoryState) error {
	narrativeJSON, _ := json.Marshal(story.Narrative)
	snapshotsJSON, _ := json.Marshal(story.Snapshots)

	_, err := s.db.Exec(`
		UPDATE story_states 
		SET scene_id=?, turn=?, narrative=?, snapshots=?, status=?, updated_at=?
		WHERE id=?
	`, story.SceneID, story.Turn, narrativeJSON, snapshotsJSON, story.Status,
		time.Now(), story.ID)

	return err
}

func (s *Storage) GetStoryState(id string) (*models.StoryState, error) {
	var story models.StoryState
	var narrativeJSON, snapshotsJSON string

	err := s.db.QueryRow(`
		SELECT id, character_id, world_id, scene_id, turn, narrative, snapshots, status, created_at, updated_at
		FROM story_states WHERE id = ?
	`, id).Scan(&story.ID, &story.CharacterID, &story.WorldID, &story.SceneID,
		&story.Turn, &narrativeJSON, &snapshotsJSON, &story.Status, &story.CreatedAt, &story.UpdatedAt)

	if err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(narrativeJSON), &story.Narrative)
	json.Unmarshal([]byte(snapshotsJSON), &story.Snapshots)

	return &story, nil
}

func (s *Storage) GetActiveStoryByCharacter(characterID string) (*models.StoryState, error) {
	var story models.StoryState
	var narrativeJSON, snapshotsJSON string

	err := s.db.QueryRow(`
		SELECT id, character_id, world_id, scene_id, turn, narrative, snapshots, status, created_at, updated_at
		FROM story_states WHERE character_id = ? AND status = 'active'
		ORDER BY updated_at DESC LIMIT 1
	`, characterID).Scan(&story.ID, &story.CharacterID, &story.WorldID, &story.SceneID,
		&story.Turn, &narrativeJSON, &snapshotsJSON, &story.Status, &story.CreatedAt, &story.UpdatedAt)

	if err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(narrativeJSON), &story.Narrative)
	json.Unmarshal([]byte(snapshotsJSON), &story.Snapshots)

	return &story, nil
}

// SaveGame operations
func (s *Storage) CreateSaveGame(save *models.SaveGame) error {
	_, err := s.db.Exec(`
		INSERT INTO save_games (id, name, story_id, character_id, world_id, turn, description, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, save.ID, save.Name, save.StoryID, save.CharacterID, save.WorldID,
		save.Turn, save.Description, save.CreatedAt)

	return err
}

func (s *Storage) GetSaveGamesByCharacter(characterID string) ([]models.SaveGame, error) {
	rows, err := s.db.Query(`
		SELECT id, name, story_id, character_id, world_id, turn, description, created_at
		FROM save_games WHERE character_id = ?
		ORDER BY created_at DESC
	`, characterID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var saves []models.SaveGame
	for rows.Next() {
		var save models.SaveGame
		err := rows.Scan(&save.ID, &save.Name, &save.StoryID, &save.CharacterID,
			&save.WorldID, &save.Turn, &save.Description, &save.CreatedAt)
		if err != nil {
			continue
		}
		saves = append(saves, save)
	}

	return saves, nil
}

func (s *Storage) DeleteSaveGame(id string) error {
	_, err := s.db.Exec(`DELETE FROM save_games WHERE id = ?`, id)
	return err
}
