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
