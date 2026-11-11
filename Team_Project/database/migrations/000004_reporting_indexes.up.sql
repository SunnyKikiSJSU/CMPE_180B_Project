BEGIN;

-- Shelf listings are ordered by recency and filtered by shelf.
CREATE INDEX IF NOT EXISTS user_shelf_items_shelf_added_at_idx
    ON user_shelf_items (shelf_id, added_at DESC);

-- Dashboard and progress pages read one user's most recently touched records.
CREATE INDEX IF NOT EXISTS reading_progress_user_updated_at_idx
    ON reading_progress (user_id, updated_at DESC);

-- Rating distribution and aggregate queries filter by book and rating value.
CREATE INDEX IF NOT EXISTS ratings_book_rating_value_idx
    ON ratings (book_id, rating_value);

-- "My reviews" and review-ranked book lists.
CREATE INDEX IF NOT EXISTS reviews_user_created_at_idx
    ON reviews (user_id, created_at DESC);

-- "My ratings" list ordered by edit time.
CREATE INDEX IF NOT EXISTS ratings_user_updated_at_idx
    ON ratings (user_id, updated_at DESC);

COMMIT;
