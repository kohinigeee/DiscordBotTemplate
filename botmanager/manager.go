package botmanager

import (
	"log/slog"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kohinigeee/DiscordBotTemplate/mylogger"
)

const (
	ImageColorHex = 0xdb8fd7
	isExcludeBOT  = true
)

var (
	logger *mylogger.LoggerItem
)

func init() {
	logger = mylogger.GetLogger("Botmanager")
}

type HandlerName string
type DiscorBotdHandler func(s *discordgo.Session, i *discordgo.InteractionCreate, manager *BotManager)

type BotManager struct {
	Session             *discordgo.Session
	BotUserInfo         *discordgo.User
	GuildID             string
	LastNotionBatchTime time.Time
	intrHandlersMap     map[HandlerName]DiscorBotdHandler
	appHandlersMap      map[HandlerName]DiscorBotdHandler
	modalHandlerMap     map[HandlerName]DiscorBotdHandler

	handlerMemory map[string]any //各コマンド内でハンドラーごとに一時的にデータを共有するためのメモリ

	//------各BOT特有のメンバ----------------
}

func NewBotManager(s *discordgo.Session) *BotManager {
	manager := &BotManager{
		Session:         s,
		intrHandlersMap: make(map[HandlerName]DiscorBotdHandler),
		appHandlersMap:  make(map[HandlerName]DiscorBotdHandler),
		modalHandlerMap: make(map[HandlerName]DiscorBotdHandler),
		GuildID:         os.Getenv("GUILD_ID"),

		handlerMemory: make(map[string]any),
	}

	// Add initial slash commands handler
	for _, slash := range InitialSlashCommands() {
		manager.AddAppHandler(HandlerName(slash.Command.Name), slash.Handler)
	}

	// Add initial interaction commands handler
	for _, interact := range InitialInteractCommands() {
		manager.AddIntrHandler(HandlerName(interact.Name), interact.Handler)
	}

	// Add initial modal commands handler
	for _, modal := range InitialDiscordModalCommands() {
		manager.AddModalHandler(HandlerName(modal.Name), modal.Handler)
	}

	manager.Session.AddHandler(manager.onInteractionCreate)
	return manager
}

func (manager *BotManager) Start() {
	manager.BotUserInfo = manager.Session.State.User
}

func (manager *BotManager) AddAppHandler(name HandlerName, handler DiscorBotdHandler) {
	manager.appHandlersMap[name] = handler
}

func (manager *BotManager) AddIntrHandler(name HandlerName, handler DiscorBotdHandler) {
	manager.intrHandlersMap[name] = handler
}

func (manager *BotManager) AddModalHandler(name HandlerName, handler DiscorBotdHandler) {
	manager.modalHandlerMap[name] = handler
}

func (manager *BotManager) SetHandlerMemory(commandName string, userID string, data any) {
	key := commandName + "_" + userID
	manager.handlerMemory[key] = data
}

func (manager *BotManager) DeleteHandlerMemory(commandName string, userID string) {
	key := commandName + "_" + userID
	delete(manager.handlerMemory, key)
}

func (manager *BotManager) GetHandlerMemory(commandName string, userID string) any {
	key := commandName + "_" + userID
	return manager.handlerMemory[key]
}

func (manager *BotManager) MakeNormalMessageEmbed(title string, msg string, fileds []*discordgo.MessageEmbedField) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: msg,
		Color:       0x00ff00,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: manager.BotUserInfo.AvatarURL("20"),
		},
	}

	if fileds != nil {
		embed.Fields = fileds
	}

	return embed
}

func (manager *BotManager) MakeErrorMessageEmbed(title string, msg string, fileds []*discordgo.MessageEmbedField) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: msg,
		Color:       0xff0000,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: manager.BotUserInfo.AvatarURL("20"),
		},
	}

	if fileds != nil {
		embed.Fields = fileds
	}

	return embed
}

func (manager *BotManager) SendNormalMessageInteractionEdit(i *discordgo.Interaction, title string, msg string, fileds []*discordgo.MessageEmbedField) {

	embed := manager.MakeNormalMessageEmbed(title, msg, fileds)

	_, err := manager.Session.InteractionResponseEdit(i, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})

	if err != nil {
		logger.Error("Error sending normal interaction message edit", slog.String("err", err.Error()))
	}
}

func (manager *BotManager) SendErrorMessageInteractionEdit(i *discordgo.Interaction, title string, msg string, fileds []*discordgo.MessageEmbedField) {

	embed := manager.MakeErrorMessageEmbed(title, msg, fileds)

	_, err := manager.Session.InteractionResponseEdit(i, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})

	if err != nil {
		logger.Error("Error sending error interaction message edit", slog.String("err", err.Error()))
	}
}

func (manager *BotManager) SendNormalMessageInteraction(i *discordgo.Interaction, msgType discordgo.InteractionResponseType, title string, msg string, fileds []*discordgo.MessageEmbedField) {

	embed := manager.MakeNormalMessageEmbed(title, msg, fileds)

	err := manager.Session.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: msgType,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})

	if err != nil {
		logger.Error("Error sending normal message", slog.String("err", err.Error()))
	}
}

func (manager *BotManager) SendErrorMessageInteracton(i *discordgo.Interaction, msgType discordgo.InteractionResponseType, title string, msg string, fileds []*discordgo.MessageEmbedField) {

	embed := manager.MakeErrorMessageEmbed(title, msg, fileds)

	err := manager.Session.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: msgType,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})

	if err != nil {
		logger.Error("Error sending error message", slog.String("err", err.Error()))
	}
}

func (manager *BotManager) SendNormalMessage(channelID string, title string, msg string, fileds []*discordgo.MessageEmbedField) {

	embed := manager.MakeNormalMessageEmbed(title, msg, fileds)

	_, err := manager.Session.ChannelMessageSendEmbed(channelID, embed)
	if err != nil {
		logger.Error("Error sending message", slog.String("err", err.Error()))
	}
}

func (manager *BotManager) SendErrorMessage(channelID string, title string, msg string, fileds []*discordgo.MessageEmbedField) {

	embed := manager.MakeErrorMessageEmbed(title, msg, fileds)

	_, err := manager.Session.ChannelMessageSendEmbed(channelID, embed)
	if err != nil {
		logger.Error("Error sending message", slog.String("err", err.Error()))
	}
}

// Discord event handler
func (manager *BotManager) onInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {

	switch i.Type {

	case discordgo.InteractionMessageComponent:
		handlerName := HandlerName(i.MessageComponentData().CustomID)
		handler, ok := manager.intrHandlersMap[handlerName]
		if !ok {
			logger.Warn("Handler not found", slog.String("handlerName", string(handlerName)))
			return
		}
		handler(s, i, manager)

	case discordgo.InteractionModalSubmit:
		handlerName := HandlerName(i.ModalSubmitData().CustomID)
		handler, ok := manager.modalHandlerMap[handlerName]
		if !ok {
			logger.Warn("Handler not found", slog.String("handlerName", string(handlerName)))
			return
		}
		handler(s, i, manager)

	case discordgo.InteractionApplicationCommand:
		handlerName := HandlerName(i.ApplicationCommandData().Name)
		handler, ok := manager.appHandlersMap[handlerName]
		if !ok {
			logger.Warn("Handler not found", slog.String("handlerName", string(handlerName)))
			return
		}
		handler(s, i, manager)

	default:

	}
}
