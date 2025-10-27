package services

import (
	"math/rand"
	"time"

	"github.com/aiwuxian/project-abyss/internal/models"
)

type RuleEngine struct {
	rng *rand.Rand
}

func NewRuleEngine() *RuleEngine {
	return &RuleEngine{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// RollD20 投D20骰子
func (re *RuleEngine) RollD20() int {
	return re.rng.Intn(20) + 1
}

// RollDice 投任意骰子
func (re *RuleEngine) RollDice(sides int) int {
	return re.rng.Intn(sides) + 1
}

// Check 执行检定
func (re *RuleEngine) Check(attribute int, difficulty int) *models.DiceRoll {
	roll := re.RollD20()
	total := roll + attribute

	result := &models.DiceRoll{
		Type:     "D20",
		Result:   roll,
		Modifier: attribute,
		Target:   difficulty,
		Success:  total >= difficulty,
		Critical: roll == 20 || roll == 1,
	}

	// 大成功
	if roll == 20 {
		result.Success = true
	}
	// 大失败
	if roll == 1 {
		result.Success = false
	}

	return result
}

// CalculateDifficulty 根据场景和行动计算难度
func (re *RuleEngine) CalculateDifficulty(sceneType string, actionType string) int {
	baseDifficulty := 10

	// 根据场景类型调整
	switch sceneType {
	case "combat":
		baseDifficulty = 15
	case "social":
		baseDifficulty = 12
	case "exploration":
		baseDifficulty = 10
	case "puzzle":
		baseDifficulty = 14
	}

	// 根据行动类型微调
	switch actionType {
	case "attack":
		baseDifficulty += 2
	case "sneak":
		baseDifficulty += 3
	case "persuade":
		baseDifficulty += 1
	}

	return baseDifficulty
}

// CalculateXPGain 计算经验值获得
func (re *RuleEngine) CalculateXPGain(difficulty int, success bool) int {
	baseXP := difficulty * 10
	if success {
		return baseXP
	}
	return baseXP / 2 // 失败也有一半经验
}

// CheckLevelUp 检查是否升级
func (re *RuleEngine) CheckLevelUp(currentXP int, currentLevel int) bool {
	requiredXP := currentLevel * 100
	return currentXP >= requiredXP
}

// CalculateDamage 计算伤害
func (re *RuleEngine) CalculateDamage(attackPower int, critical bool) int {
	damage := re.RollDice(6) + attackPower
	if critical {
		damage *= 2
	}
	return damage
}
