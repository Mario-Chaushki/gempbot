package bot

import (
	"strings"
	"time"

	"github.com/gempir/gempbot/pkg/bot/commander"
	"github.com/gempir/gempbot/pkg/chat"
	"github.com/gempir/gempbot/pkg/config"
	"github.com/gempir/gempbot/pkg/helixclient"
	"github.com/gempir/gempbot/pkg/log"
	"github.com/gempir/gempbot/pkg/store"
	"github.com/gempir/go-twitch-irc/v2"
)

// Bot basic logging bot
type Bot struct {
	startTime   time.Time
	cfg         *config.Config
	db          *store.Database
	helixClient *helixclient.Client
	listener    *commander.Listener
	Done        chan bool
	ChatClient  *chat.ChatClient
}

func NewBot(cfg *config.Config, db *store.Database, helixClient *helixclient.Client) *Bot {
	chatClient := chat.NewClient(cfg)

	handler := commander.NewHandler(cfg, helixClient, db, chatClient.Say)

	listener := commander.NewListener(db, handler, chatClient.Say)
	listener.RegisterDefaultCommands()

	return &Bot{
		Done:        make(chan bool),
		ChatClient:  chatClient,
		cfg:         cfg,
		db:          db,
		listener:    listener,
		helixClient: helixClient,
	}
}

func (b *Bot) Say(channel string, message string) {
	go b.ChatClient.Say(channel, message)
}

func (b *Bot) Join(channel string) {
	go b.ChatClient.Join(channel)
}

func (b *Bot) Part(channel string) {
	go b.ChatClient.Part(channel)
}

func (b *Bot) Connect() {
	b.startTime = time.Now()
	b.ChatClient.SetOnPrivateMessage(b.handlePrivateMessage)
	go b.ChatClient.Connect(b.joinBotConfigChannels)

	if strings.HasPrefix(b.cfg.Username, "justinfan") {
		log.Info("joining as anonymous")
	} else {
		log.Info("joining as user " + b.cfg.Username)
	}
	go b.ChatClient.Join(b.cfg.Username)
}

func (b *Bot) handlePrivateMessage(msg twitch.PrivateMessage) {
	sysMessage := msg.Channel == b.cfg.Username && msg.User.Name == b.cfg.Username && strings.Contains(msg.Message, b.cfg.Environment)
	if sysMessage {
		log.Infof("sysMessage: %s", msg.Message)
		if strings.HasPrefix(msg.Message, "JOIN "+b.cfg.Environment+" ") {
			b.ChatClient.Join(strings.TrimPrefix(msg.Message, "JOIN "+b.cfg.Environment+" "))
			return
		}

		if strings.HasPrefix(msg.Message, "PART "+b.cfg.Environment+" ") {
			b.ChatClient.Part(strings.TrimPrefix(msg.Message, "PART "+b.cfg.Environment+" "))
			return
		}
	}

	b.listener.HandlePrivateMessage(msg)
}

func (b *Bot) joinBotConfigChannels() {
	botConfigs := b.db.GetAllJoinBotConfigs()
	userIDs := []string{}
	for _, botConfig := range botConfigs {
		userIDs = append(userIDs, botConfig.OwnerTwitchID)
	}

	users, err := b.helixClient.GetUsersByUserIds(userIDs)
	if err != nil {
		log.Error(err)
	}

	b.ChatClient.WaitForConnect()
	log.Infof("joining %d channels", len(users))
	for _, user := range users {
		b.ChatClient.Join(user.Login)
	}
}
