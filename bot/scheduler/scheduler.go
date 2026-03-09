package scheduler

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/PlakarKorp/docbot/bot/common"
	"github.com/PlakarKorp/docbot/bot/db"
	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"
)

func Start(s *discordgo.Session) {
	cet := time.FixedZone("CET", 1*60*60)
	c := cron.New(cron.WithLocation(cet))

	_, err := c.AddFunc("0 9 * * 1-5", func() {
		log.Println("Scheduler: running daily review cycle")
		expireStaleJobs()
		assignDueDocuments(s)
		remindPendingReviewers(s)
		remindMaintainerKO(s)
	})
	if err != nil {
		log.Fatalf("Failed to register cron job: %v", err)
	}

	c.Start()
	log.Println("Scheduler started (9 AM UTC+1, Mon-Fri)")
}

func assignDueDocuments(s *discordgo.Session) {
	docs, err := common.Queries.ListDueDocuments(context.Background())
	if err != nil {
		log.Printf("Scheduler: failed to list due documents: %v", err)
		return
	}

	for _, doc := range docs {
		reviewer, err := common.Queries.GetLeastRecentReviewer(context.Background())
		if err != nil {
			log.Printf("Scheduler: no active reviewers available: %v", err)
			return
		}

		expiresAt := sql.NullTime{Time: time.Now().Add(3 * 24 * time.Hour), Valid: true}

		err = common.Queries.CreateReviewJob(context.Background(), db.CreateReviewJobParams{
			DocumentID: doc.ID,
			ReviewerID: reviewer.ID,
			ExpiresAt:  expiresAt,
		})
		if err != nil {
			log.Printf("Scheduler: failed to create review job for doc %d: %v", doc.ID, err)
			continue
		}

		if err := common.Queries.UpdateReviewerAssigned(context.Background(), reviewer.ID); err != nil {
			log.Printf("Scheduler: failed to update reviewer assigned: %v", err)
		}

		// Post to #documentation channel
		if common.ChannelID != "" {
			embed := &discordgo.MessageEmbed{
				Title:       "New Review Assignment",
				Description: fmt.Sprintf("[%s](%s) assigned to <@%s>", doc.Title, doc.Url, reviewer.DiscordUserID),
				Color:       common.ColorInfo,
			}
			if _, err := s.ChannelMessageSendEmbed(common.ChannelID, embed); err != nil {
				log.Printf("Scheduler: failed to post assignment to channel: %v", err)
			}
		}

		// DM the reviewer
		dmChannel, err := s.UserChannelCreate(reviewer.DiscordUserID)
		if err != nil {
			log.Printf("Scheduler: failed to create DM channel for reviewer %s: %v", reviewer.Username, err)
			continue
		}
		embed := &discordgo.MessageEmbed{
			Title:       "New Review Assignment",
			Description: fmt.Sprintf("You've been assigned to review [%s](%s).\nPlease review it within 3 days.", doc.Title, doc.Url),
			Color:       common.ColorInfo,
		}
		if _, err := s.ChannelMessageSendEmbed(dmChannel.ID, embed); err != nil {
			log.Printf("Scheduler: failed to DM reviewer %s: %v", reviewer.Username, err)
		}
	}
}

func remindPendingReviewers(s *discordgo.Session) {
	jobs, err := common.Queries.ListPendingJobs(context.Background())
	if err != nil {
		log.Printf("Scheduler: failed to list pending jobs: %v", err)
		return
	}

	for _, job := range jobs {
		// Don't remind if just assigned today
		if time.Since(job.AssignedAt) < 24*time.Hour {
			continue
		}

		reviewer, err := common.Queries.GetReviewer(context.Background(), job.ReviewerID)
		if err != nil {
			log.Printf("Scheduler: failed to get reviewer %d: %v", job.ReviewerID, err)
			continue
		}

		doc, err := common.Queries.GetDocument(context.Background(), job.DocumentID)
		if err != nil {
			log.Printf("Scheduler: failed to get document %d: %v", job.DocumentID, err)
			continue
		}

		dmChannel, err := s.UserChannelCreate(reviewer.DiscordUserID)
		if err != nil {
			log.Printf("Scheduler: failed to create DM channel for %s: %v", reviewer.Username, err)
			continue
		}

		daysLeft := max(3-int(time.Since(job.AssignedAt).Hours()/24), 1)

		embed := &discordgo.MessageEmbed{
			Title:       "Pending Review Reminder",
			Description: fmt.Sprintf("You have a pending review for [%s](%s).\n%d day(s) remaining before it expires.", doc.Title, doc.Url, daysLeft),
			Color:       common.ColorInfo,
		}
		if _, err := s.ChannelMessageSendEmbed(dmChannel.ID, embed); err != nil {
			log.Printf("Scheduler: failed to DM reviewer %s: %v", reviewer.Username, err)
		}
	}
}

func expireStaleJobs() {
	jobs, err := common.Queries.ListPendingJobs(context.Background())
	if err != nil {
		log.Printf("Scheduler: failed to list pending jobs for expiry: %v", err)
		return
	}

	for _, job := range jobs {
		if job.ExpiresAt.Valid && time.Now().After(job.ExpiresAt.Time) {
			if err := common.Queries.ExpireReviewJob(context.Background(), job.ID); err != nil {
				log.Printf("Scheduler: failed to expire job %d: %v", job.ID, err)
			} else {
				log.Printf("Scheduler: expired job %d (doc %d, reviewer %d)", job.ID, job.DocumentID, job.ReviewerID)
			}
		}
	}
}

func remindMaintainerKO(s *discordgo.Session) {
	if common.MaintainerDiscordID == "" {
		return
	}

	jobs, err := common.Queries.ListKOJobs(context.Background())
	if err != nil {
		log.Printf("Scheduler: failed to list KO jobs: %v", err)
		return
	}

	if len(jobs) == 0 {
		return
	}

	dmChannel, err := s.UserChannelCreate(common.MaintainerDiscordID)
	if err != nil {
		log.Printf("Scheduler: failed to create DM channel for maintainer: %v", err)
		return
	}

	for _, job := range jobs {
		doc, err := common.Queries.GetDocument(context.Background(), job.DocumentID)
		if err != nil {
			log.Printf("Scheduler: failed to get document %d: %v", job.DocumentID, err)
			continue
		}

		notes := ""
		if job.Notes.Valid {
			notes = job.Notes.String
		}

		embed := &discordgo.MessageEmbed{
			Title:       "KO Doc Needs Fixing",
			Description: fmt.Sprintf("[%s](%s) was flagged KO.\n\n**Feedback:** %s\n\nUse `/doc pages fixed id:%d` when done.", doc.Title, doc.Url, notes, doc.ID),
			Color:       common.ColorError,
		}
		if _, err := s.ChannelMessageSendEmbed(dmChannel.ID, embed); err != nil {
			log.Printf("Scheduler: failed to DM maintainer about doc %d: %v", doc.ID, err)
		}
	}
}
