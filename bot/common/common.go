package common

import (
	"context"
	"database/sql"

	"github.com/PlakarKorp/docbot/bot/db"
	"github.com/bwmarrin/discordgo"
)

const (
	ColorSuccess = 0x57F287
	ColorInfo    = 0x7C8AFF
	ColorError   = 0xED4245
)

var (
	Queries             *db.Queries
	DBConn              *sql.DB
	BaseURL             string
	ChannelID           string
	MaintainerDiscordID string
)

func SetConfig(conn *sql.DB, q *db.Queries, webBaseURL, channelID, maintainerDiscordID string) {
	DBConn = conn
	Queries = q
	BaseURL = webBaseURL
	ChannelID = channelID
	MaintainerDiscordID = maintainerDiscordID
}

func RespondEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, title, description string, color int) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{Title: title, Description: description, Color: color},
			},
		},
	})
}

func RespondError(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{Title: "Error", Description: msg, Color: ColorError},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

// IsAdmin returns true if the caller has Discord Administrator or Manage Server permission.
func IsAdmin(i *discordgo.InteractionCreate) bool {
	if i.Member == nil {
		return false
	}
	return i.Member.Permissions&discordgo.PermissionAdministrator != 0 ||
		i.Member.Permissions&discordgo.PermissionManageGuild != 0
}

// IsMaintainer returns true if the caller is the configured maintainer.
func IsMaintainer(i *discordgo.InteractionCreate) bool {
	return MaintainerDiscordID != "" && i.Member != nil && i.Member.User.ID == MaintainerDiscordID
}

// RequireAuthorized checks that the caller is a maintainer, Discord admin, or registered reviewer.
// Responds with an ephemeral error and returns false if not authorized.
func RequireAuthorized(s *discordgo.Session, i *discordgo.InteractionCreate) bool {
	if IsAdmin(i) || IsMaintainer(i) {
		return true
	}
	if i.Member != nil {
		_, err := Queries.GetReviewerByDiscordID(context.Background(), i.Member.User.ID)
		if err == nil {
			return true
		}
	}
	RespondError(s, i, "You don't have permission to use this command.")
	return false
}

func EditDeferredError(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	embeds := []*discordgo.MessageEmbed{
		{Title: "Error", Description: msg, Color: ColorError},
	}
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &embeds,
	})
}
