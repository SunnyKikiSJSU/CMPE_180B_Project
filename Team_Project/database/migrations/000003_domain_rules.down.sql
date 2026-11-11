BEGIN;

-- Triggers first: functions cannot be dropped while a trigger depends on them.
DROP TRIGGER IF EXISTS reading_history_immutable ON reading_history;
DROP TRIGGER IF EXISTS reading_progress_validate ON reading_progress;
DROP TRIGGER IF EXISTS reading_progress_set_updated_at ON reading_progress;
DROP TRIGGER IF EXISTS ratings_validate_value ON ratings;
DROP TRIGGER IF EXISTS ratings_refresh_summary ON ratings;
DROP TRIGGER IF EXISTS ratings_set_updated_at ON ratings;
DROP TRIGGER IF EXISTS reviews_refresh_summary ON reviews;
DROP TRIGGER IF EXISTS reviews_set_updated_at ON reviews;
DROP TRIGGER IF EXISTS books_create_summary ON books;
DROP TRIGGER IF EXISTS users_create_defaults ON users;
DROP TRIGGER IF EXISTS user_profiles_set_updated_at ON user_profiles;

-- Functions in dependency order.
DROP FUNCTION IF EXISTS fn_move_book_to_shelf(uuid, uuid, uuid);
DROP FUNCTION IF EXISTS fn_update_reading_progress(uuid, uuid, integer, text, integer);
DROP FUNCTION IF EXISTS fn_block_history_mutation();
DROP FUNCTION IF EXISTS fn_validate_reading_progress();
DROP FUNCTION IF EXISTS fn_books_after_insert();
DROP FUNCTION IF EXISTS fn_reviews_summary_trigger();
DROP FUNCTION IF EXISTS fn_ratings_summary_trigger();
DROP FUNCTION IF EXISTS fn_refresh_book_rating_summary(uuid);
DROP FUNCTION IF EXISTS fn_validate_rating();
DROP FUNCTION IF EXISTS fn_users_after_insert();
DROP FUNCTION IF EXISTS fn_create_user_defaults(uuid);
DROP FUNCTION IF EXISTS fn_set_updated_at();

DROP INDEX IF EXISTS shelves_one_per_status_type_idx;
DROP INDEX IF EXISTS categories_name_ci_unique_idx;
DROP INDEX IF EXISTS authors_name_ci_unique_idx;

ALTER TABLE reading_progress
    DROP COLUMN IF EXISTS version;

COMMIT;
