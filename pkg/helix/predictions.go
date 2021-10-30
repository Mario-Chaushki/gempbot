package helix

import (
	"fmt"
	"net/http"

	"github.com/gempir/gempbot/pkg/log"
	nickHelix "github.com/nicklaw5/helix/v2"
)

func (c *Client) GetPredictions(params *nickHelix.PredictionsParams) (*nickHelix.PredictionsResponse, error) {
	token, err := c.db.GetUserAccessToken(params.BroadcasterID)
	if err != nil {
		return &nickHelix.PredictionsResponse{}, fmt.Errorf("bot has no access token, broadcaster must login")
	}

	c.Client.SetUserAccessToken(token.AccessToken)
	resp, err := c.Client.GetPredictions(params)
	c.Client.SetUserAccessToken("")
	log.Infof("[%d] GetPredictions", resp.StatusCode)
	if err != nil {
		return resp, fmt.Errorf("could not get active predictions: %s", resp.ErrorMessage)
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return resp, fmt.Errorf("bot failed to manage predictions, broadcaster must login %s", resp.ErrorMessage)
	}
	if len(resp.Data.Predictions) < 1 {
		return resp, fmt.Errorf("no prediction active")
	}

	return resp, nil
}

func (c *Client) EndPrediction(params *nickHelix.EndPredictionParams) (*nickHelix.PredictionsResponse, error) {
	token, err := c.db.GetUserAccessToken(params.BroadcasterID)
	if err != nil {
		return &nickHelix.PredictionsResponse{}, fmt.Errorf("bot has no access token, broadcaster must login")
	}

	c.Client.SetUserAccessToken(token.AccessToken)
	resp, err := c.Client.EndPrediction(params)
	c.Client.SetUserAccessToken("")
	log.Infof("[%d] GetPredictions", resp.StatusCode)
	if err != nil {
		return resp, fmt.Errorf("could not set prediction outcome: %s", resp.ErrorMessage)
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return resp, fmt.Errorf("bot failed to manage predictions, broadcaster must login %s", resp.ErrorMessage)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return resp, fmt.Errorf("bad twitch api response %s", resp.ErrorMessage)
	}

	return resp, nil
}
