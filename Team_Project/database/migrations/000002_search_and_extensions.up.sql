BEGIN;

-- Search support. pg_trgm lets author/category/title filters match partial
-- words and small typos without a separate search service.
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- pgcrypto is used only by the demo seed to generate valid password hashes.
-- Application-created users are hashed with Argon2id in the Go auth package.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- A generated column keeps the search document in sync with every insert and
-- update. PostgreSQL 18 can build a stored tsvector from immutable functions.
ALTER TABLE books
    ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(description, '')), 'B')
    ) STORED;

-- GIN indexes make full-text and trigram searches index scans rather than
-- sequential scans as the catalog grows.
CREATE INDEX books_search_vector_gin_idx
    ON books USING GIN (search_vector);

CREATE INDEX books_title_trgm_idx
    ON books USING GIN (title gin_trgm_ops);

CREATE INDEX authors_name_trgm_idx
    ON authors USING GIN (name gin_trgm_ops);

CREATE INDEX categories_name_trgm_idx
    ON categories USING GIN (name gin_trgm_ops);

COMMIT;
