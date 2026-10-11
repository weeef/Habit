-- Applied on every start, so every statement must be safe to re-run.

CREATE TABLE IF NOT EXISTS accounts (
    id            text PRIMARY KEY,
    username      text NOT NULL,
    birth_year    integer NOT NULL,
    created_at    timestamptz NOT NULL,
    salt          bytea NOT NULL,
    password_hash bytea NOT NULL,
    recovery_hash bytea NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS accounts_username_lower ON accounts (lower(username));
-- Character style, chosen at signup (indexes into the client's palettes).
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS style_skin      smallint NOT NULL DEFAULT 0;
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS style_hair      smallint NOT NULL DEFAULT 0;
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS style_outfit    smallint NOT NULL DEFAULT 0;
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS style_accessory smallint NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS sessions (
    token_hash bytea PRIMARY KEY,
    account_id text NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL
);
CREATE INDEX IF NOT EXISTS sessions_account ON sessions (account_id);

CREATE TABLE IF NOT EXISTS messages (
    id         bigserial PRIMARY KEY,
    room       text NOT NULL,
    account_id text REFERENCES accounts (id) ON DELETE SET NULL,
    avatar_id  text NOT NULL,
    name       text NOT NULL,
    body       text NOT NULL,
    newbie     boolean NOT NULL,
    sent_at    timestamptz NOT NULL
);
CREATE INDEX IF NOT EXISTS messages_room_recent ON messages (room, id DESC);

-- Board posts are kept forever. name is the author's username if they chose
-- to show it, '' for anonymous.
CREATE TABLE IF NOT EXISTS threads (
    id          bigserial PRIMARY KEY,
    room        text NOT NULL,
    account_id  text REFERENCES accounts (id) ON DELETE SET NULL,
    name        text NOT NULL,
    title       text NOT NULL,
    body        text NOT NULL,
    created_at  timestamptz NOT NULL,
    bumped_at   timestamptz NOT NULL,
    reply_count integer NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS threads_room_bumped ON threads (room, bumped_at DESC, id DESC);
ALTER TABLE threads ADD COLUMN IF NOT EXISTS ups   integer NOT NULL DEFAULT 0;
ALTER TABLE threads ADD COLUMN IF NOT EXISTS downs integer NOT NULL DEFAULT 0;

-- One vote per account per thread: 1 up, -1 down.
CREATE TABLE IF NOT EXISTS thread_votes (
    thread_id  bigint NOT NULL REFERENCES threads (id) ON DELETE CASCADE,
    account_id text NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    value      smallint NOT NULL CHECK (value IN (-1, 1)),
    PRIMARY KEY (thread_id, account_id)
);

CREATE TABLE IF NOT EXISTS replies (
    id         bigserial PRIMARY KEY,
    thread_id  bigint NOT NULL REFERENCES threads (id) ON DELETE CASCADE,
    account_id text REFERENCES accounts (id) ON DELETE SET NULL,
    name       text NOT NULL,
    body       text NOT NULL,
    created_at timestamptz NOT NULL
);
CREATE INDEX IF NOT EXISTS replies_thread ON replies (thread_id, id);

CREATE TABLE IF NOT EXISTS notifications (
    id         bigserial PRIMARY KEY,
    account_id text NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    thread_id  bigint NOT NULL REFERENCES threads (id) ON DELETE CASCADE,
    reply_id   bigint NOT NULL REFERENCES replies (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL,
    read       boolean NOT NULL DEFAULT false
);
CREATE INDEX IF NOT EXISTS notifications_account ON notifications (account_id, id DESC);

CREATE TABLE IF NOT EXISTS settings (
    key   text PRIMARY KEY,
    value bytea NOT NULL
);
