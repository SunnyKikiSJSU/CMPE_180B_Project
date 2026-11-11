BEGIN;

-- Drop dependent indexes before the column and extensions. Extensions are
-- dropped only after every object that uses them is gone.
DROP INDEX IF EXISTS categories_name_trgm_idx;
DROP INDEX IF EXISTS authors_name_trgm_idx;
DROP INDEX IF EXISTS books_title_trgm_idx;
DROP INDEX IF EXISTS books_search_vector_gin_idx;

ALTER TABLE books
    DROP COLUMN IF EXISTS search_vector;

DROP EXTENSION IF EXISTS pg_trgm;
DROP EXTENSION IF EXISTS pgcrypto;

COMMIT;
