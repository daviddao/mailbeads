package db

// Schema is the DDL for the mailbeads database.
//
// The triage table is a thin cross-reference mapping email threads to beads
// issues. All triage state (priority, status, action, dependencies) lives in
// the beads (.beads/) database — mailbeads only stores the mapping so that
// mb untriaged / mb show can efficiently look up whether a thread has been
// triaged without querying beads.
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
    thread_id   TEXT NOT NULL,
    account     TEXT NOT NULL,
    bead_id     TEXT NOT NULL,
    created_at  TEXT NOT NULL,
    UNIQUE(thread_id, account)
);

CREATE INDEX IF NOT EXISTS idx_emails_account ON emails(account);
CREATE INDEX IF NOT EXISTS idx_emails_thread ON emails(thread_id);
CREATE INDEX IF NOT EXISTS idx_emails_date ON emails(date DESC);
CREATE INDEX IF NOT EXISTS idx_triage_thread ON triage(thread_id, account);
CREATE INDEX IF NOT EXISTS idx_triage_bead ON triage(bead_id);
`

// MigrationV2 migrates from the old fat triage table to the new slim one.
// It preserves the thread_id/account/created_at and maps the old triage ID
// as a placeholder bead_id (prefixed with "legacy-") until migration runs.
const MigrationV2 = `
-- Check if old schema exists (has 'action' column)
-- If so, rename old table and create new one
ALTER TABLE triage RENAME TO triage_old;

CREATE TABLE IF NOT EXISTS triage (
    thread_id   TEXT NOT NULL,
    account     TEXT NOT NULL,
    bead_id     TEXT NOT NULL,
    created_at  TEXT NOT NULL,
    UNIQUE(thread_id, account)
);

CREATE INDEX IF NOT EXISTS idx_triage_thread ON triage(thread_id, account);
CREATE INDEX IF NOT EXISTS idx_triage_bead ON triage(bead_id);

-- Copy pending entries with legacy prefix so they can be migrated
INSERT INTO triage (thread_id, account, bead_id, created_at)
    SELECT thread_id, account, 'legacy-' || id, created_at
    FROM triage_old
    WHERE status = 'pending';

DROP TABLE IF EXISTS triage_old;
DROP TABLE IF EXISTS triage_deps;
`
