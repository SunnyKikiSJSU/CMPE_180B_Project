# Index performance evidence

Five indexes from the migrations, each timed with the app's own query (from
`database/queries/`, with literal arguments) with and without the index.
Required by BACKEND_PLAN.md §5.1 (migration `000004`) and Phase 9. Scripts:
[`database/performance/`](../../database/performance/).

## Method

- **Disposable database** with migrations `000001`-`000008` and generated data
  ([`01_generate_data.sql`](../../database/performance/01_generate_data.sql),
  deterministic with `setseed`). The seed data (201 books) is too small for an
  index to matter: PostgreSQL rightly scans tables that fit in a few pages.

  | Table | Rows |
  | --- | --- |
  | books | 50,000 |
  | users | 2,001 (2,000 ordinary readers, 1 heavy reader) |
  | book_authors / book_categories | 66,653 / 100,001 |
  | reading_progress / user_shelf_items | 110,000 / 110,000 |
  | ratings / reviews | 111,997 / 45,000 |
  | reading_history | 230,000 |

  The heavy reader (`perf-heavy@example.com`) has 10,000 books in progress,
  10,000 ratings, 5,000 reviews, and 30,000 history entries; ordinary readers
  have 50 books each. One book is rated by every reader.
- **Same query, index on and off** ([`02_benchmark.sql`](../../database/performance/02_benchmark.sql)):
  each query runs with the index, then the index is dropped inside a
  subtransaction, the query runs again, and the subtransaction is rolled back,
  so the index is never really removed (the script checks that all of them
  still exist).
- **Measurements:** `EXPLAIN (ANALYZE, BUFFERS)`, one warm-up run, then the
  median of 7 runs. *Pages* is shared buffers hit + read for the whole plan
  (8 KB each). Pages do not depend on the machine; milliseconds do.
  Parallel workers are off so runs compare cleanly.
- Recorded 2026-11-25 on PostgreSQL 18.7 (Docker on Apple Silicon, 2 CPUs,
  4 GB). Two runs on freshly generated data gave the same pages and times
  within about 15%.

## Results: the five optimizations

| # | Index | Query | Without (ms) | With (ms) | Faster | Pages without | Pages with |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | `reading_history_user_created_at_idx` | ListReadingHistory (heavy reader, newest 50) | 22.209 | 0.146 | **152.1x** | 5037 | 119 |
| 2 | `reading_progress_user_updated_at_idx` | RecentReadingActivity (heavy reader, dashboard, 5 rows) | 11.500 | 0.060 | **191.7x** | 2006 | 23 |
| 3 | `book_authors_author_id_idx` | ListBooksByAuthor (author page, first 20) | 1.258 | 0.068 | **18.5x** | 592 | 105 |
| 4 | `reading_history_book_created_at_idx` | ListReadingHistory for one book (heavy reader, book page, 10 rows) | 1.369 | 0.016 | **85.6x** | 483 | 16 |
| 5 | `authors_name_trgm_idx` | ListAuthors search (q = "Author 123", first 20) | 1.377 | 0.194 | **7.1x** | 75 | 58 |

Each query runs 7 to 190 times faster with its index and reads fewer pages
(up to 87 times fewer; author search reads about the same number because the
`authors` table is small, but it checks 11 candidate rows instead of 5,000). The first two are on the dashboard and history
pages, which every signed-in user loads.

### 1. `reading_history_user_created_at_idx`: ListReadingHistory (heavy reader, newest 50)

Without the index PostgreSQL reads all 230,000 history rows (sequential scan), keeps the reader's 30,000, and sorts them to return 50. With `(user_id, created_at DESC)` it walks the index for that reader in date order and stops after 50 rows.

<details><summary>Plans (Sequential scan + sort → index scan that stops at LIMIT)</summary>

```text
--- WITHOUT the index
Limit (actual rows=50.00 loops=1)
  Buffers: shared hit=5037
  ->  Sort (actual rows=50.00 loops=1)
        Sort Key: rh.created_at DESC
        Sort Method: top-N heapsort  Memory: 36kB
        Buffers: shared hit=5037
        ->  Hash Join (actual rows=30000.00 loops=1)
              Hash Cond: (rh.book_id = b.book_id)
              Buffers: shared hit=5037
              ->  Seq Scan on reading_history rh (actual rows=30000.00 loops=1)
                    Filter: (user_id = '9224790e-6a2e-4db4-b4fa-095d0c60683c'::uuid)
                    Rows Removed by Filter: 200000
                    Buffers: shared hit=3286
              ->  Hash (actual rows=50000.00 loops=1)
                    Buckets: 65536  Batches: 1  Memory Usage: 3808kB
                    Buffers: shared hit=1751
                    ->  Seq Scan on books b (actual rows=50000.00 loops=1)
                          Buffers: shared hit=1751
Planning:
  Buffers: shared hit=14
--- WITH the index
Limit (actual rows=50.00 loops=1)
  Buffers: shared hit=119
  ->  Nested Loop (actual rows=50.00 loops=1)
        Buffers: shared hit=119
        ->  Index Scan using reading_history_user_created_at_idx on reading_history rh (actual rows=50.00 loops=1)
              Index Cond: (user_id = '9224790e-6a2e-4db4-b4fa-095d0c60683c'::uuid)
              Index Searches: 1
              Buffers: shared hit=53
        ->  Memoize (actual rows=1.00 loops=50)
              Cache Key: rh.book_id
              Cache Mode: logical
              Hits: 28  Misses: 22  Evictions: 0  Overflows: 0  Memory Usage: 4kB
              Buffers: shared hit=66
              ->  Index Scan using books_pkey on books b (actual rows=1.00 loops=22)
                    Index Cond: (book_id = rh.book_id)
                    Index Searches: 22
                    Buffers: shared hit=66
Planning:
  Buffers: shared hit=14
```

</details>

### 2. `reading_progress_user_updated_at_idx`: RecentReadingActivity (heavy reader, dashboard, 5 rows)

The dashboard's "recent activity" needs the reader's 5 newest progress rows. Without the index it finds all 10,000 of the reader's rows through `reading_progress_user_book_unique` and sorts them (top-N heapsort). With `(user_id, updated_at DESC)` it reads 5 index entries in order.

<details><summary>Plans (Sort of 10,000 rows → 5-row index scan)</summary>

```text
--- WITHOUT the index
Limit (actual rows=5.00 loops=1)
  Buffers: shared hit=2006
  ->  Sort (actual rows=5.00 loops=1)
        Sort Key: rp.updated_at DESC
        Sort Method: top-N heapsort  Memory: 25kB
        Buffers: shared hit=2006
        ->  Hash Join (actual rows=10000.00 loops=1)
              Hash Cond: (b.book_id = rp.book_id)
              Buffers: shared hit=2006
              ->  Seq Scan on books b (actual rows=50000.00 loops=1)
                    Buffers: shared hit=1751
              ->  Hash (actual rows=10000.00 loops=1)
                    Buckets: 16384  Batches: 1  Memory Usage: 832kB
                    Buffers: shared hit=255
                    ->  Bitmap Heap Scan on reading_progress rp (actual rows=10000.00 loops=1)
                          Recheck Cond: (user_id = '9224790e-6a2e-4db4-b4fa-095d0c60683c'::uuid)
                          Heap Blocks: exact=144
                          Buffers: shared hit=255
                          ->  Bitmap Index Scan on reading_progress_user_book_unique (actual rows=10000.00 loops=1)
                                Index Cond: (user_id = '9224790e-6a2e-4db4-b4fa-095d0c60683c'::uuid)
                                Index Searches: 1
                                Buffers: shared hit=111
Planning:
  Buffers: shared hit=14
--- WITH the index
Limit (actual rows=5.00 loops=1)
  Buffers: shared hit=23
  ->  Nested Loop (actual rows=5.00 loops=1)
        Buffers: shared hit=23
        ->  Index Scan using reading_progress_user_updated_at_idx on reading_progress rp (actual rows=5.00 loops=1)
              Index Cond: (user_id = '9224790e-6a2e-4db4-b4fa-095d0c60683c'::uuid)
              Index Searches: 1
              Buffers: shared hit=8
        ->  Memoize (actual rows=1.00 loops=5)
              Cache Key: rp.book_id
              Cache Mode: logical
              Hits: 0  Misses: 5  Evictions: 0  Overflows: 0  Memory Usage: 1kB
              Buffers: shared hit=15
              ->  Index Scan using books_pkey on books b (actual rows=1.00 loops=5)
                    Index Cond: (book_id = rp.book_id)
                    Index Searches: 5
                    Buffers: shared hit=15
Planning:
  Buffers: shared hit=14
```

</details>

### 3. `book_authors_author_id_idx`: ListBooksByAuthor (author page, first 20)

`book_authors`' primary key is `(book_id, author_id)`, which cannot find an author's books. Without the index the query scans all 66,653 rows. `(author_id, book_id)` turns it into one index lookup for the author's 14 books.

<details><summary>Plans (Sequential scan of the join table → index lookup)</summary>

```text
--- WITHOUT the index
Limit (actual rows=14.00 loops=1)
  Buffers: shared hit=592
  ->  Sort (actual rows=14.00 loops=1)
        Sort Key: b.title
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=592
        ->  WindowAgg (actual rows=14.00 loops=1)
              Window: w1 AS ()
              Storage: Memory  Maximum Storage: 18kB
              Buffers: shared hit=592
              ->  Nested Loop Left Join (actual rows=14.00 loops=1)
                    Buffers: shared hit=592
                    ->  Nested Loop (actual rows=14.00 loops=1)
                          Buffers: shared hit=536
                          ->  Index Scan using authors_pkey on authors a (actual rows=1.00 loops=1)
                                Index Cond: (author_id = '00d5a153-e378-4ec4-b76b-931754e59dd0'::uuid)
                                Index Searches: 1
                                Buffers: shared hit=3
                          ->  Nested Loop (actual rows=14.00 loops=1)
                                Buffers: shared hit=533
                                ->  Seq Scan on book_authors ba (actual rows=14.00 loops=1)
                                      Filter: (author_id = '00d5a153-e378-4ec4-b76b-931754e59dd0'::uuid)
                                      Rows Removed by Filter: 66639
                                      Buffers: shared hit=491
                                ->  Index Scan using books_pkey on books b (actual rows=1.00 loops=14)
                                      Index Cond: (book_id = ba.book_id)
                                      Index Searches: 14
                                      Buffers: shared hit=42
                    ->  Index Scan using book_rating_summaries_pkey on book_rating_summaries s (actual rows=1.00 loops=14)
                          Index Cond: (book_id = b.book_id)
                          Index Searches: 14
                          Buffers: shared hit=56
Planning:
  Buffers: shared hit=28
--- WITH the index
Limit (actual rows=14.00 loops=1)
  Buffers: shared hit=105
  ->  Sort (actual rows=14.00 loops=1)
        Sort Key: b.title
        Sort Method: quicksort  Memory: 26kB
        Buffers: shared hit=105
        ->  WindowAgg (actual rows=14.00 loops=1)
              Window: w1 AS ()
              Storage: Memory  Maximum Storage: 18kB
              Buffers: shared hit=105
              ->  Nested Loop (actual rows=14.00 loops=1)
                    Buffers: shared hit=105
                    ->  Index Scan using authors_pkey on authors a (actual rows=1.00 loops=1)
                          Index Cond: (author_id = '00d5a153-e378-4ec4-b76b-931754e59dd0'::uuid)
                          Index Searches: 1
                          Buffers: shared hit=3
                    ->  Nested Loop Left Join (actual rows=14.00 loops=1)
                          Buffers: shared hit=102
                          ->  Nested Loop (actual rows=14.00 loops=1)
                                Buffers: shared hit=46
                                ->  Index Only Scan using book_authors_author_id_idx on book_authors ba (actual rows=14.00 loops=1)
                                      Index Cond: (author_id = '00d5a153-e378-4ec4-b76b-931754e59dd0'::uuid)
                                      Heap Fetches: 0
                                      Index Searches: 1
                                      Buffers: shared hit=4
                                ->  Index Scan using books_pkey on books b (actual rows=1.00 loops=14)
                                      Index Cond: (book_id = ba.book_id)
                                      Index Searches: 14
                                      Buffers: shared hit=42
                          ->  Index Scan using book_rating_summaries_pkey on book_rating_summaries s (actual rows=1.00 loops=14)
                                Index Cond: (book_id = b.book_id)
                                Index Searches: 14
                                Buffers: shared hit=56
Planning:
  Buffers: shared hit=28
```

</details>

### 4. `reading_history_book_created_at_idx`: ListReadingHistory for one book (heavy reader, book page, 10 rows)

The book page shows the reader's history for one book. Without the index the only usable one is `(user_id, created_at)`, so PostgreSQL reads all 30,000 of the heavy reader's entries and filters to 3. `(book_id, created_at DESC)` finds the book's 9 entries directly (from all
readers) and keeps the reader's 3.

<details><summary>Plans (Filter 30,000 rows → direct lookup)</summary>

```text
--- WITHOUT the index
Limit (actual rows=3.00 loops=1)
  Buffers: shared hit=483
  ->  Sort (actual rows=3.00 loops=1)
        Sort Key: rh.created_at DESC
        Sort Method: quicksort  Memory: 25kB
        Buffers: shared hit=483
        ->  Nested Loop (actual rows=3.00 loops=1)
              Buffers: shared hit=483
              ->  Bitmap Heap Scan on reading_history rh (actual rows=3.00 loops=1)
                    Recheck Cond: (user_id = '9224790e-6a2e-4db4-b4fa-095d0c60683c'::uuid)
                    Filter: (book_id = '0000835b-a8bd-4e04-88bb-fef8bc024692'::uuid)
                    Rows Removed by Filter: 29997
                    Heap Blocks: exact=429
                    Buffers: shared hit=474
                    ->  Bitmap Index Scan on reading_history_user_created_at_idx (actual rows=30000.00 loops=1)
                          Index Cond: (user_id = '9224790e-6a2e-4db4-b4fa-095d0c60683c'::uuid)
                          Index Searches: 1
                          Buffers: shared hit=45
              ->  Index Scan using books_pkey on books b (actual rows=1.00 loops=3)
                    Index Cond: (book_id = '0000835b-a8bd-4e04-88bb-fef8bc024692'::uuid)
                    Index Searches: 3
                    Buffers: shared hit=9
--- WITH the index
Limit (actual rows=3.00 loops=1)
  Buffers: shared hit=16
  ->  Nested Loop (actual rows=3.00 loops=1)
        Buffers: shared hit=16
        ->  Index Scan using reading_history_book_created_at_idx on reading_history rh (actual rows=3.00 loops=1)
              Index Cond: (book_id = '0000835b-a8bd-4e04-88bb-fef8bc024692'::uuid)
              Filter: (user_id = '9224790e-6a2e-4db4-b4fa-095d0c60683c'::uuid)
              Rows Removed by Filter: 6
              Index Searches: 1
              Buffers: shared hit=7
        ->  Index Scan using books_pkey on books b (actual rows=1.00 loops=3)
              Index Cond: (book_id = '0000835b-a8bd-4e04-88bb-fef8bc024692'::uuid)
              Index Searches: 3
              Buffers: shared hit=9
```

</details>

### 5. `authors_name_trgm_idx`: ListAuthors search (q = "Author 123", first 20)

Author search is `name ILIKE '%Author 123%'`. A leading wildcard cannot use a B-tree, so without the index every author name is checked (sequential scan). The `pg_trgm` GIN index finds names that contain the trigrams of the search text.

<details><summary>Plans (Sequential scan → trigram index)</summary>

```text
--- WITHOUT the index
Limit (actual rows=11.00 loops=1)
  Buffers: shared hit=75
  ->  Sort (actual rows=11.00 loops=1)
        Sort Key: a.name
        Sort Method: quicksort  Memory: 25kB
        Buffers: shared hit=75
        ->  WindowAgg (actual rows=11.00 loops=1)
              Window: w1 AS ()
              Storage: Memory  Maximum Storage: 17kB
              Buffers: shared hit=75
              ->  GroupAggregate (actual rows=11.00 loops=1)
                    Group Key: a.author_id
                    Buffers: shared hit=75
                    ->  Sort (actual rows=148.00 loops=1)
                          Sort Key: a.author_id
                          Sort Method: quicksort  Memory: 35kB
                          Buffers: shared hit=75
                          ->  Nested Loop Left Join (actual rows=148.00 loops=1)
                                Buffers: shared hit=75
                                ->  Seq Scan on authors a (actual rows=11.00 loops=1)
                                      Filter: (name ~~* '%Author 123%'::text)
                                      Rows Removed by Filter: 4989
                                      Buffers: shared hit=41
                                ->  Index Only Scan using book_authors_author_id_idx on book_authors ba (actual rows=13.45 loops=11)
                                      Index Cond: (author_id = a.author_id)
                                      Heap Fetches: 0
                                      Index Searches: 11
                                      Buffers: shared hit=34
Planning:
  Buffers: shared hit=14
--- WITH the index
Limit (actual rows=11.00 loops=1)
  Buffers: shared hit=58
  ->  Sort (actual rows=11.00 loops=1)
        Sort Key: a.name
        Sort Method: quicksort  Memory: 25kB
        Buffers: shared hit=58
        ->  WindowAgg (actual rows=11.00 loops=1)
              Window: w1 AS ()
              Storage: Memory  Maximum Storage: 17kB
              Buffers: shared hit=58
              ->  GroupAggregate (actual rows=11.00 loops=1)
                    Group Key: a.author_id
                    Buffers: shared hit=58
                    ->  Sort (actual rows=148.00 loops=1)
                          Sort Key: a.author_id
                          Sort Method: quicksort  Memory: 35kB
                          Buffers: shared hit=58
                          ->  Nested Loop Left Join (actual rows=148.00 loops=1)
                                Buffers: shared hit=58
                                ->  Bitmap Heap Scan on authors a (actual rows=11.00 loops=1)
                                      Recheck Cond: (name ~~* '%Author 123%'::text)
                                      Heap Blocks: exact=2
                                      Buffers: shared hit=24
                                      ->  Bitmap Index Scan on authors_name_trgm_idx (actual rows=11.00 loops=1)
                                            Index Cond: (name ~~* '%Author 123%'::text)
                                            Index Searches: 1
                                            Buffers: shared hit=22
                                ->  Index Only Scan using book_authors_author_id_idx on book_authors ba (actual rows=13.45 loops=11)
                                      Index Cond: (author_id = a.author_id)
                                      Heap Fetches: 0
                                      Index Searches: 11
                                      Buffers: shared hit=34
Planning:
  Buffers: shared hit=15
```

</details>

## Where an index did not help

These indexes also exist, and the same benchmark shows little or no gain for
the queries listed. This is expected, and the reasons matter for the design.

| # | Index | Query | Without (ms) | With (ms) | Faster | Pages without | Pages with |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 6 | `ratings_book_rating_value_idx` | GetRatingDistribution (most-rated book) | 0.246 | 0.173 | **1.4x** | 32 | 6 |
| 7 | `book_categories_category_id_idx` | ListBooksByCategory (category page, first 20) | 13.893 | 12.274 | **1.1x** | 10491 | 9773 |
| 8 | `reviews_user_created_at_idx` | ListMyReviews (heavy reader, first 20, with total count) | 13.464 | 13.199 | **1.0x** | 1999 | 1975 |
| 9 | `user_shelf_items_shelf_added_at_idx` | ListShelfBooks core (heavy reader's Currently reading, first 20, with total count) | 16.879 | 16.797 | **1.0x** | 2146 | 2105 |

- **6. `ratings_book_rating_value_idx`**: 5 times fewer pages (an
  index-only scan, without visiting the table), but about the same time,
  because `ratings_book_id_idx (book_id)` already finds the 2,000 ratings
  quickly. The composite index starts with `book_id`, so it can serve every
  query `ratings_book_id_idx` serves: **`ratings_book_id_idx` is
  redundant** and could be dropped to make rating writes slightly cheaper.
- **7-9. List pages with a total count.** `ListBooksByCategory`,
  `ListMyReviews`, and `ListShelfBooks` return one page plus
  `count(*) OVER ()` for the total. To count, PostgreSQL must read *every*
  matching row (2,001 books in the category, 5,000 reviews, 9,969 shelf items), so an index in the sort order cannot stop early at `LIMIT 20`; it
  only replaces a sort that takes a few milliseconds. Most of the time is the
  join with `books`: for thousands of rows PostgreSQL hashes all 50,000 books
  instead of looking each one up, which is the right choice. The cost grows with
  the number of rows per user, not with the table size. For very large
  shelves the fix would be a separate, cached count (or keyset pagination
  without a total), not another index. For normal users (50 books) these
  queries take well under a millisecond.

<details><summary>Plans for 6-9</summary>


#### 6. `ratings_book_rating_value_idx`: GetRatingDistribution (most-rated book)

```text
--- WITHOUT the index
Sort (actual rows=5.00 loops=1)
  Sort Key: rating_value
  Sort Method: quicksort  Memory: 25kB
  Buffers: shared hit=32
  ->  HashAggregate (actual rows=5.00 loops=1)
        Group Key: rating_value
        Batches: 1  Memory Usage: 32kB
        Buffers: shared hit=32
        ->  Bitmap Heap Scan on ratings (actual rows=2000.00 loops=1)
              Recheck Cond: (book_id = '7f9c04ad-f762-4e88-88b2-2b14aeedbdf7'::uuid)
              Heap Blocks: exact=28
              Buffers: shared hit=32
              ->  Bitmap Index Scan on ratings_book_id_idx (actual rows=2000.00 loops=1)
                    Index Cond: (book_id = '7f9c04ad-f762-4e88-88b2-2b14aeedbdf7'::uuid)
                    Index Searches: 1
                    Buffers: shared hit=4
--- WITH the index
GroupAggregate (actual rows=5.00 loops=1)
  Group Key: rating_value
  Buffers: shared hit=6
  ->  Index Only Scan using ratings_book_rating_value_idx on ratings (actual rows=2000.00 loops=1)
        Index Cond: (book_id = '7f9c04ad-f762-4e88-88b2-2b14aeedbdf7'::uuid)
        Heap Fetches: 0
        Index Searches: 1
        Buffers: shared hit=6
```

#### 7. `book_categories_category_id_idx`: ListBooksByCategory (category page, first 20)

```text
--- WITHOUT the index
Limit (actual rows=20.00 loops=1)
  Buffers: shared hit=10491
  ->  Sort (actual rows=20.00 loops=1)
        Sort Key: b.title
        Sort Method: top-N heapsort  Memory: 29kB
        Buffers: shared hit=10491
        ->  WindowAgg (actual rows=2001.00 loops=1)
              Window: w1 AS ()
              Storage: Memory  Maximum Storage: 158kB
              Buffers: shared hit=10491
              ->  Nested Loop Left Join (actual rows=2001.00 loops=1)
                    Buffers: shared hit=10491
                    ->  Hash Join (actual rows=2001.00 loops=1)
                          Hash Cond: (b.book_id = bc.book_id)
                          Buffers: shared hit=2487
                          ->  Seq Scan on books b (actual rows=50000.00 loops=1)
                                Buffers: shared hit=1751
                          ->  Hash (actual rows=2001.00 loops=1)
                                Buckets: 2048  Batches: 1  Memory Usage: 110kB
                                Buffers: shared hit=736
                                ->  Seq Scan on book_categories bc (actual rows=2001.00 loops=1)
                                      Filter: (category_id = '0157a3f3-dabd-4c64-aba9-96ca0e450f01'::uuid)
                                      Rows Removed by Filter: 98000
                                      Buffers: shared hit=736
                    ->  Index Scan using book_rating_summaries_pkey on book_rating_summaries s (actual rows=1.00 loops=2001)
                          Index Cond: (book_id = b.book_id)
                          Index Searches: 2001
                          Buffers: shared hit=8004
Planning:
  Buffers: shared hit=28
--- WITH the index
Limit (actual rows=20.00 loops=1)
  Buffers: shared hit=9773
  ->  Sort (actual rows=20.00 loops=1)
        Sort Key: b.title
        Sort Method: top-N heapsort  Memory: 29kB
        Buffers: shared hit=9773
        ->  WindowAgg (actual rows=2001.00 loops=1)
              Window: w1 AS ()
              Storage: Memory  Maximum Storage: 158kB
              Buffers: shared hit=9773
              ->  Nested Loop Left Join (actual rows=2001.00 loops=1)
                    Buffers: shared hit=9773
                    ->  Hash Join (actual rows=2001.00 loops=1)
                          Hash Cond: (b.book_id = bc.book_id)
                          Buffers: shared hit=1769
                          ->  Seq Scan on books b (actual rows=50000.00 loops=1)
                                Buffers: shared hit=1751
                          ->  Hash (actual rows=2001.00 loops=1)
                                Buckets: 2048  Batches: 1  Memory Usage: 110kB
                                Buffers: shared hit=18
                                ->  Index Only Scan using book_categories_category_id_idx on book_categories bc (actual rows=2001.00 loops=1)
                                      Index Cond: (category_id = '0157a3f3-dabd-4c64-aba9-96ca0e450f01'::uuid)
                                      Heap Fetches: 0
                                      Index Searches: 1
                                      Buffers: shared hit=18
                    ->  Index Scan using book_rating_summaries_pkey on book_rating_summaries s (actual rows=1.00 loops=2001)
                          Index Cond: (book_id = b.book_id)
                          Index Searches: 2001
                          Buffers: shared hit=8004
Planning:
  Buffers: shared hit=28
```

#### 8. `reviews_user_created_at_idx`: ListMyReviews (heavy reader, first 20, with total count)

```text
--- WITHOUT the index
Limit (actual rows=20.00 loops=1)
  Buffers: shared hit=1999
  ->  Sort (actual rows=20.00 loops=1)
        Sort Key: r.created_at DESC
        Sort Method: top-N heapsort  Memory: 29kB
        Buffers: shared hit=1999
        ->  WindowAgg (actual rows=5000.00 loops=1)
              Window: w1 AS ()
              Storage: Memory  Maximum Storage: 495kB
              Buffers: shared hit=1999
              ->  Hash Left Join (actual rows=5000.00 loops=1)
                    Hash Cond: (r.book_id = rt.book_id)
                    Buffers: shared hit=1999
                    ->  Hash Join (actual rows=5000.00 loops=1)
                          Hash Cond: (b.book_id = r.book_id)
                          Buffers: shared hit=1856
                          ->  Seq Scan on books b (actual rows=50000.00 loops=1)
                                Buffers: shared hit=1751
                          ->  Hash (actual rows=5000.00 loops=1)
                                Buckets: 8192  Batches: 1  Memory Usage: 494kB
                                Buffers: shared hit=105
                                ->  Bitmap Heap Scan on reviews r (actual rows=5000.00 loops=1)
                                      Recheck Cond: (user_id = '9224790e-6a2e-4db4-b4fa-095d0c60683c'::uuid)
                                      Heap Blocks: exact=72
                                      Buffers: shared hit=105
                                      ->  Bitmap Index Scan on reviews_user_book_unique (actual rows=5000.00 loops=1)
                                            Index Cond: (user_id = '9224790e-6a2e-4db4-b4fa-095d0c60683c'::uuid)
                                            Index Searches: 1
                                            Buffers: shared hit=33
                    ->  Hash (actual rows=10000.00 loops=1)
                          Buckets: 16384  Batches: 1  Memory Usage: 773kB
                          Buffers: shared hit=143
                          ->  Bitmap Heap Scan on ratings rt (actual rows=10000.00 loops=1)
                                Recheck Cond: (user_id = '9224790e-6a2e-4db4-b4fa-095d0c60683c'::uuid)
                                Heap Blocks: exact=125
                                Buffers: shared hit=143
                                ->  Bitmap Index Scan on ratings_user_updated_at_idx (actual rows=10000.00 loops=1)
                                      Index Cond: (user_id = '9224790e-6a2e-4db4-b4fa-095d0c60683c'::uuid)
                                      Index Searches: 1
                                      Buffers: shared hit=18
Planning:
  Buffers: shared hit=23
--- WITH the index
Limit (actual rows=20.00 loops=1)
  Buffers: shared hit=1975
  ->  Sort (actual rows=20.00 loops=1)
        Sort Key: r.created_at DESC
        Sort Method: top-N heapsort  Memory: 29kB
        Buffers: shared hit=1975
        ->  WindowAgg (actual rows=5000.00 loops=1)
              Window: w1 AS ()
              Storage: Memory  Maximum Storage: 495kB
              Buffers: shared hit=1975
              ->  Hash Left Join (actual rows=5000.00 loops=1)
                    Hash Cond: (r.book_id = rt.book_id)
                    Buffers: shared hit=1975
                    ->  Hash Join (actual rows=5000.00 loops=1)
                          Hash Cond: (b.book_id = r.book_id)
                          Buffers: shared hit=1832
                          ->  Seq Scan on books b (actual rows=50000.00 loops=1)
                                Buffers: shared hit=1751
                          ->  Hash (actual rows=5000.00 loops=1)
                                Buckets: 8192  Batches: 1  Memory Usage: 494kB
                                Buffers: shared hit=81
                                ->  Bitmap Heap Scan on reviews r (actual rows=5000.00 loops=1)
                                      Recheck Cond: (user_id = '9224790e-6a2e-4db4-b4fa-095d0c60683c'::uuid)
                                      Heap Blocks: exact=72
                                      Buffers: shared hit=81
                                      ->  Bitmap Index Scan on reviews_user_created_at_idx (actual rows=5000.00 loops=1)
                                            Index Cond: (user_id = '9224790e-6a2e-4db4-b4fa-095d0c60683c'::uuid)
                                            Index Searches: 1
                                            Buffers: shared hit=9
                    ->  Hash (actual rows=10000.00 loops=1)
                          Buckets: 16384  Batches: 1  Memory Usage: 773kB
                          Buffers: shared hit=143
                          ->  Bitmap Heap Scan on ratings rt (actual rows=10000.00 loops=1)
                                Recheck Cond: (user_id = '9224790e-6a2e-4db4-b4fa-095d0c60683c'::uuid)
                                Heap Blocks: exact=125
                                Buffers: shared hit=143
                                ->  Bitmap Index Scan on ratings_user_updated_at_idx (actual rows=10000.00 loops=1)
                                      Index Cond: (user_id = '9224790e-6a2e-4db4-b4fa-095d0c60683c'::uuid)
                                      Index Searches: 1
                                      Buffers: shared hit=18
Planning:
  Buffers: shared hit=23
```

#### 9. `user_shelf_items_shelf_added_at_idx`: ListShelfBooks core (heavy reader's Currently reading, first 20, with total count)

```text
--- WITHOUT the index
Limit (actual rows=20.00 loops=1)
  Buffers: shared hit=2146
  ->  Sort (actual rows=20.00 loops=1)
        Sort Key: usi.added_at DESC, b.title
        Sort Method: top-N heapsort  Memory: 29kB
        Buffers: shared hit=2146
        ->  WindowAgg (actual rows=9969.00 loops=1)
              Window: w1 AS ()
              Storage: Memory  Maximum Storage: 900kB
              Buffers: shared hit=2146
              ->  Hash Left Join (actual rows=9969.00 loops=1)
                    Hash Cond: (b.book_id = rp.book_id)
                    Buffers: shared hit=2146
                    ->  Nested Loop (actual rows=9969.00 loops=1)
                          Buffers: shared hit=1945
                          ->  Index Scan using shelves_pkey on shelves sh (actual rows=1.00 loops=1)
                                Index Cond: (shelf_id = '8cfec743-3a78-4fc1-b512-34b346616046'::uuid)
                                Filter: (user_id = '9224790e-6a2e-4db4-b4fa-095d0c60683c'::uuid)
                                Index Searches: 1
                                Buffers: shared hit=3
                          ->  Hash Join (actual rows=9969.00 loops=1)
                                Hash Cond: (b.book_id = usi.book_id)
                                Buffers: shared hit=1942
                                ->  Seq Scan on books b (actual rows=50000.00 loops=1)
                                      Buffers: shared hit=1751
                                ->  Hash (actual rows=9969.00 loops=1)
                                      Buckets: 16384  Batches: 1  Memory Usage: 829kB
                                      Buffers: shared hit=191
                                      ->  Bitmap Heap Scan on user_shelf_items usi (actual rows=9969.00 loops=1)
                                            Recheck Cond: (shelf_id = '8cfec743-3a78-4fc1-b512-34b346616046'::uuid)
                                            Heap Blocks: exact=84
                                            Buffers: shared hit=191
                                            ->  Bitmap Index Scan on user_shelf_items_pkey (actual rows=9969.00 loops=1)
                                                  Index Cond: (shelf_id = '8cfec743-3a78-4fc1-b512-34b346616046'::uuid)
                                                  Index Searches: 1
                                                  Buffers: shared hit=107
                    ->  Hash (actual rows=10000.00 loops=1)
                          Buckets: 16384  Batches: 1  Memory Usage: 832kB
                          Buffers: shared hit=201
                          ->  Bitmap Heap Scan on reading_progress rp (actual rows=10000.00 loops=1)
                                Recheck Cond: (user_id = '9224790e-6a2e-4db4-b4fa-095d0c60683c'::uuid)
                                Heap Blocks: exact=144
                                Buffers: shared hit=201
                                ->  Bitmap Index Scan on reading_progress_user_updated_at_idx (actual rows=10000.00 loops=1)
                                      Index Cond: (user_id = '9224790e-6a2e-4db4-b4fa-095d0c60683c'::uuid)
                                      Index Searches: 1
                                      Buffers: shared hit=57
Planning:
  Buffers: shared hit=28
--- WITH the index
Limit (actual rows=20.00 loops=1)
  Buffers: shared hit=2105
  ->  Sort (actual rows=20.00 loops=1)
        Sort Key: usi.added_at DESC, b.title
        Sort Method: top-N heapsort  Memory: 29kB
        Buffers: shared hit=2105
        ->  WindowAgg (actual rows=9969.00 loops=1)
              Window: w1 AS ()
              Storage: Memory  Maximum Storage: 900kB
              Buffers: shared hit=2105
              ->  Hash Left Join (actual rows=9969.00 loops=1)
                    Hash Cond: (b.book_id = rp.book_id)
                    Buffers: shared hit=2105
                    ->  Nested Loop (actual rows=9969.00 loops=1)
                          Buffers: shared hit=1904
                          ->  Index Scan using shelves_pkey on shelves sh (actual rows=1.00 loops=1)
                                Index Cond: (shelf_id = '8cfec743-3a78-4fc1-b512-34b346616046'::uuid)
                                Filter: (user_id = '9224790e-6a2e-4db4-b4fa-095d0c60683c'::uuid)
                                Index Searches: 1
                                Buffers: shared hit=3
                          ->  Hash Join (actual rows=9969.00 loops=1)
                                Hash Cond: (b.book_id = usi.book_id)
                                Buffers: shared hit=1901
                                ->  Seq Scan on books b (actual rows=50000.00 loops=1)
                                      Buffers: shared hit=1751
                                ->  Hash (actual rows=9969.00 loops=1)
                                      Buckets: 16384  Batches: 1  Memory Usage: 829kB
                                      Buffers: shared hit=150
                                      ->  Bitmap Heap Scan on user_shelf_items usi (actual rows=9969.00 loops=1)
                                            Recheck Cond: (shelf_id = '8cfec743-3a78-4fc1-b512-34b346616046'::uuid)
                                            Heap Blocks: exact=84
                                            Buffers: shared hit=150
                                            ->  Bitmap Index Scan on user_shelf_items_shelf_added_at_idx (actual rows=9969.00 loops=1)
                                                  Index Cond: (shelf_id = '8cfec743-3a78-4fc1-b512-34b346616046'::uuid)
                                                  Index Searches: 1
                                                  Buffers: shared hit=66
                    ->  Hash (actual rows=10000.00 loops=1)
                          Buckets: 16384  Batches: 1  Memory Usage: 832kB
                          Buffers: shared hit=201
                          ->  Bitmap Heap Scan on reading_progress rp (actual rows=10000.00 loops=1)
                                Recheck Cond: (user_id = '9224790e-6a2e-4db4-b4fa-095d0c60683c'::uuid)
                                Heap Blocks: exact=144
                                Buffers: shared hit=201
                                ->  Bitmap Index Scan on reading_progress_user_updated_at_idx (actual rows=10000.00 loops=1)
                                      Index Cond: (user_id = '9224790e-6a2e-4db4-b4fa-095d0c60683c'::uuid)
                                      Index Searches: 1
                                      Buffers: shared hit=57
Planning:
  Buffers: shared hit=28
```

</details>

## How to reproduce

See [`database/performance/README.md`](../../database/performance/README.md).
Times will differ by machine; page counts should match.
