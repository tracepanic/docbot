package commands

import (
	"github.com/bwmarrin/discordgo"
)

func init() {
	Register(Command{
		Definition: &discordgo.ApplicationCommand{
			Name:        "doc",
			Description: "Doc review management",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "reviewer",
					Description: "Manage reviewers",
					Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "add",
							Description: "Add a reviewer",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:        "user",
									Description: "Discord user to add as reviewer",
									Type:        discordgo.ApplicationCommandOptionUser,
									Required:    true,
								},
							},
						},
						{
							Name:        "list",
							Description: "List all reviewers",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
						},
						{
							Name:        "remove",
							Description: "Remove a reviewer",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:        "user",
									Description: "Discord user to remove",
									Type:        discordgo.ApplicationCommandOptionUser,
									Required:    true,
								},
							},
						},
						{
							Name:        "pause",
							Description: "Pause a reviewer (won't be assigned new reviews)",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:        "user",
									Description: "Discord user to pause",
									Type:        discordgo.ApplicationCommandOptionUser,
									Required:    true,
								},
							},
						},
						{
							Name:        "resume",
							Description: "Resume a paused reviewer",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:        "user",
									Description: "Discord user to resume",
									Type:        discordgo.ApplicationCommandOptionUser,
									Required:    true,
								},
							},
						},
					},
				},
				{
					Name:        "pages",
					Description: "Manage documentation pages",
					Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "add",
							Description: "Add a single documentation page",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
						},
						{
							Name:        "list",
							Description: "List all documentation pages",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
						},
						{
							Name:        "import",
							Description: "Import documentation pages from a YAML file",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
						},
						{
							Name:        "mine",
							Description: "Show pages currently assigned to you",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
						},
						{
							Name:        "pending",
							Description: "List all pages with active pending reviews",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
						},
						{
							Name:        "ko-list",
							Description: "List all pages flagged KO awaiting a fix",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
						},
						{
							Name:        "info",
							Description: "Show full status of a page",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:        "id",
									Description: "The page ID",
									Type:        discordgo.ApplicationCommandOptionInteger,
									Required:    true,
								},
							},
						},
						{
							Name:        "assign",
							Description: "Manually assign a page to a specific reviewer",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:        "id",
									Description: "The page ID to assign",
									Type:        discordgo.ApplicationCommandOptionInteger,
									Required:    true,
								},
								{
									Name:        "user",
									Description: "Reviewer to assign to",
									Type:        discordgo.ApplicationCommandOptionUser,
									Required:    true,
								},
							},
						},
						{
							Name:        "delete",
							Description: "Delete a documentation page by ID",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:        "id",
									Description: "The page ID to delete",
									Type:        discordgo.ApplicationCommandOptionInteger,
									Required:    true,
								},
							},
						},
						{
							Name:        "ok",
							Description: "Mark a page as reviewed (OK)",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:        "id",
									Description: "The page ID to mark as OK",
									Type:        discordgo.ApplicationCommandOptionInteger,
									Required:    true,
								},
							},
						},
						{
							Name:        "ko",
							Description: "Mark a page as needing fixes (KO)",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:        "id",
									Description: "The page ID to mark as KO",
									Type:        discordgo.ApplicationCommandOptionInteger,
									Required:    true,
								},
							},
						},
						{
							Name:        "skip",
							Description: "Skip a review assigned to you",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:        "id",
									Description: "The page ID to skip",
									Type:        discordgo.ApplicationCommandOptionInteger,
									Required:    true,
								},
							},
						},
						{
							Name:        "pause",
							Description: "Pause a page (won't be assigned for review)",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:        "id",
									Description: "The page ID to pause",
									Type:        discordgo.ApplicationCommandOptionInteger,
									Required:    true,
								},
							},
						},
						{
							Name:        "resume",
							Description: "Resume a paused page",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:        "id",
									Description: "The page ID to resume",
									Type:        discordgo.ApplicationCommandOptionInteger,
									Required:    true,
								},
							},
						},
						{
							Name:        "fixed",
							Description: "Mark a KO'd page as fixed (maintainer)",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:        "id",
									Description: "The page ID that was fixed",
									Type:        discordgo.ApplicationCommandOptionInteger,
									Required:    true,
								},
							},
						},
					},
				},
			},
		},
		Handler: handleDoc,
	})
}

func handleDoc(s *discordgo.Session, i *discordgo.InteractionCreate) {
	group := i.ApplicationCommandData().Options[0]
	sub := group.Options[0]

	switch group.Name {
	case "reviewer":
		switch sub.Name {
		case "add":
			handleReviewerAdd(s, i, sub)
		case "list":
			handleReviewerList(s, i)
		case "remove":
			handleReviewerRemove(s, i, sub)
		case "pause":
			handleReviewerPause(s, i, sub)
		case "resume":
			handleReviewerResume(s, i, sub)
		}
	case "pages":
		switch sub.Name {
		case "add":
			handlePagesAdd(s, i)
		case "list":
			handlePagesList(s, i)
		case "import":
			handlePagesImport(s, i)
		case "mine":
			handlePagesMine(s, i)
		case "pending":
			handlePagesPending(s, i)
		case "ko-list":
			handlePagesKOList(s, i)
		case "info":
			handlePagesInfo(s, i, sub)
		case "assign":
			handlePagesAssign(s, i, sub)
		case "delete":
			handlePagesDelete(s, i, sub)
		case "ok":
			handlePagesOK(s, i, sub)
		case "ko":
			handlePagesKO(s, i, sub)
		case "skip":
			handlePagesSkip(s, i, sub)
		case "pause":
			handlePagesPause(s, i, sub)
		case "resume":
			handlePagesResume(s, i, sub)
		case "fixed":
			handlePagesFixed(s, i, sub)
		}
	}
}
