package db

// Schema is the DDL for the mailbeads database.
const Schema = `
CREATE TABLE IF NOT EXISTS emails (
    id          TEXT PRIMARY KEY,
    account     TEXT NOT NULL,
    thread_id   TEXT NOT NULL,
    message_id  TEXT,
    from_addr   TEXT NOT NULL,
    to_addr     TEXT,
    cc          TEXT,
    subject     TEXT NOT NULL,
    snippet     TEXT,
    body        TEXT,
    date        TEXT NOT NULL,
    labels      TEXT,
    is_read     INTEGER DEFAULT 0,
    fetched_at  TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS triage (
    id              TEXT PRIMARY KEY,
    thread_id       TEXT NOT NULL,
    account         TEXT NOT NULL,
    subject         TEXT NOT NULL,
    from_addr       TEXT,
    priority        TEXT NOT NULL DEFAULT 'medium',
    action          TEXT NOT NULL,
    suggestion      TEXT,
    agent_notes     TEXT,
    category        TEXT,
    status          TEXT NOT NULL DEFAULT 'pending',
    snoozed_until   TEXT,
    email_count     INTEGER DEFAULT 1,
    latest_date     TEXT,
    created_at      TEXT NOT NULL,
    updated_at      TEXT,
    UNIQUE(thread_id, account)
);

CREATE TABLE IF NOT EXISTS triage_deps (
    triage_id       TEXT NOT NULL,
    depends_on_id   TEXT NOT NULL,
    created_at      TEXT NOT NULL,
    PRIMARY KEY (triage_id, depends_on_id),
    FOREIGN KEY (triage_id) REFERENCES triage(id),
    FOREIGN KEY (depends_on_id) REFERENCES triage(id)
);

CREATE INDEX IF NOT EXISTS idx_emails_account ON emails(account);
CREATE INDEX IF NOT EXISTS idx_emails_thread ON emails(thread_id);
CREATE INDEX IF NOT EXISTS idx_emails_date ON emails(date DESC);
CREATE INDEX IF NOT EXISTS idx_triage_status ON triage(status);
CREATE INDEX IF NOT EXISTS idx_triage_priority ON triage(priority);
CREATE INDEX IF NOT EXISTS idx_triage_thread ON triage(thread_id, account);
`
