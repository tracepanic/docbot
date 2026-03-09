-- name: GetReviewJob :one
SELECT * FROM review_jobs WHERE id = ?;

-- name: GetPendingJobForDocument :one
SELECT * FROM review_jobs
WHERE document_id = ? AND status = 'pending'
LIMIT 1;

-- name: ListPendingJobs :many
SELECT * FROM review_jobs WHERE status = 'pending';

-- name: ListJobsByReviewer :many
SELECT * FROM review_jobs WHERE reviewer_id = ?
ORDER BY assigned_at DESC;

-- name: CreateReviewJob :exec
INSERT INTO review_jobs (document_id, reviewer_id, expires_at)
VALUES (?, ?, ?);

-- name: CompleteReviewJob :exec
UPDATE review_jobs
SET status = 'completed', completed_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: ExpireReviewJob :exec
UPDATE review_jobs SET status = 'expired' WHERE id = ?;

-- name: SkipReviewJob :exec
UPDATE review_jobs SET status = 'skipped' WHERE id = ?;

-- name: SetJobMessageID :exec
UPDATE review_jobs SET message_id = ? WHERE id = ?;

-- name: DeleteReviewJobsByDocument :exec
DELETE FROM review_jobs WHERE document_id = ?;

-- name: CompleteReviewJobKO :exec
UPDATE review_jobs SET status = 'ko', notes = ?, completed_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: ListKOJobs :many
SELECT * FROM review_jobs WHERE status = 'ko';

-- name: ListPendingJobsByReviewer :many
SELECT * FROM review_jobs WHERE reviewer_id = ? AND status = 'pending'
ORDER BY assigned_at ASC;

-- name: CancelReviewJob :exec
UPDATE review_jobs SET status = 'cancelled' WHERE id = ?;

-- name: GetLatestKOJobForDocument :one
SELECT * FROM review_jobs
WHERE document_id = ? AND status = 'ko'
ORDER BY completed_at DESC
LIMIT 1;
