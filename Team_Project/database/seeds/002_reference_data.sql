BEGIN;

-- Reference catalog: 50 authors, 15 categories, 200 books, and their
-- many-to-many links. Fixed UUIDs and ON CONFLICT clauses keep reruns safe.

CREATE TEMP TABLE seed_author_names ON COMMIT DROP AS
SELECT name, ordinality
FROM unnest(ARRAY[
    'Jane Austen',
    'Charles Dickens',
    'George Orwell',
    'Virginia Woolf',
    'Ernest Hemingway',
    'Toni Morrison',
    'Gabriel Garcia Marquez',
    'Haruki Murakami',
    'Ursula K. Le Guin',
    'Octavia E. Butler',
    'Neil Gaiman',
    'Terry Pratchett',
    'Margaret Atwood',
    'Kazuo Ishiguro',
    'Chimamanda Ngozi Adichie',
    'James Baldwin',
    'Fyodor Dostoevsky',
    'Leo Tolstoy',
    'Franz Kafka',
    'Marcel Proust',
    'Jorge Luis Borges',
    'Isabel Allende',
    'Salman Rushdie',
    'Arundhati Roy',
    'Jhumpa Lahiri',
    'Colson Whitehead',
    'Zadie Smith',
    'Elena Ferrante',
    'Italo Calvino',
    'Umberto Eco',
    'Milan Kundera',
    'Roberto Bolano',
    'Doris Lessing',
    'Alice Munro',
    'Raymond Carver',
    'Flannery O''Connor',
    'William Faulkner',
    'John Steinbeck',
    'F. Scott Fitzgerald',
    'Charlotte Bronte',
    'Emily Bronte',
    'Mary Shelley',
    'Bram Stoker',
    'Arthur Conan Doyle',
    'Agatha Christie',
    'J. R. R. Tolkien',
    'C. S. Lewis',
    'Philip K. Dick',
    'Isaac Asimov',
    'Arthur C. Clarke'
]) WITH ORDINALITY AS t(name, ordinality);

CREATE TEMP TABLE seed_category_names ON COMMIT DROP AS
SELECT name, ordinality
FROM unnest(ARRAY[
    'Fantasy',
    'Science Fiction',
    'Romance',
    'Mystery',
    'Thriller',
    'Historical Fiction',
    'Literary Fiction',
    'Biography',
    'Memoir',
    'Self-Help',
    'Business',
    'Philosophy',
    'Poetry',
    'Young Adult',
    'Children'
]) WITH ORDINALITY AS t(name, ordinality);

INSERT INTO authors (author_id, name)
SELECT
    ('00000000-0000-4000-8001-' || lpad(ordinality::text, 12, '0'))::uuid,
    name
FROM seed_author_names
ON CONFLICT DO NOTHING;

INSERT INTO categories (category_id, name)
SELECT
    ('00000000-0000-4000-8002-' || lpad(ordinality::text, 12, '0'))::uuid,
    name
FROM seed_category_names
ON CONFLICT DO NOTHING;

-- 20 adjectives x 10 nouns gives 200 deterministic, distinct titles.
CREATE TEMP TABLE seed_book_data ON COMMIT DROP AS
WITH adjectives AS (
    SELECT word, ordinality
    FROM unnest(ARRAY[
        'Silent', 'Hidden', 'Golden', 'Crimson', 'Distant',
        'Endless', 'Fragile', 'Radiant', 'Forgotten', 'Secret',
        'Wild', 'Quiet', 'Burning', 'Frozen', 'Luminous',
        'Wandering', 'Broken', 'Hopeful', 'Ancient', 'Electric'
    ]) WITH ORDINALITY AS t(word, ordinality)
),
nouns AS (
    SELECT word, ordinality
    FROM unnest(ARRAY[
        'Garden', 'Library', 'River', 'Empire', 'Letter',
        'Door', 'City', 'Memory', 'Voyage', 'Season'
    ]) WITH ORDINALITY AS t(word, ordinality)
)
SELECT
    n,
    (SELECT word FROM adjectives WHERE ordinality = ((n - 1) % 20) + 1)
        || ' ' ||
    (SELECT word FROM nouns WHERE ordinality = ((n - 1) / 20) + 1)
        AS title
FROM generate_series(1, 200) AS n;

INSERT INTO books (
    book_id,
    google_books_id,
    isbn13,
    title,
    description,
    published_date,
    publisher,
    page_count,
    cover_url,
    source,
    published_date_precision
)
-- source = 'seed': "seed-book-<n>" is a synthetic ID, not a Google volume.
-- No seed file may contain data fetched from Google Books (public repository).
SELECT
    ('00000000-0000-4000-8003-' || lpad(n::text, 12, '0'))::uuid,
    'seed-book-' || n,
    lpad((9780000001000 + n)::text, 13, '0'),
    title,
    'A demonstration catalog record titled "' || title ||
        '" used for search, shelf, and report evidence.',
    DATE '1990-01-01' + ((n * 53) % 12000),
    'Book Metaverse Press',
    120 + ((n * 17) % 480),
    'https://picsum.photos/seed/book-' || n || '/400/600',
    'seed',
    'day'
FROM seed_book_data
ON CONFLICT DO NOTHING;

-- Each book gets one primary author; even-numbered books get a co-author.
-- Books are numbered by the integer in google_books_id (seed-book-<n>) and
-- authors by their position in the list above, not by text order.
WITH numbered_books AS (
    SELECT book_id, substring(google_books_id FROM '^seed-book-([0-9]+)$')::int AS n
    FROM books
    WHERE google_books_id ~ '^seed-book-[0-9]+$'
),
numbered_authors AS (
    SELECT a.author_id, s.ordinality AS n
    FROM authors a
    JOIN seed_author_names s
        ON lower(btrim(s.name)) = lower(btrim(a.name))
),
links AS (
    SELECT b.book_id, a.author_id
    FROM numbered_books b
    JOIN numbered_authors a ON a.n = ((b.n - 1) % 50) + 1

    UNION

    SELECT b.book_id, a.author_id
    FROM numbered_books b
    JOIN numbered_authors a ON a.n = ((b.n * 7) % 50) + 1
    WHERE b.n % 2 = 0
)
INSERT INTO book_authors (book_id, author_id)
SELECT book_id, author_id
FROM links
ON CONFLICT DO NOTHING;

-- Each book gets one primary category; every third book gets a second.
WITH numbered_books AS (
    SELECT book_id, substring(google_books_id FROM '^seed-book-([0-9]+)$')::int AS n
    FROM books
    WHERE google_books_id ~ '^seed-book-[0-9]+$'
),
numbered_categories AS (
    SELECT c.category_id, s.ordinality AS n
    FROM categories c
    JOIN seed_category_names s
        ON lower(btrim(s.name)) = lower(btrim(c.name))
),
links AS (
    SELECT b.book_id, c.category_id
    FROM numbered_books b
    JOIN numbered_categories c ON c.n = ((b.n - 1) % 15) + 1

    UNION

    SELECT b.book_id, c.category_id
    FROM numbered_books b
    JOIN numbered_categories c ON c.n = ((b.n * 5) % 15) + 1
    WHERE b.n % 3 = 0
)
INSERT INTO book_categories (book_id, category_id)
SELECT book_id, category_id
FROM links
ON CONFLICT DO NOTHING;

COMMIT;
