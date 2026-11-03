BEGIN;

-- One connected dataset for verifying the schema and database connectivity.
-- Fixed UUIDs and ON CONFLICT clauses make this seed safe to run more than once.
INSERT INTO users (
    user_id,
    email,
    password_hash,
    display_name,
    avatar_url
)
VALUES (
    '00000000-0000-4000-8000-000000000001',
    'smoke.reader@example.com',
    'smoke-test-placeholder-not-a-real-password-hash',
    'Smoke Test Reader',
    NULL
)
ON CONFLICT DO NOTHING;

INSERT INTO user_profiles (
    user_id,
    bio,
    favorite_genre
)
VALUES (
    '00000000-0000-4000-8000-000000000001',
    'Seed profile used to verify database connectivity.',
    'Fantasy'
)
ON CONFLICT DO NOTHING;

INSERT INTO authors (
    author_id,
    name
)
VALUES (
    '00000000-0000-4000-8000-000000000002',
    'Smoke Test Author'
)
ON CONFLICT DO NOTHING;

INSERT INTO categories (
    category_id,
    name
)
VALUES (
    '00000000-0000-4000-8000-000000000003',
    'Smoke Test Category'
)
ON CONFLICT DO NOTHING;

INSERT INTO books (
    book_id,
    google_books_id,
    isbn13,
    title,
    description,
    published_date,
    publisher,
    page_count,
    cover_url
)
VALUES (
    '00000000-0000-4000-8000-000000000004',
    'smoke-test-google-books-id',
    '9780000000001',
    'The Lantern Library',
    'A synthetic book record used only for connectivity testing.',
    DATE '2026-01-15',
    'Smoke Test Press',
    200,
    NULL
)
ON CONFLICT DO NOTHING;

INSERT INTO book_authors (
    book_id,
    author_id
)
VALUES (
    '00000000-0000-4000-8000-000000000004',
    '00000000-0000-4000-8000-000000000002'
)
ON CONFLICT DO NOTHING;

INSERT INTO book_categories (
    book_id,
    category_id
)
VALUES (
    '00000000-0000-4000-8000-000000000004',
    '00000000-0000-4000-8000-000000000003'
)
ON CONFLICT DO NOTHING;

INSERT INTO shelves (
    shelf_id,
    user_id,
    name,
    shelf_type
)
VALUES (
    '00000000-0000-4000-8000-000000000005',
    '00000000-0000-4000-8000-000000000001',
    'Smoke Test Currently Reading',
    'currently_reading'
)
ON CONFLICT DO NOTHING;

INSERT INTO user_shelf_items (
    shelf_id,
    book_id,
    moved_at
)
VALUES (
    '00000000-0000-4000-8000-000000000005',
    '00000000-0000-4000-8000-000000000004',
    CURRENT_TIMESTAMP
)
ON CONFLICT DO NOTHING;

INSERT INTO reading_progress (
    progress_id,
    user_id,
    book_id,
    percent_complete,
    current_page,
    status
)
VALUES (
    '00000000-0000-4000-8000-000000000006',
    '00000000-0000-4000-8000-000000000001',
    '00000000-0000-4000-8000-000000000004',
    42.50,
    85,
    'reading'
)
ON CONFLICT DO NOTHING;

INSERT INTO reading_history (
    history_id,
    user_id,
    book_id,
    event_type,
    from_status,
    to_status
)
VALUES (
    '00000000-0000-4000-8000-000000000007',
    '00000000-0000-4000-8000-000000000001',
    '00000000-0000-4000-8000-000000000004',
    'progress_updated',
    'want_to_read',
    'reading'
)
ON CONFLICT DO NOTHING;

INSERT INTO ratings (
    rating_id,
    user_id,
    book_id,
    rating_value
)
VALUES (
    '00000000-0000-4000-8000-000000000008',
    '00000000-0000-4000-8000-000000000001',
    '00000000-0000-4000-8000-000000000004',
    4
)
ON CONFLICT DO NOTHING;

INSERT INTO reviews (
    review_id,
    user_id,
    book_id,
    title,
    review_text
)
VALUES (
    '00000000-0000-4000-8000-000000000009',
    '00000000-0000-4000-8000-000000000001',
    '00000000-0000-4000-8000-000000000004',
    'Connectivity test review',
    'This review verifies the review table and its relationships.'
)
ON CONFLICT DO NOTHING;

INSERT INTO book_rating_summaries (
    book_id,
    average_rating,
    rating_count,
    review_count
)
VALUES (
    '00000000-0000-4000-8000-000000000004',
    4.00,
    1,
    1
)
ON CONFLICT DO NOTHING;

COMMIT;
