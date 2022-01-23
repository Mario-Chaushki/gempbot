package emotechief

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"regexp"

	"github.com/carlmjohnson/requests"
	"github.com/gempir/gempbot/internal/channelpoint"
	"github.com/gempir/gempbot/internal/dto"
	"github.com/gempir/gempbot/internal/log"
	"github.com/gempir/gempbot/internal/store"
	"github.com/nicklaw5/helix/v2"
)

var sevenTvRegex = regexp.MustCompile(`https?:\/\/7tv.app\/emotes\/(\w*)`)

const sevenTvApiBaseUrl = "https://api.7tv.app/v2"

const (
	EmoteVisibilityPrivate int32 = 1 << iota
	EmoteVisibilityGlobal
	EmoteVisibilityUnlisted
	EmoteVisibilityOverrideBTTV
	EmoteVisibilityOverrideFFZ
	EmoteVisibilityOverrideTwitchGlobal
	EmoteVisibilityOverrideTwitchSubscriber
	EmoteVisibilityZeroWidth
	EmoteVisibilityPermanentlyUnlisted

	EmoteVisibilityAll int32 = (1 << iota) - 1
)

type SevenTvUserResponse struct {
	Data struct {
		User struct {
			ID     string `json:"id"`
			Emotes []struct {
				ID         string `json:"id"`
				Name       string `json:"name"`
				Status     int    `json:"status"`
				Visibility int    `json:"visibility"`
				Width      []int  `json:"width"`
				Height     []int  `json:"height"`
			} `json:"emotes"`
			EmoteSlots int `json:"emote_slots"`
		} `json:"user"`
	} `json:"data"`
}

func (ec *EmoteChief) VerifySetSevenTvEmote(channelUserID, emoteId, channel, redeemedByUsername string, slots int) (emoteAddType dto.EmoteChangeType, removalTargetEmoteId string, err error) {
	if ec.db.IsEmoteBlocked(channelUserID, emoteId, dto.REWARD_SEVENTV) {
		return dto.EMOTE_ADD_ADD, "", errors.New("Emote is blocked")
	}

	nextEmote, err := ec.sevenTvClient.GetEmote(emoteId)
	if err != nil {
		return
	}

	user, err := ec.sevenTvClient.GetUser(channelUserID)
	if err != nil {
		return
	}

	for _, emote := range user.Emotes {
		if emote.Code == nextEmote.Code {
			return dto.EMOTE_ADD_ADD, "", fmt.Errorf("Emote code \"%s\" already added", newEmote.Name)
		}
	}
	log.Infof("Current 7tv emotes: %d/%d", len(user.Emotes), user.EmoteSlots)

	emotesAdded := ec.db.GetEmoteAdded(channelUserID, dto.REWARD_SEVENTV, slots)
	log.Infof("Total Previous emotes %d in %s", len(emotesAdded), channelUserID)

	if len(emotesAdded) > 0 {
		oldestEmote := emotesAdded[len(emotesAdded)-1]
		if !oldestEmote.Blocked {
			for _, sharedEmote := range user.Emotes {
				if oldestEmote.EmoteID == sharedEmote.ID {
					removalTargetEmoteId = oldestEmote.EmoteID
					log.Infof("Found removal target %s in %s", removalTargetEmoteId, channelUserID)
				}
			}
		} else {
			log.Infof("Removal target %s is already blocked, so already removed, skipping removal", oldestEmote.EmoteID)
		}
	}

	emoteAddType = dto.EMOTE_ADD_REMOVED_PREVIOUS
	if removalTargetEmoteId == "" && len(user.Emotes) >= user.EmoteSlots {
		if len(user.Emotes) == 0 {
			return dto.EMOTE_ADD_ADD, "", errors.New("emotes limit reached and can't find amount of emotes added to choose random")
		}

		emoteAddType = dto.EMOTE_ADD_REMOVED_RANDOM
		log.Infof("Didn't find previous emote history of %d emotes and limit reached, choosing random in %s", slots, channelUserID)
		removalTargetEmoteId = user.Emotes[rand.Intn(len(user.Emotes))].ID
	}

	return
}

func (ec *EmoteChief) RemoveSevenTvEmote(channelUserID, emoteID string) (*sevenTvEmote, error) {
	var userData SevenTvUserResponse
	err := ec.QuerySevenTvGQL(SEVEN_TV_USER_DATA_QUERY, map[string]interface{}{"id": channelUserID}, &userData)
	if err != nil {
		return nil, err
	}

	var empty struct{}
	err = ec.QuerySevenTvGQL(
		SEVEN_TV_DELETE_EMOTE_QUERY,
		map[string]interface{}{
			"ch": userData.Data.User.ID,
			"re": "blocked emote",
			"em": emoteID,
		}, &empty,
	)
	if err != nil {
		return nil, err
	}

	ec.db.CreateEmoteAdd(channelUserID, dto.REWARD_SEVENTV, emoteID, dto.EMOTE_ADD_REMOVED_BLOCKED)

	return getSevenTvEmote(emoteID)
}

func (ec *EmoteChief) SetSevenTvEmote(channelUserID, emoteId, channel, redeemedByUsername string, slots int) error {
	emoteAddType, removalTargetEmoteId, err := ec.VerifySetSevenTvEmote(channelUserID, emoteId, channel, redeemedByUsername, slots)
	if err != nil {
		return err
	}

	// do we need to remove the emote?
	if removalTargetEmoteId != "" {
		err := ec.sevenTvClient.RemoveEmote(channelUserID, removalTargetEmoteId)
		if err != nil {
			return err
		}

		ec.db.CreateEmoteAdd(channelUserID, dto.REWARD_SEVENTV, removalTargetEmoteId, emoteAddType)
	}

	err = ec.sevenTvClient.AddEmote(channelUserID, emoteId)
	if err != nil {
		return err
	}

	ec.db.CreateEmoteAdd(channelUserID, dto.REWARD_SEVENTV, emoteId, dto.EMOTE_ADD_ADD)

	return nil
}

const SEVEN_TV_ADD_EMOTE_QUERY = `mutation AddChannelEmote($ch: String!, $em: String!, $re: String!) {addChannelEmote(channel_id: $ch, emote_id: $em, reason: $re) {emote_ids}}`
const SEVEN_TV_DELETE_EMOTE_QUERY = `mutation RemoveChannelEmote($ch: String!, $em: String!, $re: String!) {removeChannelEmote(channel_id: $ch, emote_id: $em, reason: $re) {emote_ids}}`
const SEVEN_TV_USER_DATA_QUERY = `
query GetUser($id: String!) {
	user(id: $id) {
	  ...FullUser
	}
  }
  
fragment FullUser on User {
	id
	emotes {
		id
		name
		status
		visibility
		width
		height
	}
	emote_slots
}
`

type GqlQuery struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

const SEVEN_TV_API = "https://api.7tv.app/v2/gql"

func (ec *EmoteChief) QuerySevenTvGQL(query string, variables map[string]interface{}, response interface{}) error {
	gqlQuery := GqlQuery{Query: query, Variables: variables}

	err := requests.
		URL(SEVEN_TV_API).
		BodyJSON(gqlQuery).
		Bearer(ec.cfg.SevenTvToken).
		ToJSON(&response).
		Fetch(context.Background())
	if err != nil {
		log.Infof("7tv query '%s' with '%v' resp: '%v'", query, variables, response)
		return err
	}

	log.Infof("7tv query '%s' with '%v' resp: '%v'", query, variables, response)

	return nil
}

func GetSevenTvEmoteId(message string) (string, error) {
	matches := sevenTvRegex.FindAllStringSubmatch(message, -1)

	if len(matches) == 1 && len(matches[0]) == 2 {
		return matches[0][1], nil
	}

	return "", errors.New("no 7tv emote link found")
}

func (ec *EmoteChief) VerifySeventvRedemption(reward store.ChannelPointReward, redemption helix.EventSubChannelPointsCustomRewardRedemptionEvent) bool {
	opts := channelpoint.UnmarshallSevenTvAdditionalOptions(reward.AdditionalOptions)

	emoteID, err := GetSevenTvEmoteId(redemption.UserInput)
	if err == nil {
		_, _, _, _, err := ec.VerifySetSevenTvEmote(redemption.BroadcasterUserID, emoteID, redemption.BroadcasterUserLogin, redemption.UserLogin, opts.Slots)
		if err != nil {
			log.Warnf("7tv error %s %s", redemption.BroadcasterUserLogin, err)
			ec.chatClient.Say(redemption.BroadcasterUserLogin, fmt.Sprintf("⚠️ Failed to add 7tv emote from @%s error: %s", redemption.UserName, err.Error()))
			return false
		}

		return true
	}

	ec.chatClient.Say(redemption.BroadcasterUserLogin, fmt.Sprintf("⚠️ Failed to add 7tv emote from @%s error: %s", redemption.UserName, err.Error()))
	return false
}

func (ec *EmoteChief) HandleSeventvRedemption(reward store.ChannelPointReward, redemption helix.EventSubChannelPointsCustomRewardRedemptionEvent, updateStatus bool) {
	opts := channelpoint.UnmarshallSevenTvAdditionalOptions(reward.AdditionalOptions)
	success := false

	emoteID, err := GetSevenTvEmoteId(redemption.UserInput)
	if err == nil {
		emoteAdded, emoteRemoved, err := ec.SetSevenTvEmote(redemption.BroadcasterUserID, emoteID, redemption.BroadcasterUserLogin, redemption.UserName, opts.Slots)
		if err != nil {
			log.Warnf("7tv error %s %s", redemption.BroadcasterUserLogin, err)
			ec.chatClient.Say(redemption.BroadcasterUserLogin, fmt.Sprintf("⚠️ Failed to add 7tv emote from @%s error: %s", redemption.UserName, err.Error()))
		} else if emoteAdded != nil && emoteRemoved != nil {
			success = true
			ec.chatClient.Say(redemption.BroadcasterUserLogin, fmt.Sprintf("✅ Added new 7tv emote %s redeemed by @%s removed: %s", emoteAdded.Name, redemption.UserName, emoteRemoved.Name))
		} else if emoteAdded != nil {
			success = true
			ec.chatClient.Say(redemption.BroadcasterUserLogin, fmt.Sprintf("✅ Added new 7tv emote %s redeemed by @%s", emoteAdded.Name, redemption.UserName))
		} else {
			success = true
			ec.chatClient.Say(redemption.BroadcasterUserLogin, fmt.Sprintf("✅ Added new 7tv emote [unknown] redeemed by @%s", redemption.UserName))
		}
	} else {
		ec.chatClient.Say(redemption.BroadcasterUserLogin, fmt.Sprintf("⚠️ Failed to add 7tv emote from @%s error: %s", redemption.UserName, err.Error()))
	}

	if redemption.UserID == dto.GEMPIR_USER_ID {
		return
	}

	if updateStatus {
		err := ec.helixClient.UpdateRedemptionStatus(redemption.BroadcasterUserID, redemption.Reward.ID, redemption.ID, success)
		if err != nil {
			log.Errorf("Failed to update redemption status %s", err.Error())
			return
		}
	}
}

type sevenTvEmote struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Owner struct {
		ID          string `json:"id"`
		TwitchID    string `json:"twitch_id"`
		Login       string `json:"login"`
		DisplayName string `json:"display_name"`
		Role        struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			Position int    `json:"position"`
			Color    int    `json:"color"`
			Allowed  int    `json:"allowed"`
			Denied   int    `json:"denied"`
			Default  bool   `json:"default"`
		} `json:"role"`
	} `json:"owner"`
	Visibility       int           `json:"visibility"`
	VisibilitySimple []interface{} `json:"visibility_simple"`
	Mime             string        `json:"mime"`
	Status           int           `json:"status"`
	Tags             []interface{} `json:"tags"`
	Width            []int         `json:"width"`
	Height           []int         `json:"height"`
	Urls             [][]string    `json:"urls"`
}

func getSevenTvEmote(emoteID string) (*sevenTvEmote, error) {
	if emoteID == "" {
		return nil, nil
	}

	response, err := http.Get(sevenTvApiBaseUrl + "/emotes/" + emoteID)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	if response.StatusCode <= 100 || response.StatusCode >= 400 {
		return nil, fmt.Errorf("Bad 7tv response: %d", response.StatusCode)
	}

	var emoteResponse sevenTvEmote
	err = json.NewDecoder(response.Body).Decode(&emoteResponse)
	if err != nil {
		return nil, err
	}

	return &emoteResponse, nil
}
