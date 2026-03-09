package bot

import (
	"log"
	"os"
	"os/signal"

	"github.com/PlakarKorp/docbot/bot/commands"
	"github.com/PlakarKorp/docbot/bot/common"
	"github.com/PlakarKorp/docbot/bot/db"
	"github.com/PlakarKorp/docbot/bot/scheduler"
	"github.com/PlakarKorp/docbot/bot/web"
	"github.com/bwmarrin/discordgo"
)

func RunBot(token, guildID, dbPath, webBaseURL, channelID, maintainerDiscordID string) {
	conn := db.RunMigrations(dbPath)
	defer conn.Close()

	q := db.New(conn)
	common.SetConfig(conn, q, webBaseURL, channelID, maintainerDiscordID)

	discord, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal(err)
	}

	discord.Identify.Intents = discordgo.IntentsGuilds

	// Register slash command and modal submit handlers
	handlers := commands.Handlers()
	discord.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			if h, ok := handlers[i.ApplicationCommandData().Name]; ok {
				h(s, i)
			}
		case discordgo.InteractionModalSubmit:
			if commands.IsDocPageModal(i.ModalSubmitData().CustomID) {
				commands.HandleModalSubmit(s, i)
			}
		}
	})

	err = discord.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer discord.Close()

	// Register slash commands with Discord
	for _, cmd := range commands.Definitions() {
		_, err := discord.ApplicationCommandCreate(discord.State.User.ID, guildID, cmd)
		if err != nil {
			log.Printf("Failed to register command %s: %v", cmd.Name, err)
		}
	}

	// Start scheduler
	scheduler.Start(discord)

	// Start web server
	webServer := web.New(":8080", q)

	go func() {
		log.Printf("Web server listening on :8080")
		if err := webServer.Start(); err != nil {
			log.Printf("Web server error: %v", err)
		}
	}()

	log.Println("Doc bot is running...")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}
