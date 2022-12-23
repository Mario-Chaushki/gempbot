package helixclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gempir/gempbot/internal/log"
	"github.com/nicklaw5/helix/v2"
)

type CreateCustomRewardRequest struct {
	Title                             string `json:"title"`
	Prompt                            string `json:"prompt"`
	Cost                              int    `json:"cost"`
	IsEnabled                         bool   `json:"is_enabled"`
	BackgroundColor                   string `json:"background_color,omitempty"`
	IsUserInputRequired               bool   `json:"is_user_input_required"`
	IsMaxPerStreamEnabled             bool   `json:"is_max_per_stream_enabled"`
	MaxPerStream                      int    `json:"max_per_stream"`
	IsMaxPerUserPerStreamEnabled      bool   `json:"is_max_per_user_per_stream_enabled"`
	MaxPerUserPerStream               int    `json:"max_per_user_per_stream"`
	IsGlobalCooldownEnabled           bool   `json:"is_global_cooldown_enabled"`
	GlobalCoolDownSeconds             int    `json:"global_cooldown_seconds"`
	ShouldRedemptionsSkipRequestQueue bool   `json:"should_redemptions_skip_request_queue"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Status  int    `json:"status"`
	Message string `json:"message"`
}

type CreateCustomRewardResponse struct {
	Data []CreateCustomRewardResponseDataItem `json:"data"`
}

type CreateCustomRewardResponseDataItem struct {
	BroadcasterName     string      `json:"broadcaster_name"`
	BroadcasterLogin    string      `json:"broadcaster_login"`
	BroadcasterID       string      `json:"broadcaster_id"`
	ID                  string      `json:"id"`
	Image               interface{} `json:"image"`
	BackgroundColor     string      `json:"background_color"`
	IsEnabled           bool        `json:"is_enabled"`
	Cost                int         `json:"cost"`
	Title               string      `json:"title"`
	Prompt              string      `json:"prompt"`
	IsUserInputRequired bool        `json:"is_user_input_required"`
	MaxPerStreamSetting struct {
		IsEnabled    bool `json:"is_enabled"`
		MaxPerStream int  `json:"max_per_stream"`
	} `json:"max_per_stream_setting"`
	MaxPerUserPerStreamSetting struct {
		IsEnabled           bool `json:"is_enabled"`
		MaxPerUserPerStream int  `json:"max_per_user_per_stream"`
	} `json:"max_per_user_per_stream_setting"`
	GlobalCooldownSetting struct {
		IsEnabled             bool `json:"is_enabled"`
		GlobalCooldownSeconds int  `json:"global_cooldown_seconds"`
	} `json:"global_cooldown_setting"`
	IsPaused     bool `json:"is_paused"`
	IsInStock    bool `json:"is_in_stock"`
	DefaultImage struct {
		URL1X string `json:"url_1x"`
		URL2X string `json:"url_2x"`
		URL4X string `json:"url_4x"`
	} `json:"default_image"`
	ShouldRedemptionsSkipRequestQueue bool        `json:"should_redemptions_skip_request_queue"`
	RedemptionsRedeemedCurrentStream  interface{} `json:"redemptions_redeemed_current_stream"`
	CooldownExpiresAt                 interface{} `json:"cooldown_expires_at"`
}

func (c *HelixClient) CreateOrUpdateReward(userID, userAccessToken string, reward CreateCustomRewardRequest, rewardID string) (*helix.ChannelCustomReward, error) {
	log.Infof("Creating/Updating Reward for user %s reward title: %s", userID, reward.Title)

	var resp *helix.ChannelCustomRewardResponse
	var err error

	c.Client.SetUserAccessToken(userAccessToken)
	if rewardID == "" {
		resp, err = c.Client.CreateCustomReward(&helix.ChannelCustomRewardsParams{
			BroadcasterID:                userID,
			Title:                        reward.Title,
			Prompt:                       reward.Prompt,
			Cost:                         reward.Cost,
			IsEnabled:                    reward.IsEnabled,
			BackgroundColor:              reward.BackgroundColor,
			IsUserInputRequired:          reward.IsUserInputRequired,
			IsMaxPerStreamEnabled:        reward.IsMaxPerStreamEnabled,
			MaxPerStream:                 reward.MaxPerStream,
			IsMaxPerUserPerStreamEnabled: reward.IsMaxPerUserPerStreamEnabled,
			MaxPerUserPerStream:          reward.MaxPerUserPerStream,
		})
	} else {
		resp, err = c.Client.UpdateCustomReward(&helix.UpdateChannelCustomRewardsParams{
			BroadcasterID:                userID,
			ID:                           rewardID,
			Title:                        reward.Title,
			Prompt:                       reward.Prompt,
			Cost:                         reward.Cost,
			BackgroundColor:              reward.BackgroundColor,
			IsEnabled:                    reward.IsEnabled,
			IsUserInputRequired:          reward.IsUserInputRequired,
			IsMaxPerStreamEnabled:        reward.IsMaxPerStreamEnabled,
			MaxPerStream:                 reward.MaxPerStream,
			IsMaxPerUserPerStreamEnabled: reward.IsMaxPerUserPerStreamEnabled,
			MaxPerUserPerStream:          reward.MaxPerUserPerStream,
		})
	}
	c.Client.SetUserAccessToken("")
	if err != nil {
		return &helix.ChannelCustomReward{}, err
	}
	if resp.ResponseCommon.Error != "" {
		return &helix.ChannelCustomReward{}, fmt.Errorf("%s", resp.ResponseCommon.ErrorMessage)
	}

	if len(resp.Data.ChannelCustomRewards) != 1 {
		return &helix.ChannelCustomReward{}, fmt.Errorf("none or multiple rewards returned")
	}

	return &resp.Data.ChannelCustomRewards[0], err
}

type UpdateRedemptionStatusRequest struct {
	Status string `json:"status"`
}

type UpdateRedemptionStatusResponse struct {
	Data []struct {
		BroadcasterName  string    `json:"broadcaster_name"`
		BroadcasterLogin string    `json:"broadcaster_login"`
		BroadcasterID    string    `json:"broadcaster_id"`
		ID               string    `json:"id"`
		UserID           string    `json:"user_id"`
		UserName         string    `json:"user_name"`
		UserLogin        string    `json:"user_login"`
		UserInput        string    `json:"user_input"`
		Status           string    `json:"status"`
		RedeemedAt       time.Time `json:"redeemed_at"`
		Reward           struct {
			ID     string `json:"id"`
			Title  string `json:"title"`
			Prompt string `json:"prompt"`
			Cost   int    `json:"cost"`
		} `json:"reward"`
	} `json:"data"`
}

func (c *HelixClient) UpdateRedemptionStatus(broadcasterID, rewardID string, redemptionID string, statusSuccess bool) error {
	token, err := c.db.GetUserAccessToken(broadcasterID)
	if err != nil {
		return fmt.Errorf("failed to get userAccess token to update redemption status for %s", broadcasterID)
	}

	request := UpdateRedemptionStatusRequest{}
	if statusSuccess {
		request.Status = "FULFILLED"
	} else {
		request.Status = "CANCELED"
	}

	marshalled, err := json.Marshal(request)
	if err != nil {
		return err
	}

	method := http.MethodPatch
	reqUrl, err := url.Parse("https://api.twitch.tv/helix/channel_points/custom_rewards/redemptions")
	if err != nil {
		return err
	}

	query := reqUrl.Query()

	query.Set("broadcaster_id", broadcasterID)
	query.Set("id", redemptionID)
	query.Set("reward_id", rewardID)

	reqUrl.RawQuery = query.Encode()

	req, err := http.NewRequest(method, reqUrl.String(), bytes.NewBuffer(marshalled))
	req.Header.Set("authorization", "Bearer "+token.AccessToken)
	req.Header.Set("client-id", c.clientID)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		log.Error(err)
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Error(err)
		return err
	}

	log.Infof("[%d][%s] %s %s", resp.StatusCode, method, reqUrl, marshalled)

	if resp.StatusCode >= 400 {
		var response ErrorResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			return fmt.Errorf("failed to unmarshal redemption status error response: %s", err.Error())
		}

		return fmt.Errorf("failed update redemption: %s", response.Message)
	}

	var response UpdateRedemptionStatusResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return err
	}

	if len(response.Data) != 1 {
		return fmt.Errorf("%d amount of redemptions returned after creation invalid", len(response.Data))
	}

	return nil
}

func (c *HelixClient) DeleteReward(userID string, userAccessToken string, rewardID string) error {
	method := http.MethodDelete
	reqUrl, err := url.Parse("https://api.twitch.tv/helix/channel_points/custom_rewards")
	if err != nil {
		return err
	}

	query := reqUrl.Query()
	query.Set("broadcaster_id", userID)
	query.Set("id", rewardID)

	reqUrl.RawQuery = query.Encode()

	req, err := http.NewRequest(method, reqUrl.String(), nil)
	req.Header.Set("authorization", "Bearer "+userAccessToken)
	req.Header.Set("client-id", c.clientID)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		log.Error(err)
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Error(err)
		return err
	}

	log.Infof("[%d][%s] %s", resp.StatusCode, method, reqUrl.String())

	if resp.StatusCode >= 400 {
		var response ErrorResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			return fmt.Errorf("failed to unmarshal reward error response: %s", err.Error())
		}

		return fmt.Errorf("failed to create reward: %s", response.Message)
	}

	return nil
}

func RewardStatusIsUnfullfilled(status string) bool {
	return strings.ToLower(status) == "unfulfilled"
}

func RewardStatusIsCancelled(status string) bool {
	return strings.ToLower(status) == "cancelled" || strings.ToLower(status) == "canceled"
}

func RewardStatusIsFullfilled(status string) bool {
	return strings.ToLower(status) == "fulfilled"
}
