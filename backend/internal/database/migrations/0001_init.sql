-- Enable pgcrypto for gen_random_uuid() on older Postgres installs.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ---------------------------------------------------------------------------
-- Dictionary: JMdict entries pre-parsed into Postgres with GIN indexes.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS jmdict_entries (
    ent_seq   INTEGER PRIMARY KEY,
    kanji     TEXT[]  DEFAULT '{}',
    readings  TEXT[]  DEFAULT '{}',
    meanings  JSONB   NOT NULL DEFAULT '[]'
);
CREATE INDEX IF NOT EXISTS idx_jmdict_kanji ON jmdict_entries USING GIN (kanji);
CREATE INDEX IF NOT EXISTS idx_jmdict_readings ON jmdict_entries USING GIN (readings);

-- ---------------------------------------------------------------------------
-- Single web articles produced by the browser extension.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS articles (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    url          TEXT UNIQUE NOT NULL,
    title        TEXT NOT NULL,
    content_html TEXT NOT NULL,
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_articles_url ON articles(url);

-- ---------------------------------------------------------------------------
-- EPUB books, chapters and offloaded images.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS books (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title      TEXT NOT NULL,
    author     TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS chapters (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    book_id       UUID REFERENCES books(id) ON DELETE CASCADE,
    chapter_index INTEGER NOT NULL,
    title         TEXT,
    content_html  TEXT NOT NULL,
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_chapters_book_id ON chapters(book_id);

CREATE TABLE IF NOT EXISTS epub_images (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    book_id       UUID REFERENCES books(id) ON DELETE CASCADE,
    original_path TEXT NOT NULL,
    mime_type     TEXT NOT NULL,
    s3_key        TEXT NOT NULL,
    s3_url        TEXT NOT NULL,
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_epub_images_book_path ON epub_images(book_id, original_path);
