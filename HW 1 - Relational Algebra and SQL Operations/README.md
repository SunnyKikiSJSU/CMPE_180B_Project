# HW 1 - Relational Algebra and SQL Operations

## Part 1: Database Schema Design

### A. Entities

- **department** (dept_name, building, budget)
- **student** (id, name, dept_name, tot_cred)
- **instructor** (id, name, dept_name, salary)
- **classroom** (building, room_number, capacity)
- **course** (course_id, title, dept_name, credits)
- **section** (course_id, sec_id, semester, year, building, room_number, time_slot_id)
- **teaches** (id, course_id, sec_id, semester, year)
- **takes** (id, course_id, sec_id, semester, year, grade)
- **advisor** (s_id, i_id)

### B. Files

- [schema.sql](schema.sql) — `CREATE TABLE` statements with primary keys, foreign keys, and data types.
- [data.sql](data.sql) — `INSERT` statements populating every table with sample data.

### C. How to run (MySQL)

```bash
mysql -u <user> -p < schema.sql
mysql -u <user> -p < data.sql
```

`schema.sql` creates the `university` database and all tables in dependency order
(department/classroom → student/instructor/course → section → teaches/takes/advisor)
so foreign keys resolve correctly. `data.sql` populates the same tables in that order.

## Part 2: Writing SQL Queries

### A. Files

- [queries.sql](queries.sql) — all Part 2 queries (join/aggregation, subqueries, filtering).
- [RESULTS.md](RESULTS.md) — result tables produced by running `queries.sql` against the sample data.

### B. How to run (MySQL)

```bash
mysql -u <user> -p < schema.sql
mysql -u <user> -p < data.sql
mysql -u <user> -p < queries.sql
```

### C. Tasks covered

- **A. Join Operations and Aggregation** — students with total credits and department;
  student count per department; instructor course-load counts.
- **B. Subqueries and Nested Queries** — students who took every section of
  "Database System Concepts" (relational division via nested `NOT EXISTS`); instructors
  who teach no course in the "Comp. Sci." department (`NOT IN` subquery).
- **C. Complex Filtering and Set Operations** — students taking more than one course in
  the same semester (`GROUP BY ... HAVING`); students who never received a grade lower
  than 'B' (`NOT EXISTS` over a letter-grade allow-list).

## Deliverables

- [schema.sql](schema.sql), [data.sql](data.sql), [queries.sql](queries.sql) — all SQL scripts.
- [RESULTS.md](RESULTS.md) — query results captured from a live MySQL run.
- [LLM_PROMPTS.md](LLM_PROMPTS.md) — log of prompts used to help produce this assignment.
