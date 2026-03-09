-- name: GetDocument :one
SELECT * FROM documents WHERE id = ?;

-- name: GetDocumentByURL :one
SELECT * FROM documents WHERE url = ?;

-- name: ListAllDocuments :many
SELECT * FROM documents;

-- name: ListActiveDocuments :many
SELECT * FROM documents WHERE active = 1;

-- name: ListDueDocuments :many
SELECT * FROM documents d
WHERE d.active = 1
  AND d.next_review <= CURRENT_TIMESTAMP
  AND NOT EXISTS (
    SELECT 1 FROM review_jobs rj
    WHERE rj.document_id = d.id AND rj.status = 'pending'
  );

-- name: CreateDocument :exec
INSERT INTO documents (url, title, interval_days, next_review)
VALUES (?, ?, ?, CURRENT_TIMESTAMP);

-- name: UpdateDocumentReview :exec
UPDATE documents
SET last_reviewed = CURRENT_TIMESTAMP,
  next_review = DATETIME(CURRENT_TIMESTAMP, '+' || interval_days || ' days'),
  review_count = review_count + 1
WHERE id = ?;

-- name: DeactivateDocument :exec
UPDATE documents SET active = 0 WHERE id = ?;

-- name: ActivateDocument :exec
UPDATE documents SET active = 1 WHERE id = ?;

-- name: DeleteDocument :exec
DELETE FROM documents WHERE id = ?;

-- name: IncrementDocumentReviewCount :exec
UPDATE documents SET review_count = review_count + 1 WHERE id = ?;

-- name: ResetDocumentSchedule :exec
UPDATE documents
SET last_reviewed = CURRENT_TIMESTAMP,
  next_review = DATETIME(CURRENT_TIMESTAMP, '+' || interval_days || ' days')
WHERE id = ?;
