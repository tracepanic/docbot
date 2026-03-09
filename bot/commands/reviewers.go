package commands

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/PlakarKorp/docbot/bot/common"
	"github.com/PlakarKorp/docbot/bot/db"
	"github.com/bwmarrin/discordgo"
)

func handleReviewerList(s *discordgo.Session, i *discordgo.InteractionCreate) {
	reviewers, err := common.Queries.ListAllReviewers(context.Background())
	if err != nil {
		log.Printf("Failed to list reviewers: %v", err)
		common.RespondError(s, i, "Failed to get reviewers.")
		return
	}

	if len(reviewers) == 0 {
		common.RespondEmbed(s, i, "Reviewers", "No reviewers found.", common.ColorInfo)
		return
	}

	link := fmt.Sprintf("%s/reviewers", common.BaseURL)
	common.RespondEmbed(s, i, "Reviewers", fmt.Sprintf("You have **%d** reviewers.\n\n[View all reviewers](%s)", len(reviewers), link), common.ColorInfo)
}

func handleReviewerAdd(s *discordgo.Session, i *discordgo.InteractionCreate, sub *discordgo.ApplicationCommandInteractionDataOption) {
	if !common.RequireAuthorized(s, i) {
		return
	}
	user := sub.Options[0].UserValue(s)

	err := common.Queries.CreateReviewer(context.Background(), db.CreateReviewerParams{
		DiscordUserID: user.ID,
		Username:      user.Username,
	})
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			log.Printf("Reviewer %s already exists: %v", user.Username, err)
			common.RespondError(s, i, fmt.Sprintf("**%s** is already a reviewer.", user.Username))
			return
		}
		log.Printf("Failed to add reviewer %s: %v", user.Username, err)
		common.RespondError(s, i, "Failed to add reviewer.")
		return
	}

	common.RespondEmbed(s, i, "Reviewer Added", fmt.Sprintf("**%s** has been added as a reviewer.", user.Username), common.ColorSuccess)
}

func handleReviewerPause(s *discordgo.Session, i *discordgo.InteractionCreate, sub *discordgo.ApplicationCommandInteractionDataOption) {
	if !common.RequireAuthorized(s, i) {
		return
	}
	user := sub.Options[0].UserValue(s)

	reviewer, err := common.Queries.GetReviewerByDiscordID(context.Background(), user.ID)
	if err != nil {
		common.RespondError(s, i, fmt.Sprintf("**%s** is not a reviewer.", user.Username))
		return
	}
	if !reviewer.Active {
		common.RespondError(s, i, fmt.Sprintf("**%s** is already paused.", reviewer.Username))
		return
	}

	// Cancel all pending jobs assigned to this reviewer so docs return to pool
	jobs, err := common.Queries.ListPendingJobsByReviewer(context.Background(), reviewer.ID)
	if err != nil {
		log.Printf("Failed to list jobs for reviewer %d while pausing: %v", reviewer.ID, err)
	}
	for _, job := range jobs {
		if err := common.Queries.CancelReviewJob(context.Background(), job.ID); err != nil {
			log.Printf("Failed to cancel job %d while pausing reviewer: %v", job.ID, err)
		}
	}

	if err := common.Queries.DeactivateReviewer(context.Background(), reviewer.ID); err != nil {
		log.Printf("Failed to deactivate reviewer %s: %v", user.Username, err)
		common.RespondError(s, i, "Failed to pause reviewer.")
		return
	}

	msg := fmt.Sprintf("**%s** has been paused — they won't be assigned new reviews.", reviewer.Username)
	if len(jobs) > 0 {
		msg += fmt.Sprintf(" %d pending review(s) were cancelled and returned to the pool.", len(jobs))
	}
	common.RespondEmbed(s, i, "Reviewer Paused", msg, common.ColorInfo)
}

func handleReviewerResume(s *discordgo.Session, i *discordgo.InteractionCreate, sub *discordgo.ApplicationCommandInteractionDataOption) {
	if !common.RequireAuthorized(s, i) {
		return
	}
	user := sub.Options[0].UserValue(s)

	reviewer, err := common.Queries.GetReviewerByDiscordID(context.Background(), user.ID)
	if err != nil {
		common.RespondError(s, i, fmt.Sprintf("**%s** is not a reviewer.", user.Username))
		return
	}
	if reviewer.Active {
		common.RespondError(s, i, fmt.Sprintf("**%s** is already active.", reviewer.Username))
		return
	}

	if err := common.Queries.ActivateReviewer(context.Background(), reviewer.ID); err != nil {
		log.Printf("Failed to activate reviewer %s: %v", user.Username, err)
		common.RespondError(s, i, "Failed to resume reviewer.")
		return
	}

	common.RespondEmbed(s, i, "Reviewer Resumed", fmt.Sprintf("**%s** is now active and will be included in review assignments.", reviewer.Username), common.ColorSuccess)
}

func handleReviewerRemove(s *discordgo.Session, i *discordgo.InteractionCreate, sub *discordgo.ApplicationCommandInteractionDataOption) {
	if !common.RequireAuthorized(s, i) {
		return
	}
	user := sub.Options[0].UserValue(s)

	reviewer, err := common.Queries.GetReviewerByDiscordID(context.Background(), user.ID)
	if err != nil {
		common.RespondError(s, i, fmt.Sprintf("**%s** is not a reviewer.", user.Username))
		return
	}

	if err := common.Queries.DeleteReviewer(context.Background(), reviewer.ID); err != nil {
		log.Printf("Failed to delete reviewer %s: %v", user.Username, err)
		common.RespondError(s, i, "Failed to remove reviewer.")
		return
	}

	common.RespondEmbed(s, i, "Reviewer Removed", fmt.Sprintf("**%s** has been removed as a reviewer.", user.Username), common.ColorSuccess)
}
