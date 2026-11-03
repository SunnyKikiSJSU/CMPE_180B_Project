BEGIN;

-- Drop indexes before their tables. Table drops are ordered to respect foreign
-- keys even when PostgreSQL does not need the explicit order.
DROP INDEX IF EXISTS reviews_book_created_at_idx;
DROP INDEX IF EXISTS ratings_book_id_idx;
DROP INDEX IF EXISTS reading_history_book_created_at_idx;
DROP INDEX IF EXISTS reading_history_user_created_at_idx;
DROP INDEX IF EXISTS reading_progress_book_id_idx;
DROP INDEX IF EXISTS user_shelf_items_book_id_idx;
DROP INDEX IF EXISTS shelves_user_id_idx;
DROP INDEX IF EXISTS books_published_date_idx;
DROP INDEX IF EXISTS books_title_lower_idx;
DROP INDEX IF EXISTS book_categories_category_id_idx;
DROP INDEX IF EXISTS book_authors_author_id_idx;

DROP TABLE IF EXISTS book_rating_summaries;
DROP TABLE IF EXISTS reviews;
DROP TABLE IF EXISTS ratings;
DROP TABLE IF EXISTS reading_history;
DROP TABLE IF EXISTS reading_progress;
DROP TABLE IF EXISTS user_shelf_items;
DROP TABLE IF EXISTS shelves;
DROP TABLE IF EXISTS book_categories;
DROP TABLE IF EXISTS book_authors;
DROP TABLE IF EXISTS books;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS authors;
DROP TABLE IF EXISTS user_profiles;
DROP TABLE IF EXISTS users;

COMMIT;
