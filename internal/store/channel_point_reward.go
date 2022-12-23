package store

import (
	"errors"
	"time"

	"github.com/gempir/gempbot/internal/dto"
	"gorm.io/gorm/clause"
)

type ChannelPointReward struct {
	OwnerTwitchID                     string         `gorm:"primaryKey"`
	Type                              dto.RewardType `gorm:"primaryKey"`
	RewardID                          string         `gorm:"index"`
	ApproveOnly                       bool           `gorm:"default:false"`
	CreatedAt                         time.Time
	UpdatedAt                         time.Time
	Title                             string
	Prompt                            string
	Cost                              int
	BackgroundColor                   string
	IsMaxPerStreamEnabled             bool
	MaxPerStream                      int
	IsUserInputRequired               bool
	IsMaxPerUserPerStreamEnabled      bool
	MaxPerUserPerStream               int
	IsGlobalCooldownEnabled           bool
	GlobalCooldownSeconds             int
	ShouldRedemptionsSkipRequestQueue bool
	Enabled                           bool
	AdditionalOptions                 string
}

func (db *Database) GetChannelPointRewards(userID string) []ChannelPointReward {
	var rewards []ChannelPointReward

	db.Client.Where("owner_twitch_id = ?", userID).Find(&rewards)

	return rewards
}

func (db *Database) GetEnabledChannelPointRewardByID(rewardID string) (ChannelPointReward, error) {
	var reward ChannelPointReward
	result := db.Client.Where("reward_id = ? AND enabled = ?", rewardID, true).First(&reward)
	if result.RowsAffected == 0 {
		return reward, errors.New("not found")
	}

	return reward, nil
}

func (db *Database) GetChannelPointReward(userID string, rewardType dto.RewardType) (ChannelPointReward, error) {
	var reward ChannelPointReward
	result := db.Client.Where("owner_twitch_id = ? AND type = ?", userID, rewardType).First(&reward)
	if result.RowsAffected == 0 {
		return reward, errors.New("not found")
	}

	return reward, nil
}

func (db *Database) DeleteChannelPointReward(userID string, rewardType dto.RewardType) {
	db.Client.Where("owner_twitch_id = ? AND type = ?", userID, rewardType).Delete(&ChannelPointReward{})
}

func (db *Database) DeleteChannelPointRewardById(userID string, rewardID string) {
	db.Client.Where("owner_twitch_id = ? AND reward_id = ?", userID, rewardID).Delete(&ChannelPointReward{})
}

func (db *Database) GetDistinctRewardsPerUser() []ChannelPointReward {
	var rewards []ChannelPointReward
	db.Client.Distinct("owner_twitch_id").Find(&rewards)

	return rewards
}

func (db *Database) SaveReward(reward ChannelPointReward) error {
	update := db.Client.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&reward)

	return update.Error
}
