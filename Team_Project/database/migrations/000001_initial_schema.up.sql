BEGIN;

-- Identity
CREATE TABLE users (
    user_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    display_name TEXT NOT NULL,
    avatar_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT users_email_unique UNIQUE (email),
    CONSTRAINT users_email_normalized
        CHECK (email = lower(btrim(email))),
    CONSTRAINT users_email_not_blank
        CHECK (char_length(btrim(email)) > 3),
    CONSTRAINT users_password_hash_not_blank
        CHECK (char_length(btrim(password_hash)) > 0),
    CONSTRAINT users_display_name_not_blank
        CHECK (char_length(btrim(display_name)) > 0)
);

CREATE TABLE user_profiles (
    user_id UUID PRIMARY KEY,
    bio TEXT,
    favorite_genre TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT user_profiles_user_fk
        FOREIGN KEY (user_id)
        REFERENCES users (user_id)
        ON DELETE CASCADE
);

-- Catalog
CREATE TABLE authors (
    author_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    CONSTRAINT authors_name_not_blank
        CHECK (char_length(btrim(name)) > 0)
);

CREATE TABLE categories (
    category_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    CONSTRAINT categories_name_unique UNIQUE (name),
    CONSTRAINT categories_name_not_blank
        CHECK (char_length(btrim(name)) > 0)
);

CREATE TABLE books (
    book_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    google_books_id TEXT,
    isbn13 VARCHAR(13),
    title TEXT NOT NULL,
    description TEXT,
    published_date DATE,
    publisher TEXT,
    page_count INTEGER,
    cover_url TEXT,
    CONSTRAINT books_google_books_id_unique UNIQUE (google_books_id),
    CONSTRAINT books_isbn13_unique UNIQUE (isbn13),
    CONSTRAINT books_title_not_blank
        CHECK (char_length(btrim(title)) > 0),
    CONSTRAINT books_isbn13_format
        CHECK (isbn13 IS NULL OR isbn13 ~ '^[0-9]{13}$'),
    CONSTRAINT books_page_count_positive
        CHECK (page_count IS NULL OR page_count > 0)
);

CREATE TABLE book_authors (
    book_id UUID NOT NULL,
    author_id UUID NOT NULL,
    PRIMARY KEY (book_id, author_id),
    CONSTRAINT book_authors_book_fk
        FOREIGN KEY (book_id)
        REFERENCES books (book_id)
        ON DELETE CASCADE,
    CONSTRAINT book_authors_author_fk
        FOREIGN KEY (author_id)
        REFERENCES authors (author_id)
        ON DELETE CASCADE
);

CREATE TABLE book_categories (
    book_id UUID NOT NULL,
    category_id UUID NOT NULL,
    PRIMARY KEY (book_id, category_id),
    CONSTRAINT book_categories_book_fk
        FOREIGN KEY (book_id)
        REFERENCES books (book_id)
        ON DELETE CASCADE,
    CONSTRAINT book_categories_category_fk
        FOREIGN KEY (category_id)
        REFERENCES categories (category_id)
        ON DELETE CASCADE
);

-- Reading activity
CREATE TABLE shelves (
    shelf_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    name TEXT NOT NULL,
    shelf_type TEXT NOT NULL DEFAULT 'custom',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT shelves_user_fk
        FOREIGN KEY (user_id)
        REFERENCES users (user_id)
        ON DELETE CASCADE,
    CONSTRAINT shelves_user_name_unique UNIQUE (user_id, name),
    CONSTRAINT shelves_name_not_blank
        CHECK (char_length(btrim(name)) > 0),
    CONSTRAINT shelves_type_valid
        CHECK (
            shelf_type IN (
                'want_to_read',
                'currently_reading',
                'finished',
                'paused',
                'custom'
            )
        )
);

CREATE TABLE user_shelf_items (
    shelf_id UUID NOT NULL,
    book_id UUID NOT NULL,
    added_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    moved_at TIMESTAMPTZ,
    PRIMARY KEY (shelf_id, book_id),
    CONSTRAINT user_shelf_items_shelf_fk
        FOREIGN KEY (shelf_id)
        REFERENCES shelves (shelf_id)
        ON DELETE CASCADE,
    CONSTRAINT user_shelf_items_book_fk
        FOREIGN KEY (book_id)
        REFERENCES books (book_id)
        ON DELETE CASCADE
);

CREATE TABLE reading_progress (
    progress_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    book_id UUID NOT NULL,
    percent_complete NUMERIC(5, 2) NOT NULL DEFAULT 0,
    current_page INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'want_to_read',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT reading_progress_user_fk
        FOREIGN KEY (user_id)
        REFERENCES users (user_id)
        ON DELETE CASCADE,
    CONSTRAINT reading_progress_book_fk
        FOREIGN KEY (book_id)
        REFERENCES books (book_id)
        ON DELETE CASCADE,
    CONSTRAINT reading_progress_user_book_unique
        UNIQUE (user_id, book_id),
    CONSTRAINT reading_progress_percent_range
        CHECK (percent_complete >= 0 AND percent_complete <= 100),
    CONSTRAINT reading_progress_current_page_nonnegative
        CHECK (current_page >= 0),
    CONSTRAINT reading_progress_status_valid
        CHECK (status IN ('want_to_read', 'reading', 'completed', 'paused'))
);

CREATE TABLE reading_history (
    history_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    book_id UUID NOT NULL,
    event_type TEXT NOT NULL,
    from_status TEXT,
    to_status TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT reading_history_user_fk
        FOREIGN KEY (user_id)
        REFERENCES users (user_id)
        ON DELETE CASCADE,
    CONSTRAINT reading_history_book_fk
        FOREIGN KEY (book_id)
        REFERENCES books (book_id)
        ON DELETE CASCADE,
    CONSTRAINT reading_history_event_type_not_blank
        CHECK (char_length(btrim(event_type)) > 0),
    CONSTRAINT reading_history_from_status_valid
        CHECK (
            from_status IS NULL
            OR from_status IN ('want_to_read', 'reading', 'completed', 'paused')
        ),
    CONSTRAINT reading_history_to_status_valid
        CHECK (
            to_status IS NULL
            OR to_status IN ('want_to_read', 'reading', 'completed', 'paused')
        )
);

-- Ratings and reviews
CREATE TABLE ratings (
    rating_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    book_id UUID NOT NULL,
    rating_value SMALLINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT ratings_user_fk
        FOREIGN KEY (user_id)
        REFERENCES users (user_id)
        ON DELETE CASCADE,
    CONSTRAINT ratings_book_fk
        FOREIGN KEY (book_id)
        REFERENCES books (book_id)
        ON DELETE CASCADE,
    CONSTRAINT ratings_user_book_unique
        UNIQUE (user_id, book_id),
    CONSTRAINT ratings_value_range
        CHECK (rating_value BETWEEN 1 AND 5)
);

CREATE TABLE reviews (
    review_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    book_id UUID NOT NULL,
    title TEXT,
    review_text TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT reviews_user_fk
        FOREIGN KEY (user_id)
        REFERENCES users (user_id)
        ON DELETE CASCADE,
    CONSTRAINT reviews_book_fk
        FOREIGN KEY (book_id)
        REFERENCES books (book_id)
        ON DELETE CASCADE,
    CONSTRAINT reviews_user_book_unique
        UNIQUE (user_id, book_id),
    CONSTRAINT reviews_text_not_blank
        CHECK (char_length(btrim(review_text)) > 0)
);

CREATE TABLE book_rating_summaries (
    book_id UUID PRIMARY KEY,
    average_rating NUMERIC(3, 2) NOT NULL DEFAULT 0,
    rating_count INTEGER NOT NULL DEFAULT 0,
    review_count INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT book_rating_summaries_book_fk
        FOREIGN KEY (book_id)
        REFERENCES books (book_id)
        ON DELETE CASCADE,
    CONSTRAINT book_rating_summaries_average_range
        CHECK (average_rating >= 0 AND average_rating <= 5),
    CONSTRAINT book_rating_summaries_rating_count_nonnegative
        CHECK (rating_count >= 0),
    CONSTRAINT book_rating_summaries_review_count_nonnegative
        CHECK (review_count >= 0)
);

-- Supporting indexes for foreign-key lookups and common report ordering.
CREATE INDEX book_authors_author_id_idx
    ON book_authors (author_id, book_id);

CREATE INDEX book_categories_category_id_idx
    ON book_categories (category_id, book_id);

CREATE INDEX books_title_lower_idx
    ON books (lower(title));

CREATE INDEX books_published_date_idx
    ON books (published_date DESC);

CREATE INDEX shelves_user_id_idx
    ON shelves (user_id);

CREATE INDEX user_shelf_items_book_id_idx
    ON user_shelf_items (book_id);

CREATE INDEX reading_progress_book_id_idx
    ON reading_progress (book_id);

CREATE INDEX reading_history_user_created_at_idx
    ON reading_history (user_id, created_at DESC);

CREATE INDEX reading_history_book_created_at_idx
    ON reading_history (book_id, created_at DESC);

CREATE INDEX ratings_book_id_idx
    ON ratings (book_id);

CREATE INDEX reviews_book_created_at_idx
    ON reviews (book_id, created_at DESC);

COMMIT;
