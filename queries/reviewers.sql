-- name: GetReviewer :one
SELECT * FROM reviewers WHERE id = ?;

-- name: GetReviewerByDiscordID :one
SELECT * FROM reviewers WHERE discord_user_id = ?;

-- name: ListActiveReviewers :many
SELECT * FROM reviewers WHERE active = 1;

-- name: ListAllReviewers :many
SELECT * FROM reviewers;

-- name: GetLeastRecentReviewer :one
SELECT * FROM reviewers
WHERE active = 1
  AND NOT EXISTS (
    SELECT 1 FROM review_jobs rj
    WHERE rj.reviewer_id = reviewers.id AND rj.status = 'pending'
  )
ORDER BY last_assigned ASC NULLS FIRST
LIMIT 1;

-- name: CreateReviewer :exec
INSERT INTO reviewers (discord_user_id, username)
VALUES (?, ?);

-- name: UpdateReviewerAssigned :exec
UPDATE reviewers SET last_assigned = CURRENT_TIMESTAMP WHERE id = ?;

-- name: DeactivateReviewer :exec
UPDATE reviewers SET active = 0 WHERE id = ?;

-- name: ActivateReviewer :exec
UPDATE reviewers SET active = 1 WHERE id = ?;

-- name: DeleteReviewer :exec
DELETE FROM reviewers WHERE id = ?;
