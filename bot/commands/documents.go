package commands

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PlakarKorp/docbot/bot/common"
	"github.com/PlakarKorp/docbot/bot/db"
	"github.com/bwmarrin/discordgo"
	"gopkg.in/yaml.v3"
)

const (
	modalAddPageID    = "doc_pages_add"
	modalImportPageID = "doc_pages_import"
	modalKOPagePrefix = "doc_pages_ko:"
)

func handlePagesList(s *discordgo.Session, i *discordgo.InteractionCreate) {
	docs, err := common.Queries.ListAllDocuments(context.Background())
	if err != nil {
		log.Printf("Failed to list documents: %v", err)
		common.RespondError(s, i, "Failed to get documents.")
		return
	}

	if len(docs) == 0 {
		common.RespondEmbed(s, i, "Documentation Pages", "No pages found.", common.ColorInfo)
		return
	}

	link := fmt.Sprintf("%s/pages", common.BaseURL)
	common.RespondEmbed(s, i, "Documentation Pages", fmt.Sprintf("You have **%d** pages.\n\n[View all pages](%s)", len(docs), link), common.ColorInfo)
}

func handlePagesAdd(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !common.RequireAuthorized(s, i) {
		return
	}
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: modalAddPageID,
			Title:    "Add Documentation Page",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "title",
							Label:       "Page Title",
							Style:       discordgo.TextInputShort,
							Placeholder: "e.g. Getting Started Guide",
							Required:    true,
							MinLength:   1,
							MaxLength:   200,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "url",
							Label:       "Page URL",
							Style:       discordgo.TextInputShort,
							Placeholder: "e.g. https://docs.example.com/getting-started",
							Required:    true,
							MinLength:   1,
							MaxLength:   500,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "interval",
							Label:       "Review Interval (days)",
							Style:       discordgo.TextInputShort,
							Placeholder: "e.g. 30",
							Required:    true,
							MinLength:   1,
							MaxLength:   5,
						},
					},
				},
			},
		},
	})
	if err != nil {
		log.Printf("Failed to show add page modal: %v", err)
	}
}

func IsDocPageModal(customID string) bool {
	return customID == modalAddPageID || customID == modalImportPageID || strings.HasPrefix(customID, modalKOPagePrefix)
}

func HandleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	switch {
	case data.CustomID == modalAddPageID:
		handleAddPageSubmit(s, i, data)
	case data.CustomID == modalImportPageID:
		handleImportPageSubmit(s, i, data)
	case strings.HasPrefix(data.CustomID, modalKOPagePrefix):
		handleKOSubmit(s, i, data)
	}
}

func handleAddPageSubmit(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ModalSubmitInteractionData) {
	if !common.RequireAuthorized(s, i) {
		return
	}
	var title, url, intervalStr string

	for _, row := range data.Components {
		ar, ok := row.(*discordgo.ActionsRow)
		if !ok {
			continue
		}
		for _, comp := range ar.Components {
			input, ok := comp.(*discordgo.TextInput)
			if !ok {
				continue
			}
			switch input.CustomID {
			case "title":
				title = input.Value
			case "url":
				url = input.Value
			case "interval":
				intervalStr = input.Value
			}
		}
	}

	interval, err := strconv.ParseInt(strings.TrimSpace(intervalStr), 10, 64)
	if err != nil || interval <= 0 {
		common.RespondError(s, i, "Invalid interval. Please enter a positive number of days.")
		return
	}

	err = common.Queries.CreateDocument(context.Background(), db.CreateDocumentParams{
		Url:          strings.TrimSpace(url),
		Title:        strings.TrimSpace(title),
		IntervalDays: interval,
	})
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			common.RespondError(s, i, fmt.Sprintf("A document with URL `%s` already exists.", url))
			return
		}
		log.Printf("Failed to create document: %v", err)
		common.RespondError(s, i, "Failed to add document.")
		return
	}

	common.RespondEmbed(s, i, "Page Added", fmt.Sprintf("[%s](%s) — %dd", title, url, interval), common.ColorSuccess)
}

func handlePagesDelete(s *discordgo.Session, i *discordgo.InteractionCreate, sub *discordgo.ApplicationCommandInteractionDataOption) {
	if !common.RequireAuthorized(s, i) {
		return
	}
	id := sub.Options[0].IntValue()

	doc, err := common.Queries.GetDocument(context.Background(), id)
	if err != nil {
		common.RespondError(s, i, fmt.Sprintf("No page found with ID `%d`.", id))
		return
	}

	tx, err := common.DBConn.BeginTx(context.Background(), nil)
	if err != nil {
		log.Printf("Failed to start transaction: %v", err)
		common.RespondError(s, i, "Failed to delete page.")
		return
	}
	defer tx.Rollback()

	txQueries := common.Queries.WithTx(tx)

	if err := txQueries.DeleteReviewJobsByDocument(context.Background(), id); err != nil {
		log.Printf("Failed to delete review jobs for document %d: %v", id, err)
		common.RespondError(s, i, "Failed to delete page.")
		return
	}

	if err := txQueries.DeleteDocument(context.Background(), id); err != nil {
		log.Printf("Failed to delete document %d: %v", id, err)
		common.RespondError(s, i, "Failed to delete page.")
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit deletion: %v", err)
		common.RespondError(s, i, "Failed to delete page.")
		return
	}

	common.RespondEmbed(s, i, "Page Deleted", fmt.Sprintf("Deleted **%s** (ID: %d)", doc.Title, doc.ID), common.ColorSuccess)
}

func handlePagesOK(s *discordgo.Session, i *discordgo.InteractionCreate, sub *discordgo.ApplicationCommandInteractionDataOption) {
	if !common.RequireAuthorized(s, i) {
		return
	}
	id := sub.Options[0].IntValue()

	doc, err := common.Queries.GetDocument(context.Background(), id)
	if err != nil {
		common.RespondError(s, i, fmt.Sprintf("No page found with ID `%d`.", id))
		return
	}

	if err := common.Queries.UpdateDocumentReview(context.Background(), id); err != nil {
		log.Printf("Failed to update document review %d: %v", id, err)
		common.RespondError(s, i, "Failed to mark page as OK.")
		return
	}

	// Complete pending review job if one exists
	job, err := common.Queries.GetPendingJobForDocument(context.Background(), id)
	if err == nil {
		callerDiscordID := i.Member.User.ID
		originalReviewer, revErr := common.Queries.GetReviewer(context.Background(), job.ReviewerID)
		if revErr == nil && originalReviewer.DiscordUserID != callerDiscordID {
			dmCh, dmErr := s.UserChannelCreate(originalReviewer.DiscordUserID)
			if dmErr == nil {
				embed := &discordgo.MessageEmbed{
					Title:       "Review Completed by Someone Else",
					Description: fmt.Sprintf("**%s** reviewed [%s](%s) — you've been unassigned.", i.Member.User.Username, doc.Title, doc.Url),
					Color:       common.ColorInfo,
				}
				if _, err := s.ChannelMessageSendEmbed(dmCh.ID, embed); err != nil {
					log.Printf("Failed to DM unassigned reviewer %s: %v", originalReviewer.Username, err)
				}
			}
		}
		if err := common.Queries.CompleteReviewJob(context.Background(), job.ID); err != nil {
			log.Printf("Failed to complete review job %d: %v", job.ID, err)
		}
	}

	common.RespondEmbed(s, i, "Page OK", fmt.Sprintf("Reviewed **%s** (ID: %d) — next review in %dd", doc.Title, doc.ID, doc.IntervalDays), common.ColorSuccess)
}

func handlePagesKO(s *discordgo.Session, i *discordgo.InteractionCreate, sub *discordgo.ApplicationCommandInteractionDataOption) {
	if !common.RequireAuthorized(s, i) {
		return
	}
	id := sub.Options[0].IntValue()

	_, err := common.Queries.GetDocument(context.Background(), id)
	if err != nil {
		common.RespondError(s, i, fmt.Sprintf("No page found with ID `%d`.", id))
		return
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("%s%d", modalKOPagePrefix, id),
			Title:    "Report Issue",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "notes",
							Label:       "What's wrong with this page?",
							Style:       discordgo.TextInputParagraph,
							Placeholder: "Describe the issues...",
							Required:    true,
							MinLength:   1,
							MaxLength:   2000,
						},
					},
				},
			},
		},
	})
	if err != nil {
		log.Printf("Failed to show KO modal: %v", err)
	}
}

func handleKOSubmit(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ModalSubmitInteractionData) {
	if !common.RequireAuthorized(s, i) {
		return
	}
	idStr := strings.TrimPrefix(data.CustomID, modalKOPagePrefix)
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		common.RespondError(s, i, "Invalid page ID.")
		return
	}

	doc, err := common.Queries.GetDocument(context.Background(), id)
	if err != nil {
		common.RespondError(s, i, fmt.Sprintf("No page found with ID `%d`.", id))
		return
	}

	var notes string
	for _, row := range data.Components {
		ar, ok := row.(*discordgo.ActionsRow)
		if !ok {
			continue
		}
		for _, comp := range ar.Components {
			input, ok := comp.(*discordgo.TextInput)
			if !ok {
				continue
			}
			if input.CustomID == "notes" {
				notes = strings.TrimSpace(input.Value)
			}
		}
	}

	if err := common.Queries.IncrementDocumentReviewCount(context.Background(), id); err != nil {
		log.Printf("Failed to increment review count for document %d: %v", id, err)
		common.RespondError(s, i, "Failed to mark page as KO.")
		return
	}

	// Mark pending review job as KO if one exists
	job, err := common.Queries.GetPendingJobForDocument(context.Background(), id)
	if err == nil {
		callerDiscordID := i.Member.User.ID
		originalReviewer, revErr := common.Queries.GetReviewer(context.Background(), job.ReviewerID)
		if revErr == nil && originalReviewer.DiscordUserID != callerDiscordID {
			dmCh, dmErr := s.UserChannelCreate(originalReviewer.DiscordUserID)
			if dmErr == nil {
				embed := &discordgo.MessageEmbed{
					Title:       "Review Completed by Someone Else",
					Description: fmt.Sprintf("**%s** flagged [%s](%s) as KO — you've been unassigned.", i.Member.User.Username, doc.Title, doc.Url),
					Color:       common.ColorInfo,
				}
				if _, err := s.ChannelMessageSendEmbed(dmCh.ID, embed); err != nil {
					log.Printf("Failed to DM unassigned reviewer %s: %v", originalReviewer.Username, err)
				}
			}
		}
		if err := common.Queries.CompleteReviewJobKO(context.Background(), db.CompleteReviewJobKOParams{
			Notes: sql.NullString{String: notes, Valid: true},
			ID:    job.ID,
		}); err != nil {
			log.Printf("Failed to KO review job %d: %v", job.ID, err)
		}
	}

	// Ephemeral confirmation to the reviewer
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{Title: "Feedback Received", Description: fmt.Sprintf("Got your feedback on **%s** — it's been posted to #documentation.", doc.Title), Color: common.ColorInfo},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	// Public post in #documentation
	if common.ChannelID != "" {
		user := i.Member.User.Username
		embed := &discordgo.MessageEmbed{
			Title:       "Page Needs Fixing",
			Description: fmt.Sprintf("[%s](%s)\n\n**%s** flagged this page:\n> %s", doc.Title, doc.Url, user, notes),
			Color:       common.ColorError,
		}
		if _, err := s.ChannelMessageSendEmbed(common.ChannelID, embed); err != nil {
			log.Printf("Failed to post KO to channel: %v", err)
		}
	}

	// DM the maintainer
	if common.MaintainerDiscordID != "" {
		dmChannel, err := s.UserChannelCreate(common.MaintainerDiscordID)
		if err == nil {
			embed := &discordgo.MessageEmbed{
				Title:       "Page KO",
				Description: fmt.Sprintf("[%s](%s) needs fixing.\n\n**Feedback:** %s\n\nUse `/doc pages fixed id:%d` when done.", doc.Title, doc.Url, notes, doc.ID),
				Color:       common.ColorError,
			}
			if _, err := s.ChannelMessageSendEmbed(dmChannel.ID, embed); err != nil {
				log.Printf("Failed to DM maintainer about KO: %v", err)
			}
		}
	}
}

func handlePagesSkip(s *discordgo.Session, i *discordgo.InteractionCreate, sub *discordgo.ApplicationCommandInteractionDataOption) {
	if !common.RequireAuthorized(s, i) {
		return
	}
	id := sub.Options[0].IntValue()

	doc, err := common.Queries.GetDocument(context.Background(), id)
	if err != nil {
		common.RespondError(s, i, fmt.Sprintf("No page found with ID `%d`.", id))
		return
	}

	job, err := common.Queries.GetPendingJobForDocument(context.Background(), id)
	if err != nil {
		common.RespondError(s, i, fmt.Sprintf("No pending review for **%s**.", doc.Title))
		return
	}

	// Check the job is assigned to the caller
	discordUserID := i.Member.User.ID
	reviewer, err := common.Queries.GetReviewerByDiscordID(context.Background(), discordUserID)
	if err != nil || reviewer.ID != job.ReviewerID {
		common.RespondError(s, i, "This review isn't assigned to you.")
		return
	}

	if err := common.Queries.SkipReviewJob(context.Background(), job.ID); err != nil {
		log.Printf("Failed to skip review job %d: %v", job.ID, err)
		common.RespondError(s, i, "Failed to skip review.")
		return
	}

	common.RespondEmbed(s, i, "Review Skipped", fmt.Sprintf("Skipped review for **%s** (ID: %d) — it'll go back to the pool.", doc.Title, doc.ID), common.ColorInfo)
}

func handlePagesMine(s *discordgo.Session, i *discordgo.InteractionCreate) {
	discordUserID := i.Member.User.ID
	reviewer, err := common.Queries.GetReviewerByDiscordID(context.Background(), discordUserID)
	if err != nil {
		common.RespondEmbed(s, i, "My Reviews", "You are not registered as a reviewer.", common.ColorInfo)
		return
	}

	jobs, err := common.Queries.ListPendingJobsByReviewer(context.Background(), reviewer.ID)
	if err != nil {
		log.Printf("Failed to list jobs for reviewer %d: %v", reviewer.ID, err)
		common.RespondError(s, i, "Failed to get your reviews.")
		return
	}

	if len(jobs) == 0 {
		common.RespondEmbed(s, i, "My Reviews", "You have no pending reviews.", common.ColorInfo)
		return
	}

	var lines []string
	for _, job := range jobs {
		doc, err := common.Queries.GetDocument(context.Background(), job.DocumentID)
		if err != nil {
			continue
		}
		daysLeft := 0
		if job.ExpiresAt.Valid {
			daysLeft = max(int(time.Until(job.ExpiresAt.Time).Hours()/24), 0)
		}
		lines = append(lines, fmt.Sprintf("**[%d]** [%s](%s) — %d day(s) left", doc.ID, doc.Title, doc.Url, daysLeft))
	}

	common.RespondEmbed(s, i, fmt.Sprintf("My Reviews (%d)", len(jobs)), strings.Join(lines, "\n"), common.ColorInfo)
}

func handlePagesInfo(s *discordgo.Session, i *discordgo.InteractionCreate, sub *discordgo.ApplicationCommandInteractionDataOption) {
	id := sub.Options[0].IntValue()

	doc, err := common.Queries.GetDocument(context.Background(), id)
	if err != nil {
		common.RespondError(s, i, fmt.Sprintf("No page found with ID `%d`.", id))
		return
	}

	lastReviewed := "Never"
	if doc.LastReviewed.Valid {
		lastReviewed = doc.LastReviewed.Time.Format("2006-01-02")
	}
	nextReview := "Not scheduled"
	if doc.NextReview.Valid {
		nextReview = doc.NextReview.Time.Format("2006-01-02")
	}

	assignee := "Unassigned"
	job, err := common.Queries.GetPendingJobForDocument(context.Background(), id)
	if err == nil {
		reviewer, rErr := common.Queries.GetReviewer(context.Background(), job.ReviewerID)
		if rErr == nil {
			daysLeft := 0
			if job.ExpiresAt.Valid {
				daysLeft = max(int(time.Until(job.ExpiresAt.Time).Hours()/24), 0)
			}
			assignee = fmt.Sprintf("<@%s> (%d day(s) left)", reviewer.DiscordUserID, daysLeft)
		}
	}

	status := "active"
	if !doc.Active {
		status = "paused"
	}

	desc := fmt.Sprintf("[%s](%s)\n\n**Status:** %s\n**Interval:** %dd\n**Reviews:** %d\n**Last reviewed:** %s\n**Next review:** %s\n**Assignee:** %s",
		doc.Title, doc.Url, status, doc.IntervalDays, doc.ReviewCount, lastReviewed, nextReview, assignee)
	common.RespondEmbed(s, i, fmt.Sprintf("Page Info — ID %d", doc.ID), desc, common.ColorInfo)
}

func handlePagesPending(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !common.RequireAuthorized(s, i) {
		return
	}
	jobs, err := common.Queries.ListPendingJobs(context.Background())
	if err != nil {
		log.Printf("Failed to list pending jobs: %v", err)
		common.RespondError(s, i, "Failed to get pending reviews.")
		return
	}

	if len(jobs) == 0 {
		common.RespondEmbed(s, i, "Pending Reviews", "No pending reviews.", common.ColorInfo)
		return
	}

	var lines []string
	for _, job := range jobs {
		doc, dErr := common.Queries.GetDocument(context.Background(), job.DocumentID)
		reviewer, rErr := common.Queries.GetReviewer(context.Background(), job.ReviewerID)
		if dErr != nil || rErr != nil {
			continue
		}
		lines = append(lines, fmt.Sprintf("**[%d]** [%s](%s) → **%s**", doc.ID, doc.Title, doc.Url, reviewer.Username))
	}

	common.RespondEmbed(s, i, fmt.Sprintf("Pending Reviews (%d)", len(jobs)), strings.Join(lines, "\n"), common.ColorInfo)
}

func handlePagesPause(s *discordgo.Session, i *discordgo.InteractionCreate, sub *discordgo.ApplicationCommandInteractionDataOption) {
	if !common.RequireAuthorized(s, i) {
		return
	}
	id := sub.Options[0].IntValue()

	doc, err := common.Queries.GetDocument(context.Background(), id)
	if err != nil {
		common.RespondError(s, i, fmt.Sprintf("No page found with ID `%d`.", id))
		return
	}
	if !doc.Active {
		common.RespondError(s, i, fmt.Sprintf("**%s** is already paused.", doc.Title))
		return
	}

	job, err := common.Queries.GetPendingJobForDocument(context.Background(), id)
	if err == nil {
		if assignedReviewer, rErr := common.Queries.GetReviewer(context.Background(), job.ReviewerID); rErr == nil {
			if dmCh, dmErr := s.UserChannelCreate(assignedReviewer.DiscordUserID); dmErr == nil {
				embed := &discordgo.MessageEmbed{
					Title:       "Review Cancelled",
					Description: fmt.Sprintf("Your review for **%s** has been cancelled — the page was paused.", doc.Title),
					Color:       common.ColorInfo,
				}
				if _, err := s.ChannelMessageSendEmbed(dmCh.ID, embed); err != nil {
					log.Printf("Failed to DM reviewer %s about cancelled review: %v", assignedReviewer.Username, err)
				}
			}
		}
		if err := common.Queries.CancelReviewJob(context.Background(), job.ID); err != nil {
			log.Printf("Failed to cancel job %d while pausing doc: %v", job.ID, err)
		}
	}

	if err := common.Queries.DeactivateDocument(context.Background(), id); err != nil {
		log.Printf("Failed to deactivate document %d: %v", id, err)
		common.RespondError(s, i, "Failed to pause page.")
		return
	}

	common.RespondEmbed(s, i, "Page Paused", fmt.Sprintf("**%s** (ID: %d) is now paused — it won't be assigned for review.", doc.Title, doc.ID), common.ColorInfo)
}

func handlePagesResume(s *discordgo.Session, i *discordgo.InteractionCreate, sub *discordgo.ApplicationCommandInteractionDataOption) {
	if !common.RequireAuthorized(s, i) {
		return
	}
	id := sub.Options[0].IntValue()

	doc, err := common.Queries.GetDocument(context.Background(), id)
	if err != nil {
		common.RespondError(s, i, fmt.Sprintf("No page found with ID `%d`.", id))
		return
	}
	if doc.Active {
		common.RespondError(s, i, fmt.Sprintf("**%s** is already active.", doc.Title))
		return
	}

	if err := common.Queries.ActivateDocument(context.Background(), id); err != nil {
		log.Printf("Failed to activate document %d: %v", id, err)
		common.RespondError(s, i, "Failed to resume page.")
		return
	}

	common.RespondEmbed(s, i, "Page Resumed", fmt.Sprintf("**%s** (ID: %d) is now active — it'll be assigned on the next review cycle.", doc.Title, doc.ID), common.ColorSuccess)
}

func handlePagesKOList(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !common.RequireAuthorized(s, i) {
		return
	}
	jobs, err := common.Queries.ListKOJobs(context.Background())
	if err != nil {
		log.Printf("Failed to list KO jobs: %v", err)
		common.RespondError(s, i, "Failed to get KO list.")
		return
	}

	if len(jobs) == 0 {
		common.RespondEmbed(s, i, "KO Pages", "No pages flagged KO.", common.ColorSuccess)
		return
	}

	var lines []string
	for _, job := range jobs {
		doc, err := common.Queries.GetDocument(context.Background(), job.DocumentID)
		if err != nil {
			continue
		}
		notes := ""
		if job.Notes.Valid {
			notes = job.Notes.String
		}
		lines = append(lines, fmt.Sprintf("**[%d]** [%s](%s)\n> %s", doc.ID, doc.Title, doc.Url, notes))
	}

	common.RespondEmbed(s, i, fmt.Sprintf("KO Pages (%d)", len(jobs)), strings.Join(lines, "\n\n"), common.ColorError)
}

func handlePagesAssign(s *discordgo.Session, i *discordgo.InteractionCreate, sub *discordgo.ApplicationCommandInteractionDataOption) {
	if !common.RequireAuthorized(s, i) {
		return
	}
	id := sub.Options[0].IntValue()
	user := sub.Options[1].UserValue(s)

	doc, err := common.Queries.GetDocument(context.Background(), id)
	if err != nil {
		common.RespondError(s, i, fmt.Sprintf("No page found with ID `%d`.", id))
		return
	}

	reviewer, err := common.Queries.GetReviewerByDiscordID(context.Background(), user.ID)
	if err != nil {
		common.RespondError(s, i, fmt.Sprintf("**%s** is not a registered reviewer.", user.Username))
		return
	}
	if !reviewer.Active {
		common.RespondError(s, i, fmt.Sprintf("**%s** is currently paused.", reviewer.Username))
		return
	}

	existingJob, err := common.Queries.GetPendingJobForDocument(context.Background(), id)
	if err == nil {
		if err := common.Queries.CancelReviewJob(context.Background(), existingJob.ID); err != nil {
			log.Printf("Failed to cancel existing job %d for assign: %v", existingJob.ID, err)
		}
	}

	expiresAt := sql.NullTime{Time: time.Now().Add(3 * 24 * time.Hour), Valid: true}
	if err := common.Queries.CreateReviewJob(context.Background(), db.CreateReviewJobParams{
		DocumentID: doc.ID,
		ReviewerID: reviewer.ID,
		ExpiresAt:  expiresAt,
	}); err != nil {
		log.Printf("Failed to create review job for assign: %v", err)
		common.RespondError(s, i, "Failed to assign page.")
		return
	}

	if err := common.Queries.UpdateReviewerAssigned(context.Background(), reviewer.ID); err != nil {
		log.Printf("Failed to update reviewer assigned: %v", err)
	}

	dmCh, err := s.UserChannelCreate(reviewer.DiscordUserID)
	if err == nil {
		embed := &discordgo.MessageEmbed{
			Title:       "New Review Assignment",
			Description: fmt.Sprintf("You've been manually assigned to review [%s](%s).\nPlease review it within 3 days.", doc.Title, doc.Url),
			Color:       common.ColorInfo,
		}
		if _, err := s.ChannelMessageSendEmbed(dmCh.ID, embed); err != nil {
			log.Printf("Failed to DM reviewer %s for assign: %v", reviewer.Username, err)
		}
	}

	common.RespondEmbed(s, i, "Page Assigned", fmt.Sprintf("**%s** (ID: %d) assigned to **%s**.", doc.Title, doc.ID, reviewer.Username), common.ColorSuccess)
}

func handlePagesFixed(s *discordgo.Session, i *discordgo.InteractionCreate, sub *discordgo.ApplicationCommandInteractionDataOption) {
	if !common.RequireAuthorized(s, i) {
		return
	}
	id := sub.Options[0].IntValue()

	doc, err := common.Queries.GetDocument(context.Background(), id)
	if err != nil {
		common.RespondError(s, i, fmt.Sprintf("No page found with ID `%d`.", id))
		return
	}

	if err := common.Queries.ResetDocumentSchedule(context.Background(), id); err != nil {
		log.Printf("Failed to reset document schedule %d: %v", id, err)
		common.RespondError(s, i, "Failed to mark page as fixed.")
		return
	}

	common.RespondEmbed(s, i, "Page Fixed", fmt.Sprintf("**%s** (ID: %d) marked as fixed — next review in %dd", doc.Title, doc.ID, doc.IntervalDays), common.ColorSuccess)

	// Post to #documentation channel
	if common.ChannelID != "" {
		embed := &discordgo.MessageEmbed{
			Title:       "Page Fixed",
			Description: fmt.Sprintf("[%s](%s) has been fixed", doc.Title, doc.Url),
			Color:       common.ColorSuccess,
		}
		if _, err := s.ChannelMessageSendEmbed(common.ChannelID, embed); err != nil {
			log.Printf("Failed to post fixed to channel: %v", err)
		}
	}

}

type yamlDocument struct {
	Title    string `yaml:"title"`
	URL      string `yaml:"url"`
	Interval int64  `yaml:"interval_days"`
}

type yamlDocuments struct {
	Documents []yamlDocument `yaml:"documents"`
}

func handlePagesImport(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !common.RequireAuthorized(s, i) {
		return
	}
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: modalImportPageID,
			Title:    "Import Pages from YAML",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "yaml_url",
							Label:       "YAML File URL",
							Style:       discordgo.TextInputShort,
							Placeholder: "e.g. https://raw.githubusercontent.com/.../docs.yaml",
							Required:    true,
							MinLength:   1,
							MaxLength:   500,
						},
					},
				},
			},
		},
	})
	if err != nil {
		log.Printf("Failed to show import modal: %v", err)
	}
}

func handleImportPageSubmit(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ModalSubmitInteractionData) {
	if !common.RequireAuthorized(s, i) {
		return
	}
	var yamlURL string

	for _, row := range data.Components {
		ar, ok := row.(*discordgo.ActionsRow)
		if !ok {
			continue
		}
		for _, comp := range ar.Components {
			input, ok := comp.(*discordgo.TextInput)
			if !ok {
				continue
			}
			if input.CustomID == "yaml_url" {
				yamlURL = strings.TrimSpace(input.Value)
			}
		}
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(yamlURL)
	if err != nil {
		common.EditDeferredError(s, i, fmt.Sprintf("Failed to fetch YAML file: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		common.EditDeferredError(s, i, fmt.Sprintf("Failed to fetch YAML file: HTTP %d", resp.StatusCode))
		return
	}

	var docs yamlDocuments
	decoder := yaml.NewDecoder(resp.Body)
	if err := decoder.Decode(&docs); err != nil {
		common.EditDeferredError(s, i, fmt.Sprintf("Invalid YAML format: %v", err))
		return
	}

	if len(docs.Documents) == 0 {
		common.EditDeferredError(s, i, "No documents found in YAML file. Expected a `documents` list with `title`, `url`, and `interval_days` fields.")
		return
	}

	tx, err := common.DBConn.BeginTx(context.Background(), nil)
	if err != nil {
		common.EditDeferredError(s, i, fmt.Sprintf("Failed to start transaction: %v", err))
		return
	}
	defer tx.Rollback()

	txQueries := common.Queries.WithTx(tx)

	var added, skipped int

	for _, doc := range docs.Documents {
		if doc.Title == "" || doc.URL == "" || doc.Interval <= 0 {
			skipped++
			continue
		}

		err := txQueries.CreateDocument(context.Background(), db.CreateDocumentParams{
			Url:          doc.URL,
			Title:        doc.Title,
			IntervalDays: doc.Interval,
		})
		if err != nil {
			skipped++
			if !strings.Contains(err.Error(), "UNIQUE constraint failed") {
				log.Printf("Failed to import document %s: %v", doc.Title, err)
			}
			continue
		}

		added++
	}

	if err := tx.Commit(); err != nil {
		common.EditDeferredError(s, i, fmt.Sprintf("Failed to commit import: %v", err))
		return
	}

	link := fmt.Sprintf("%s/pages", common.BaseURL)
	embeds := []*discordgo.MessageEmbed{
		{
			Title:       "Import Complete",
			Description: fmt.Sprintf("Added **%d**, skipped **%d**\n\n[View all pages](%s)", added, skipped, link),
			Color:       common.ColorSuccess,
		},
	}
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Embeds: &embeds})
	if err != nil {
		log.Printf("Failed to edit deferred response: %v", err)
	}
}
