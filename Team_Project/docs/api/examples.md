# API Examples

Base URL: `http://localhost:8080`

## Register and capture the session cookie

```bash
curl -i -c cookies.txt \
  -H 'Content-Type: application/json' \
  -d '{"email":"new.reader@example.com","password":"Password123!","displayName":"New Reader"}' \
  http://localhost:8080/api/auth/register
```

## Log in with the seeded demo account

```bash
curl -i -c cookies.txt \
  -H 'Content-Type: application/json' \
  -d '{"email":"reader1@example.com","password":"Password123!"}' \
  http://localhost:8080/api/auth/login
```

For a non-browser client, use the returned `accessToken`:

```bash
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/auth/me
```

## Search Open Library

The server calls Open Library; nothing is stored.

```bash
curl 'http://localhost:8080/api/external/books?q=dune&page=1&pageSize=20'
curl 'http://localhost:8080/api/external/books?title=dune&author=herbert&sort=newest'
curl 'http://localhost:8080/api/external/books?isbn=9780441013593'
```

Each result has `externalId`, `infoLink`, and `bookId`. `bookId` is set when
the work, or a book with one of its ISBN-13s, is already in the library:

```json
[
  {
    "source": "openlibrary",
    "externalId": "OL893414W",
    "bookId": null,
    "title": "Dune",
    "subtitle": null,
    "authors": ["Frank Herbert"],
    "categories": ["Fiction", "Science fiction", "American literature", "American Science fiction", "Ecology"],
    "description": null,
    "publishedDate": "1965",
    "publisher": "Berkley",
    "pageCount": 604,
    "isbn13": "9780425038918",
    "coverUrl": "https://covers.openlibrary.org/b/id/11481354-M.jpg",
    "infoLink": "https://openlibrary.org/works/OL893414W"
  }
]
```

Recorded from Open Library on 2026-11-23; its data changes over time.

## Import a result before acting on it

Shelves, progress, ratings, and reviews take the local `bookId`. When a result
has `bookId: null`, import it first:

```bash
curl -b cookies.txt \
  -H 'Content-Type: application/json' \
  -d '{"externalId":"OL893414W"}' \
  http://localhost:8080/api/books/import
```

`201` means the book was imported; `200` means it was already in the library.
Both return a `Book` with `source: "openlibrary"`, its description, and
`infoLink`. Use its `id` in the shelf, progress, rating, and review calls
below.

## Search the local library

Searches books already in PostgreSQL (seed, imported, and user-created).

```bash
curl 'http://localhost:8080/api/books?q=night&sort=relevance&page=1&pageSize=20'
curl 'http://localhost:8080/api/books?categoryId=<category-uuid>&sort=rating'
curl 'http://localhost:8080/api/books?authorId=<author-uuid>&sort=newest'
```

Pagination is returned in headers:

```text
X-Total-Count: 12
X-Page: 1
X-Page-Size: 20
```

## Book detail

```bash
curl -b cookies.txt http://localhost:8080/api/books/<book-uuid>
```

When the cookie is present, the response includes `myProgress` and `myRating`.

## Rate and review

```bash
curl -b cookies.txt \
  -H 'Content-Type: application/json' \
  -d '{"value":5}' \
  http://localhost:8080/api/books/<book-uuid>/ratings
```

```bash
curl -b cookies.txt \
  -H 'Content-Type: application/json' \
  -d '{"rating":5,"title":"Loved it","body":"A wonderful read."}' \
  -X PUT http://localhost:8080/api/books/<book-uuid>/review
```

## Shelves

```bash
curl -b cookies.txt http://localhost:8080/api/shelves
```

```bash
curl -b cookies.txt \
  -H 'Content-Type: application/json' \
  -d '{"bookId":"<book-uuid>"}' \
  http://localhost:8080/api/shelves/<currently-reading-shelf-uuid>/books
```

```bash
curl -b cookies.txt \
  -H 'Content-Type: application/json' \
  -d '{"targetShelfId":"<finished-shelf-uuid>"}' \
  -X PATCH \
  http://localhost:8080/api/shelves/<currently-reading-shelf-uuid>/books/<book-uuid>
```

## Progress

```bash
curl -b cookies.txt \
  -H 'Content-Type: application/json' \
  -d '{"currentPage":85,"status":"reading","version":1}' \
  -X PATCH http://localhost:8080/api/reading-progress/<book-uuid>
```

Reaching the final page marks the book completed and moves it to Finished.

## Reports

```bash
curl -b cookies.txt http://localhost:8080/api/reports/reading-history
curl -b cookies.txt http://localhost:8080/api/reports/user-stats
curl 'http://localhost:8080/api/reports/popular?days=30&limit=10'
curl 'http://localhost:8080/api/reports/top-rated?minRatings=5&limit=10'
curl -b cookies.txt 'http://localhost:8080/api/reports/recommendations?limit=10'
curl 'http://localhost:8080/api/reports/authors?limit=20'
curl 'http://localhost:8080/api/reports/categories?limit=20'
curl 'http://localhost:8080/api/reports/review-activity?days=30'
```
