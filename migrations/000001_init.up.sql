CREATE TABLE documents (
  id              INTEGER PRIMARY KEY,
  url             TEXT NOT NULL UNIQUE,
  title           TEXT NOT NULL,
  interval_days   INTEGER NOT NULL DEFAULT 30,
  active          BOOLEAN NOT NULL DEFAULT 1,
  last_reviewed   DATETIME,
  next_review     DATETIME,
  review_count    INTEGER NOT NULL DEFAULT 0,
  created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE reviewers (
  id              INTEGER PRIMARY KEY,
  discord_user_id TEXT NOT NULL UNIQUE,
  username        TEXT NOT NULL,
  active          BOOLEAN NOT NULL DEFAULT 1,
  last_assigned   DATETIME,
  created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE review_jobs (
  id              INTEGER PRIMARY KEY,
  document_id     INTEGER NOT NULL,
  reviewer_id     INTEGER NOT NULL,
  status          TEXT NOT NULL DEFAULT 'pending',
  assigned_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  completed_at    DATETIME,
  expires_at      DATETIME,
  message_id      TEXT,
  notes           TEXT,
  FOREIGN KEY(document_id) REFERENCES documents(id),
  FOREIGN KEY(reviewer_id) REFERENCES reviewers(id)
);
