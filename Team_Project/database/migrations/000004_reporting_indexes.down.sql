BEGIN;

DROP INDEX IF EXISTS ratings_user_updated_at_idx;
DROP INDEX IF EXISTS reviews_user_created_at_idx;
DROP INDEX IF EXISTS ratings_book_rating_value_idx;
DROP INDEX IF EXISTS reading_progress_user_updated_at_idx;
DROP INDEX IF EXISTS user_shelf_items_shelf_added_at_idx;

COMMIT;
