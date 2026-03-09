package commands

import (
	"github.com/bwmarrin/discordgo"
)

type Command struct {
	Definition *discordgo.ApplicationCommand
	Handler    func(s *discordgo.Session, i *discordgo.InteractionCreate)
}

var registered []Command

func Register(c Command) {
	registered = append(registered, c)
}

func Definitions() []*discordgo.ApplicationCommand {
	defs := make([]*discordgo.ApplicationCommand, len(registered))
	for i, c := range registered {
		defs[i] = c.Definition
	}
	return defs
}

func Handlers() map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	h := make(map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate), len(registered))
	for _, c := range registered {
		h[c.Definition.Name] = c.Handler
	}
	return h
}
