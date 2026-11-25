# Schema dictionary

Every table, column, key, constraint, index, and trigger in the Book Metaverse
database: what each one means, who writes it, and what happens on delete.

- **Source of truth:** the migrations in `database/migrations/`
  (`000001`-`000008`).
- **How this page was made:** the column, constraint, index, and trigger
  tables below come from the PostgreSQL 18.7 catalog of a database migrated
  to version 8 (2026-11-25), so types, nullability, defaults, and definitions
  are exact. The meanings and the "written by" notes are written by hand.
- **Related:** the ER diagram is [`ER_diagram.png`](../../ER_diagram.png), with
  its reading guide [`ER_diagram.md`](../../ER_diagram.md). The queries are in
  [`query-catalog.md`](query-catalog.md), and the rules are tested in
  [`docs/evidence/validation.md`](../evidence/validation.md).

## Overview

| Table | Holds | Owned by | Typical size |
| --- | --- | --- | --- |
| `users` | Accounts | The user | One per person |
| `user_profiles` | Bio and favorite genre | The user (1:1) | One per user |
| `books` | Shared catalog | Everyone (shared) | Grows with imports |
| `authors`, `categories` | Shared reference data | Everyone (shared) | — |
| `book_authors`, `book_categories` | Many-to-many links | The book | 1-3 per book |
| `book_rating_summaries` | Cached rating average and counts | The book (1:1) | One per book |
| `shelves` | Status and custom shelves | The user | 4 + custom per user |
| `user_shelf_items` | Books on shelves | The shelf's user | — |
| `reading_progress` | Status, page, percent, version | The user | One per user and tracked book |
| `reading_history` | Append-only reading events | The user | Several per tracked book |
| `ratings` | 1-5 stars | The user | At most one per user and book |
| `reviews` | Written reviews | The user | At most one per user and book |

`schema_migrations` belongs to the migration tool (`golang-migrate`: current
`version`, `dirty` flag) and is not part of the data model.

## Conventions

- **Keys:** every entity has a `uuid` primary key with default
  `gen_random_uuid()`. Link tables use composite primary keys. One-to-one
  tables (`user_profiles`, `book_rating_summaries`) use the parent's key as
  their own.
- **Time:** every timestamp is `timestamptz`. `created_at` defaults to the
  insert time; `updated_at` is set by a `BEFORE UPDATE` trigger
  (`fn_set_updated_at`), not by the application.
- **Text rules:** a required text column also has a `*_not_blank` CHECK
  (`char_length(btrim(x)) > 0`), so whitespace-only values are rejected.
- **Vocabularies:** small fixed sets are `text` plus a CHECK, not enum
  types, so a value can be added in a migration without `ALTER TYPE`.
- **Extensions:** `pgcrypto` (UUIDs), `pg_trgm` (substring search indexes:
  `ILIKE '%...%'` on titles, authors, and categories), and `plpgsql`.

### Reading status and shelf type

A book is on exactly one status shelf per user, and that shelf matches its
`reading_progress.status`. `fn_move_book_to_shelf` and
`fn_update_reading_progress` keep the two in step inside one transaction.

| `shelves.shelf_type` | `reading_progress.status` |
| --- | --- |
| `want_to_read` | `want_to_read` |
| `currently_reading` | `reading` |
| `finished` | `completed` |
| `paused` | `paused` |
| `custom` | (unchanged; custom shelves are tags) |

## Delete and cascade rules

Every foreign key is `ON DELETE CASCADE`. There are no `SET NULL` or
`RESTRICT` rules.

```text
users ──┬─► user_profiles
        ├─► shelves ──► user_shelf_items
        ├─► reading_progress
        ├─► reading_history      (append-only; deletable only through this cascade)
        ├─► ratings ──┐
        └─► reviews ──┴─► triggers recompute book_rating_summaries of the affected books

books ──┬─► book_authors, book_categories
        ├─► book_rating_summaries
        ├─► user_shelf_items, reading_progress, reading_history
        └─► ratings, reviews

authors ──► book_authors          categories ──► book_categories
```

- **Deleting an account** (`DELETE /api/users/me`) removes everything the
  user owns in one statement. Other users' data is unchanged, except that the
  summaries of the books they had rated or reviewed are recomputed.
- **Books, authors, and categories** are never deleted by the API. The
  cascades keep test cleanups and manual deletes consistent.
- **Removing a book from a status shelf** is an API rule, not a cascade: the
  service deletes the progress row too (`DELETE /api/shelves/{id}/books/{bookId}`).
  Removing it from a custom shelf only deletes the `user_shelf_items` row.

## Functions

The canonical source is the migrations; `docs/evidence/transactions.md` and
`concurrency.md` show them running.

| Function | Called by | What it does |
| --- | --- | --- |
| `fn_move_book_to_shelf(user, book, shelf)` | API shelf moves | Locks the user's status shelves and the progress row. Puts the book on the shelf (removing it from the other status shelves for a status shelf), sets status and page, bumps `version`, writes a `shelf_moved` history row |
| `fn_update_reading_progress(user, book, page, status, expected_version)` | `PATCH /api/reading-progress/{bookId}` | Same locks. Rejects a stale `expected_version` with `BM001`; creates or updates progress; writes history (`status_changed`, `book_completed`, or `progress_updated`); moves the book to the matching status shelf |
| `fn_refresh_book_rating_summary(book)` | Rating and review triggers | Locks the summary row, then recomputes average, rating count, and review count from the tables |
| `fn_create_user_defaults(user)` | `users_create_defaults` | Creates the profile and the four status shelves (idempotent) |
| `fn_validate_reading_progress()` | `reading_progress_validate` | Page range against the page count; completion and restart rules; computes `percent_complete` |
| `fn_validate_rating()` | `ratings_validate_value` | Rejects values outside 1-5 with `22023` |
| `fn_block_history_mutation()` | `reading_history_immutable` | Rejects `UPDATE` and direct `DELETE` with `55000` |
| `fn_books_after_insert()` | `books_create_summary` | Creates the book's summary row |
| `fn_ratings_summary_trigger()`, `fn_reviews_summary_trigger()` | Rating and review triggers | Call `fn_refresh_book_rating_summary` for the affected book |
| `fn_set_updated_at()` | `*_set_updated_at` | Sets `updated_at = now()` |

Both reading functions lock the status shelves first, ordered by `shelf_id`,
and then the progress row. Every writer takes the locks in the same order, so
two requests for the same user cannot deadlock.

Errors the database raises on purpose, and what the API returns for them
(`docs/api/errors.md`): `23505` unique → `409 CONFLICT`;
`23503`/`23514`/`22023` → `400 VALIDATION_ERROR`; `BM001` →
`409 STALE_VERSION`; `55000` → `409 CONFLICT` ("Reading history cannot be
changed"); `P0002` → `404 NOT_FOUND`.

## Tables

### `users`

One row per account. Login identity and public display name.

**Written by:** Written by `POST /api/auth/register`, `PATCH /api/users/me` (display name, avatar), `PUT /api/users/me/password`, and `DELETE /api/users/me`.

| Column | Type | Null | Default | Meaning |
| --- | --- | --- | --- | --- |
| `user_id` | `uuid` | no | `gen_random_uuid()` | Primary key. |
| `email` | `text` | no |  | Login email, stored lower-case and trimmed (the API normalizes it; the CHECK rejects anything else), so the plain `UNIQUE` is case-insensitive. |
| `password_hash` | `text` | no |  | bcrypt hash of the password. Never returned by the API. |
| `display_name` | `text` | no |  | Name shown on reviews and in the app. |
| `avatar_url` | `text` | yes |  | Optional image URL. |
| `created_at` | `timestamp with time zone` | no | `CURRENT_TIMESTAMP` | Registration time. |

| Constraint | Kind | Definition |
| --- | --- | --- |
| `users_pkey` | Primary key | `PRIMARY KEY (user_id)` |
| `users_email_unique` | Unique | `UNIQUE (email)` |
| `users_display_name_not_blank` | Check | `CHECK ((char_length(btrim(display_name)) > 0))` |
| `users_email_normalized` | Check | `CHECK ((email = lower(btrim(email))))` |
| `users_email_not_blank` | Check | `CHECK ((char_length(btrim(email)) > 3))` |
| `users_password_hash_not_blank` | Check | `CHECK ((char_length(btrim(password_hash)) > 0))` |

| Trigger | When | Function |
| --- | --- | --- |
| `users_create_defaults` | AFTER INSERT | `fn_users_after_insert()` |

Inserting a user fires `users_create_defaults`, which creates the `user_profiles` row and the four status shelves in the same transaction.

### `user_profiles`

Optional profile details, one-to-one with `users` (the primary key is also the foreign key).

**Written by:** Created by the `users_create_defaults` trigger; updated by `PATCH /api/users/me`.

| Column | Type | Null | Default | Meaning |
| --- | --- | --- | --- | --- |
| `user_id` | `uuid` | no |  | Primary key and foreign key to `users`. |
| `bio` | `text` | yes |  | Free-text bio. |
| `favorite_genre` | `text` | yes |  | Free-text favorite genre. |
| `updated_at` | `timestamp with time zone` | no | `CURRENT_TIMESTAMP` | Last change (set by `user_profiles_set_updated_at`). |

| Constraint | Kind | Definition |
| --- | --- | --- |
| `user_profiles_pkey` | Primary key | `PRIMARY KEY (user_id)` |
| `user_profiles_user_fk` | Foreign key | `FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE` |

| Trigger | When | Function |
| --- | --- | --- |
| `user_profiles_set_updated_at` | BEFORE UPDATE | `fn_set_updated_at()` |

### `books`

The shared catalog: seeded books, books added by users, and books imported from Open Library.

**Written by:** `POST /api/books` (source `user`), `POST /api/books/import` (source `openlibrary`), `PATCH /api/books/{bookId}` (only `user` and `seed` books; `UpdateBook` skips imported ones), and the seeds (source `seed`).

| Column | Type | Null | Default | Meaning |
| --- | --- | --- | --- | --- |
| `book_id` | `uuid` | no | `gen_random_uuid()` | Primary key; every other table refers to books by this. |
| `google_books_id` | `text` | yes |  | Legacy Google Books volume ID (source `google`, from before Open Library). Unique when present. |
| `isbn13` | `character varying(13)` | yes |  | 13-digit ISBN without dashes. Unique when present; the import deduplicates on it. |
| `title` | `text` | no |  | Title (not blank). |
| `description` | `text` | yes |  | Description; also feeds `search_vector`. |
| `published_date` | `date` | yes |  | Publication date. |
| `publisher` | `text` | yes |  | Publisher. |
| `page_count` | `integer` | yes |  | Number of pages, positive when known. Progress pages are checked against it. |
| `cover_url` | `text` | yes |  | Cover image URL (Open Library covers or placeholders). |
| `search_vector` | `tsvector` | yes | generated | Full-text search document: title (weight A) + description (weight B), English configuration. Generated and stored by PostgreSQL. |
| `source` | `text` | no | `'user'` | Where the row came from: `seed`, `user`, `openlibrary`, or `google` (legacy). |
| `published_date_precision` | `text` | yes |  | `year`, `month`, or `day`: how much of `published_date` is real (Open Library often gives only a year). Present exactly when the date is. |
| `imported_at` | `timestamp with time zone` | yes |  | When the book was imported. Required for `openlibrary` and `google` books. |
| `openlibrary_id` | `text` | yes |  | Open Library work ID such as `OL893414W`. Unique; the import upserts on it. |

| Constraint | Kind | Definition |
| --- | --- | --- |
| `books_pkey` | Primary key | `PRIMARY KEY (book_id)` |
| `books_google_books_id_unique` | Unique | `UNIQUE (google_books_id)` |
| `books_isbn13_unique` | Unique | `UNIQUE (isbn13)` |
| `books_openlibrary_id_unique` | Unique | `UNIQUE (openlibrary_id)` |
| `books_google_source_complete` | Check | `CHECK (((source <> 'google') OR ((google_books_id IS NOT NULL) AND (imported_at IS NOT NULL))))` |
| `books_isbn13_format` | Check | `CHECK (((isbn13 IS NULL) OR ((isbn13) ~ '^[0-9]{13}$')))` |
| `books_openlibrary_id_format` | Check | `CHECK (((openlibrary_id IS NULL) OR (openlibrary_id ~ '^OL[0-9]+W$')))` |
| `books_openlibrary_source_complete` | Check | `CHECK (((source <> 'openlibrary') OR ((openlibrary_id IS NOT NULL) AND (imported_at IS NOT NULL))))` |
| `books_page_count_positive` | Check | `CHECK (((page_count IS NULL) OR (page_count > 0)))` |
| `books_published_date_precision_matches` | Check | `CHECK (((published_date IS NULL) = (published_date_precision IS NULL)))` |
| `books_published_date_precision_valid` | Check | `CHECK (((published_date_precision IS NULL) OR (published_date_precision = ANY (ARRAY['year', 'month', 'day']))))` |
| `books_source_valid` | Check | `CHECK ((source = ANY (ARRAY['seed', 'google', 'openlibrary', 'user'])))` |
| `books_title_not_blank` | Check | `CHECK ((char_length(btrim(title)) > 0))` |

| Index | Definition |
| --- | --- |
| `books_published_date_idx` | `btree (published_date DESC)` |
| `books_search_vector_gin_idx` | `gin (search_vector)` |
| `books_title_lower_idx` | `btree (lower(title))` |
| `books_title_trgm_idx` | `gin (title gin_trgm_ops)` |

| Trigger | When | Function |
| --- | --- | --- |
| `books_create_summary` | AFTER INSERT | `fn_books_after_insert()` |

Inserting a book fires `books_create_summary`, which creates its `book_rating_summaries` row.

### `authors`

Book authors, shared by all users.

**Written by:** Created when a book is created or imported (`UpsertAuthor`, matched by name). Never deleted by the API.

| Column | Type | Null | Default | Meaning |
| --- | --- | --- | --- | --- |
| `author_id` | `uuid` | no | `gen_random_uuid()` | Primary key. |
| `name` | `text` | no |  | Display name. Unique ignoring case and surrounding spaces (`authors_name_ci_unique_idx`). |

| Constraint | Kind | Definition |
| --- | --- | --- |
| `authors_pkey` | Primary key | `PRIMARY KEY (author_id)` |
| `authors_name_not_blank` | Check | `CHECK ((char_length(btrim(name)) > 0))` |

| Index | Definition |
| --- | --- |
| `authors_name_ci_unique_idx` | `btree (lower(btrim(name)))` |
| `authors_name_trgm_idx` | `gin (name gin_trgm_ops)` |

Authors are identified by name, so two different people with the same name share one row (`ER_diagram.md` §8).

### `book_authors`

Many-to-many link between books and authors.

**Written by:** Rewritten with the book on create, import, and update (`ClearBookAuthors` + `AddBookAuthor`).

| Column | Type | Null | Default | Meaning |
| --- | --- | --- | --- | --- |
| `book_id` | `uuid` | no |  | The book. |
| `author_id` | `uuid` | no |  | One of its authors. |

| Constraint | Kind | Definition |
| --- | --- | --- |
| `book_authors_pkey` | Primary key | `PRIMARY KEY (book_id, author_id)` |
| `book_authors_author_fk` | Foreign key | `FOREIGN KEY (author_id) REFERENCES authors(author_id) ON DELETE CASCADE` |
| `book_authors_book_fk` | Foreign key | `FOREIGN KEY (book_id) REFERENCES books(book_id) ON DELETE CASCADE` |

| Index | Definition |
| --- | --- |
| `book_authors_author_id_idx` | `btree (author_id, book_id)` |

### `categories`

Book categories (subjects), shared by all users.

**Written by:** Created when a book is created or imported (`UpsertCategory`). Never deleted by the API.

| Column | Type | Null | Default | Meaning |
| --- | --- | --- | --- | --- |
| `category_id` | `uuid` | no | `gen_random_uuid()` | Primary key. |
| `name` | `text` | no |  | Display name; unique, also ignoring case and spaces. |

| Constraint | Kind | Definition |
| --- | --- | --- |
| `categories_pkey` | Primary key | `PRIMARY KEY (category_id)` |
| `categories_name_unique` | Unique | `UNIQUE (name)` |
| `categories_name_not_blank` | Check | `CHECK ((char_length(btrim(name)) > 0))` |

| Index | Definition |
| --- | --- |
| `categories_name_ci_unique_idx` | `btree (lower(btrim(name)))` |
| `categories_name_trgm_idx` | `gin (name gin_trgm_ops)` |

### `book_categories`

Many-to-many link between books and categories.

**Written by:** Rewritten with the book (`ClearBookCategories` + `AddBookCategory`).

| Column | Type | Null | Default | Meaning |
| --- | --- | --- | --- | --- |
| `book_id` | `uuid` | no |  | The book. |
| `category_id` | `uuid` | no |  | One of its categories. |

| Constraint | Kind | Definition |
| --- | --- | --- |
| `book_categories_pkey` | Primary key | `PRIMARY KEY (book_id, category_id)` |
| `book_categories_book_fk` | Foreign key | `FOREIGN KEY (book_id) REFERENCES books(book_id) ON DELETE CASCADE` |
| `book_categories_category_fk` | Foreign key | `FOREIGN KEY (category_id) REFERENCES categories(category_id) ON DELETE CASCADE` |

| Index | Definition |
| --- | --- |
| `book_categories_category_id_idx` | `btree (category_id, book_id)` |

### `book_rating_summaries`

Cached rating average and counts per book, so catalog pages and reports do not aggregate `ratings` and `reviews` on every request.

**Written by:** Only the triggers: created by `books_create_summary`, recomputed by `fn_refresh_book_rating_summary` after every rating or review insert, update, or delete. The API never writes it directly.

| Column | Type | Null | Default | Meaning |
| --- | --- | --- | --- | --- |
| `book_id` | `uuid` | no |  | Primary key and foreign key: one summary per book. |
| `average_rating` | `numeric(3,2)` | no | `0` | Average of `ratings.rating_value`, rounded to 2 decimals; 0 when there are no ratings. |
| `rating_count` | `integer` | no | `0` | Number of ratings. |
| `review_count` | `integer` | no | `0` | Number of reviews. |
| `updated_at` | `timestamp with time zone` | no | `CURRENT_TIMESTAMP` | Last recomputation. |

| Constraint | Kind | Definition |
| --- | --- | --- |
| `book_rating_summaries_pkey` | Primary key | `PRIMARY KEY (book_id)` |
| `book_rating_summaries_book_fk` | Foreign key | `FOREIGN KEY (book_id) REFERENCES books(book_id) ON DELETE CASCADE` |
| `book_rating_summaries_average_range` | Check | `CHECK (((average_rating >= (0)::numeric) AND (average_rating <= (5)::numeric)))` |
| `book_rating_summaries_rating_count_nonnegative` | Check | `CHECK ((rating_count >= 0))` |
| `book_rating_summaries_review_count_nonnegative` | Check | `CHECK ((review_count >= 0))` |

The refresh locks this row (`FOR UPDATE`) before reading the aggregates, so concurrent ratings cannot overwrite each other's result (migration `000007`; see `docs/evidence/concurrency.md`).

### `shelves`

A user's shelves: four status shelves (one each) plus any number of custom shelves.

**Written by:** Status shelves: the `users_create_defaults` trigger. Custom shelves: `POST`/`PATCH`/`DELETE /api/shelves`. Status shelves cannot be renamed or deleted, and a user can have at most 50 custom shelves (both enforced by the API).

| Column | Type | Null | Default | Meaning |
| --- | --- | --- | --- | --- |
| `shelf_id` | `uuid` | no | `gen_random_uuid()` | Primary key. |
| `user_id` | `uuid` | no |  | Owner. |
| `name` | `text` | no |  | Display name; unique per user. |
| `shelf_type` | `text` | no | `'custom'` | `want_to_read`, `currently_reading`, `finished`, `paused` (status shelves), or `custom`. |
| `created_at` | `timestamp with time zone` | no | `CURRENT_TIMESTAMP` | Creation time. |

| Constraint | Kind | Definition |
| --- | --- | --- |
| `shelves_pkey` | Primary key | `PRIMARY KEY (shelf_id)` |
| `shelves_user_name_unique` | Unique | `UNIQUE (user_id, name)` |
| `shelves_user_fk` | Foreign key | `FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE` |
| `shelves_name_not_blank` | Check | `CHECK ((char_length(btrim(name)) > 0))` |
| `shelves_type_valid` | Check | `CHECK ((shelf_type = ANY (ARRAY['want_to_read', 'currently_reading', 'finished', 'paused', 'custom'])))` |

| Index | Definition |
| --- | --- |
| `shelves_one_per_status_type_idx` | `btree (user_id, shelf_type) WHERE (shelf_type <> 'custom')` |
| `shelves_user_id_idx` | `btree (user_id)` |

`shelves_one_per_status_type_idx` is a partial unique index: at most one shelf of each status type per user, and any number of `custom` shelves.

### `user_shelf_items`

Which books are on which shelf.

**Written by:** `fn_move_book_to_shelf` and `fn_update_reading_progress` (status shelves, kept in step with progress), `POST`/`PATCH`/`DELETE /api/shelves/{shelfId}/books`.

| Column | Type | Null | Default | Meaning |
| --- | --- | --- | --- | --- |
| `shelf_id` | `uuid` | no |  | The shelf (and through it, the owner). |
| `book_id` | `uuid` | no |  | The book. |
| `added_at` | `timestamp with time zone` | no | `CURRENT_TIMESTAMP` | When the book was put on this shelf. |
| `moved_at` | `timestamp with time zone` | yes |  | Last time it was moved onto this shelf again. |

| Constraint | Kind | Definition |
| --- | --- | --- |
| `user_shelf_items_pkey` | Primary key | `PRIMARY KEY (shelf_id, book_id)` |
| `user_shelf_items_book_fk` | Foreign key | `FOREIGN KEY (book_id) REFERENCES books(book_id) ON DELETE CASCADE` |
| `user_shelf_items_shelf_fk` | Foreign key | `FOREIGN KEY (shelf_id) REFERENCES shelves(shelf_id) ON DELETE CASCADE` |

| Index | Definition |
| --- | --- |
| `user_shelf_items_book_id_idx` | `btree (book_id)` |
| `user_shelf_items_shelf_added_at_idx` | `btree (shelf_id, added_at DESC)` |

A book is on at most one status shelf per user (the functions remove it from the other status shelves); custom shelves are additive.

### `reading_progress`

One row per user and book that the user is tracking: status, page, and percentage.

**Written by:** `fn_update_reading_progress` (`PATCH /api/reading-progress/{bookId}`), `fn_move_book_to_shelf` (status shelf moves), and `DELETE /api/reading-progress/{bookId}`.

| Column | Type | Null | Default | Meaning |
| --- | --- | --- | --- | --- |
| `progress_id` | `uuid` | no | `gen_random_uuid()` | Primary key. |
| `user_id` | `uuid` | no |  | Reader. |
| `book_id` | `uuid` | no |  | Book. |
| `percent_complete` | `numeric(5,2)` | no | `0` | 0-100. Computed from `current_page` when the book has a page count; otherwise the value the client sent. |
| `current_page` | `integer` | no | `0` | Current page, 0 to `books.page_count`. |
| `status` | `text` | no | `'want_to_read'` | `want_to_read`, `reading`, `completed`, or `paused`; matches the status shelf the book is on. |
| `updated_at` | `timestamp with time zone` | no | `CURRENT_TIMESTAMP` | Last change (trigger). |
| `version` | `integer` | no | `1` | Optimistic-lock counter, +1 on every change. A save with an old version raises `BM001` (API: `409 STALE_VERSION`). |

| Constraint | Kind | Definition |
| --- | --- | --- |
| `reading_progress_pkey` | Primary key | `PRIMARY KEY (progress_id)` |
| `reading_progress_user_book_unique` | Unique | `UNIQUE (user_id, book_id)` |
| `reading_progress_book_fk` | Foreign key | `FOREIGN KEY (book_id) REFERENCES books(book_id) ON DELETE CASCADE` |
| `reading_progress_user_fk` | Foreign key | `FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE` |
| `reading_progress_current_page_nonnegative` | Check | `CHECK ((current_page >= 0))` |
| `reading_progress_percent_range` | Check | `CHECK (((percent_complete >= (0)::numeric) AND (percent_complete <= (100)::numeric)))` |
| `reading_progress_status_valid` | Check | `CHECK ((status = ANY (ARRAY['want_to_read', 'reading', 'completed', 'paused'])))` |

| Index | Definition |
| --- | --- |
| `reading_progress_book_id_idx` | `btree (book_id)` |
| `reading_progress_user_updated_at_idx` | `btree (user_id, updated_at DESC)` |

| Trigger | When | Function |
| --- | --- | --- |
| `reading_progress_set_updated_at` | BEFORE UPDATE | `fn_set_updated_at()` |
| `reading_progress_validate` | BEFORE INSERT OR UPDATE | `fn_validate_reading_progress()` |

`reading_progress_validate` checks the page against the page count and applies the completion rules: reaching the last page sets `completed`, marking `completed` jumps to the last page, and leaving `completed` restarts at page 0.

### `reading_history`

Append-only log of reading events, used for the timeline and the reading reports.

**Written by:** Only `fn_update_reading_progress` and `fn_move_book_to_shelf`. Rows are never updated or deleted, except through the cascade when the user or book is deleted.

| Column | Type | Null | Default | Meaning |
| --- | --- | --- | --- | --- |
| `history_id` | `uuid` | no | `gen_random_uuid()` | Primary key. |
| `user_id` | `uuid` | no |  | Reader. |
| `book_id` | `uuid` | no |  | Book. |
| `event_type` | `text` | no |  | `status_changed`, `book_completed`, `progress_updated`, or `shelf_moved`. |
| `from_status` | `text` | yes |  | Status before the event (null when not a status change). |
| `to_status` | `text` | yes |  | Status after the event. |
| `created_at` | `timestamp with time zone` | no | `CURRENT_TIMESTAMP` | Event time. |

| Constraint | Kind | Definition |
| --- | --- | --- |
| `reading_history_pkey` | Primary key | `PRIMARY KEY (history_id)` |
| `reading_history_book_fk` | Foreign key | `FOREIGN KEY (book_id) REFERENCES books(book_id) ON DELETE CASCADE` |
| `reading_history_user_fk` | Foreign key | `FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE` |
| `reading_history_event_type_not_blank` | Check | `CHECK ((char_length(btrim(event_type)) > 0))` |
| `reading_history_from_status_valid` | Check | `CHECK (((from_status IS NULL) OR (from_status = ANY (ARRAY['want_to_read', 'reading', 'completed', 'paused']))))` |
| `reading_history_to_status_valid` | Check | `CHECK (((to_status IS NULL) OR (to_status = ANY (ARRAY['want_to_read', 'reading', 'completed', 'paused']))))` |

| Index | Definition |
| --- | --- |
| `reading_history_book_created_at_idx` | `btree (book_id, created_at DESC)` |
| `reading_history_user_created_at_idx` | `btree (user_id, created_at DESC)` |

| Trigger | When | Function |
| --- | --- | --- |
| `reading_history_immutable` | BEFORE DELETE OR UPDATE | `fn_block_history_mutation()` |

`reading_history_immutable` raises `55000` on `UPDATE` and on a direct `DELETE`. Deletes coming from the `ON DELETE CASCADE` of `users` or `books` are allowed (they run at trigger depth > 1).

### `ratings`

A user's 1-5 star rating of a book (at most one per user and book).

**Written by:** `POST`/`DELETE /api/books/{bookId}/ratings` and `PUT /api/books/{bookId}/review` (review and rating in one transaction).

| Column | Type | Null | Default | Meaning |
| --- | --- | --- | --- | --- |
| `rating_id` | `uuid` | no | `gen_random_uuid()` | Primary key. |
| `user_id` | `uuid` | no |  | Rater. |
| `book_id` | `uuid` | no |  | Book. |
| `rating_value` | `smallint` | no |  | 1 to 5 (checked by `ratings_validate_value` with SQLSTATE `22023`, and by the CHECK). |
| `created_at` | `timestamp with time zone` | no | `CURRENT_TIMESTAMP` | First rating. |
| `updated_at` | `timestamp with time zone` | no | `CURRENT_TIMESTAMP` | Last change (trigger). |

| Constraint | Kind | Definition |
| --- | --- | --- |
| `ratings_pkey` | Primary key | `PRIMARY KEY (rating_id)` |
| `ratings_user_book_unique` | Unique | `UNIQUE (user_id, book_id)` |
| `ratings_book_fk` | Foreign key | `FOREIGN KEY (book_id) REFERENCES books(book_id) ON DELETE CASCADE` |
| `ratings_user_fk` | Foreign key | `FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE` |
| `ratings_value_range` | Check | `CHECK (((rating_value >= 1) AND (rating_value <= 5)))` |

| Index | Definition |
| --- | --- |
| `ratings_book_id_idx` | `btree (book_id)` |
| `ratings_book_rating_value_idx` | `btree (book_id, rating_value)` |
| `ratings_user_updated_at_idx` | `btree (user_id, updated_at DESC)` |

| Trigger | When | Function |
| --- | --- | --- |
| `ratings_refresh_summary` | AFTER DELETE OR INSERT OR UPDATE | `fn_ratings_summary_trigger()` |
| `ratings_set_updated_at` | BEFORE UPDATE | `fn_set_updated_at()` |
| `ratings_validate_value` | BEFORE INSERT OR UPDATE | `fn_validate_rating()` |

Every change refreshes `book_rating_summaries`.

### `reviews`

A user's written review of a book (at most one per user and book).

**Written by:** `POST /api/books/{bookId}/reviews`, `PUT /api/books/{bookId}/review`, `PATCH`/`DELETE /api/reviews/{reviewId}` (owner only).

| Column | Type | Null | Default | Meaning |
| --- | --- | --- | --- | --- |
| `review_id` | `uuid` | no | `gen_random_uuid()` | Primary key. |
| `user_id` | `uuid` | no |  | Author of the review. |
| `book_id` | `uuid` | no |  | Book. |
| `title` | `text` | yes |  | Optional headline. |
| `review_text` | `text` | no |  | Review body (not blank). |
| `created_at` | `timestamp with time zone` | no | `CURRENT_TIMESTAMP` | Creation time. |
| `updated_at` | `timestamp with time zone` | no | `CURRENT_TIMESTAMP` | Last edit (trigger). |

| Constraint | Kind | Definition |
| --- | --- | --- |
| `reviews_pkey` | Primary key | `PRIMARY KEY (review_id)` |
| `reviews_user_book_unique` | Unique | `UNIQUE (user_id, book_id)` |
| `reviews_book_fk` | Foreign key | `FOREIGN KEY (book_id) REFERENCES books(book_id) ON DELETE CASCADE` |
| `reviews_user_fk` | Foreign key | `FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE` |
| `reviews_text_not_blank` | Check | `CHECK ((char_length(btrim(review_text)) > 0))` |

| Index | Definition |
| --- | --- |
| `reviews_book_created_at_idx` | `btree (book_id, created_at DESC)` |
| `reviews_user_created_at_idx` | `btree (user_id, created_at DESC)` |

| Trigger | When | Function |
| --- | --- | --- |
| `reviews_refresh_summary` | AFTER DELETE OR INSERT OR UPDATE | `fn_reviews_summary_trigger()` |
| `reviews_set_updated_at` | BEFORE UPDATE | `fn_set_updated_at()` |

Every change refreshes `review_count` in `book_rating_summaries`.

## Checking this page against a database

```sql
-- Columns
SELECT table_name, column_name, format_type(a.atttypid, a.atttypmod) AS type, is_nullable, column_default
FROM information_schema.columns c
JOIN pg_attribute a ON a.attrelid = (c.table_schema || '.' || c.table_name)::regclass AND a.attname = c.column_name
WHERE c.table_schema = 'public' ORDER BY table_name, ordinal_position;
-- Constraints (without NOT NULL)
SELECT conrelid::regclass, conname, contype, pg_get_constraintdef(oid)
FROM pg_constraint WHERE connamespace = 'public'::regnamespace AND contype <> 'n' ORDER BY 1, 2;
-- Indexes and triggers
SELECT tablename, indexname, indexdef FROM pg_indexes WHERE schemaname = 'public' ORDER BY 1, 2;
SELECT event_object_table, trigger_name, action_timing, event_manipulation, action_statement
FROM information_schema.triggers WHERE trigger_schema = 'public' ORDER BY 1, 2;
```

A migration that changes a table, column, constraint, index, or trigger must
update this page, `ER_diagram.md`, and `database/schema/schema.sql` in the
same PR.
