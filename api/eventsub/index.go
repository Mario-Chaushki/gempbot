package eventsub

import (
	"net/http"

	"github.com/gempir/gempbot/pkg/channelpoint"
	"github.com/gempir/gempbot/pkg/chat"
	"github.com/gempir/gempbot/pkg/config"
	"github.com/gempir/gempbot/pkg/emotechief"
	"github.com/gempir/gempbot/pkg/eventsub"
	"github.com/gempir/gempbot/pkg/helix"
	"github.com/gempir/gempbot/pkg/store"
	nickHelix "github.com/nicklaw5/helix/v2"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	cfg := config.FromEnv()
	db := store.NewDatabase(cfg)
	helixClient := helix.NewClient(cfg, db)
	emoteChief := emotechief.NewEmoteChief(cfg, db)
	chatClient := chat.NewClient(cfg)
	go chatClient.Connect()
	channelPointManager := channelpoint.NewChannelPointManager(cfg, helixClient, db, emoteChief, chatClient)
	eventSubManager := eventsub.NewEventSubManager(cfg, helixClient, db, channelPointManager, chatClient)

	event, err := eventSubManager.HandleWebhook(w, r)
	if err != nil || len(event) == 0 {
		if err != nil {
			http.Error(w, err.Error(), err.Status())
		}
		return
	}

	if r.URL.Query().Get("type") == nickHelix.EventSubTypeChannelPointsCustomRewardRedemptionAdd {
		eventSubManager.HandleChannelPointsCustomRewardRedemption(event)
		return
	}
	if r.URL.Query().Get("type") == nickHelix.EventSubTypeChannelPredictionBegin {
		eventSubManager.HandlePredictionBegin(event)
		return
	}
	if r.URL.Query().Get("type") == nickHelix.EventSubTypeChannelPredictionLock {
		eventSubManager.HandlePredictionLock(event)
		return
	}
	if r.URL.Query().Get("type") == nickHelix.EventSubTypeChannelPredictionEnd {
		eventSubManager.HandlePredictionEnd(event)
		return
	}

	http.Error(w, "Invalid event type", http.StatusBadRequest)
}
