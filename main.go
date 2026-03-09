package main

import (
	"log"
	"os"

	"github.com/PlakarKorp/docbot/bot"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	botToken := os.Getenv("BOT_TOKEN")
	guildID := os.Getenv("GUILD_ID")
	channelID := os.Getenv("DISCORD_CHANNEL_ID")
	maintainerDiscordID := os.Getenv("MAINTAINER_DISCORD_ID")
	dbPath := os.Getenv("DB_PATH")
	webBaseURL := os.Getenv("WEB_BASE_URL")

	if botToken == "" || guildID == "" || channelID == "" || maintainerDiscordID == "" || dbPath == "" || webBaseURL == "" {
		log.Printf("BOT_TOKEN, GUILD_ID, DISCORD_CHANNEL_ID, MAINTAINER_DISCORD_ID, DB_PATH, and WEB_BASE_URL are required")
		panic("missing required environment variables")
	}

	bot.RunBot(botToken, guildID, dbPath, webBaseURL, channelID, maintainerDiscordID)
}
