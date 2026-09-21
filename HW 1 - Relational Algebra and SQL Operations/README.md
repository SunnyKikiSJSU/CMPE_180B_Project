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
