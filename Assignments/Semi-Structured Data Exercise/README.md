# Semi-Structured Data Exercise (JSON)

Objective: model a nested JSON document in a MySQL `JSON` column and query
into it with MySQL's `JSON_*` functions.

## Files

- [Assignment1_SemiStructured.sql](Assignment1_SemiStructured.sql) — DDL/DML
  and the three required queries (no answer key separate from the script —
  each query is run directly against `people.doc`).

## Schema

One `people` table: `id`, `name`, and a `doc JSON` column holding
`{"dept": "...", "children": [{"first":"...","last":"..."}, ...],
"interests": ["...", ...]}`.

## Queries

1. **Children last names for a given id, as a JSON array** —
   `JSON_EXTRACT(doc, '$.children[*].last')` using MySQL's JSON path
   wildcard, which returns the result already as a JSON array (no
   `JSON_ARRAYAGG`/`JSON_TABLE` needed for this shape).
2. **Rows where `interests` contains `'databases'`** —
   `JSON_CONTAINS(doc->'$.interests', '"databases"')`.
3. **Per-row `{"name": ..., "dept": ...}` object** —
   `JSON_OBJECT('name', name, 'dept', doc->>'$.dept')`, using the `->>`
   inline path operator so `dept` comes back unquoted (plain string) rather
   than as a quoted JSON scalar.

## How to run (MySQL 5.7+)

```bash
mysql -u <user> -p < Assignment1_SemiStructured.sql
```

Capture a screenshot of the three query outputs after running this for the
assignment deliverable.

## SQL Script

```sql
-- Assignment 1: Semi-Structured Data (JSON)
-- Dialect: MySQL 8.0 (JSON column type + JSON_* functions)

CREATE DATABASE IF NOT EXISTS semi_structured_demo;
USE semi_structured_demo;

DROP TABLE IF EXISTS people;

-- 1. `people` table with a JSON column holding a nested document:
--    dept (string), children (array of {first,last}), interests (array of strings)
CREATE TABLE people (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    doc JSON NOT NULL
);

-- 2. Insert at least two rows.
INSERT INTO people (name, doc) VALUES
('Alice Johnson', JSON_OBJECT(
    'dept', 'Engineering',
    'children', JSON_ARRAY(
        JSON_OBJECT('first', 'Emma', 'last', 'Johnson'),
        JSON_OBJECT('first', 'Liam', 'last', 'Johnson')
    ),
    'interests', JSON_ARRAY('databases', 'hiking', 'chess')
)),
('Bob Smith', JSON_OBJECT(
    'dept', 'Marketing',
    'children', JSON_ARRAY(
        JSON_OBJECT('first', 'Noah', 'last', 'Smith')
    ),
    'interests', JSON_ARRAY('photography', 'travel')
));

-- 3a. Extract all children last names for a given id as a JSON array.
SELECT id, JSON_EXTRACT(doc, '$.children[*].last') AS children_last_names
FROM people
WHERE id = 1;

-- 3b. Filter rows where the interests array contains 'databases'.
SELECT id, name
FROM people
WHERE JSON_CONTAINS(doc->'$.interests', '"databases"');

-- 3c. Generate a JSON object per row: {"name": <name>, "dept": <dept>}
SELECT JSON_OBJECT('name', name, 'dept', doc->>'$.dept') AS person_summary
FROM people;
```

## Query Results

**3a. Children last names for `id = 1` (JSON array):**

```
+----+------------------------+
| id | children_last_names    |
+----+------------------------+
|  1 | ["Johnson", "Johnson"] |
+----+------------------------+
```

**3b. Rows where `interests` contains `'databases'`:**

```
+----+---------------+
| id | name          |
+----+---------------+
|  1 | Alice Johnson |
+----+---------------+
```

**3c. Per-row `{"name": ..., "dept": ...}` object:**

```
+---------------------------------------------------+
| person_summary                                     |
+---------------------------------------------------+
| {"dept": "Engineering", "name": "Alice Johnson"}   |
| {"dept": "Marketing", "name": "Bob Smith"}         |
+---------------------------------------------------+
```

## Short Answer: Normalizing `children` into 1NF

To bring `children` into 1NF, pull it out of the JSON array into its own
table: `children(child_id PK, person_id FK -> people.id, first_name,
last_name)`, with one row per child instead of one array element per row.
`people` keeps only its own scalar columns (`id`, `name`, `dept`); a join
`people.id = children.person_id` reconstructs the parent/child
relationship. The same pattern applies to `interests` — either a
`person_interests(person_id FK, interest)` table if interests are freeform
per person, or a normalized `interests(interest_id PK, label)` lookup table
plus a `person_interests(person_id FK, interest_id FK)` join table if
interests are a shared, finite vocabulary across many people.
