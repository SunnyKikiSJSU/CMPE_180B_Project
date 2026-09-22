# Recursive CTE Exercise

Objective: use a recursive Common Table Expression to walk hierarchical data,
using the same `university` database as
[HW 1](../HW%201%20-%20Relational%20Algebra%20and%20SQL%20Operations).

## Scenario

Given a course ID, find all its prerequisites — direct and indirect — along
with each prerequisite course's title.

## Files

- [prereq_schema.sql](prereq_schema.sql) — adds a `prereq(course_id,
  prereq_id)` table (not present in HW1's schema) plus a sample chain:
  `CS-315 → CS-190 → CS-101` and `CS-315 → EE-181 → PHY-101`.
- [recursive_prereqs.sql](recursive_prereqs.sql) — `WITH RECURSIVE` CTE that
  starts from `@start_course`, recurses through `prereq`, and joins `course`
  for titles.

## How to run (MySQL)

```bash
mysql -u <user> -p < "../HW 1 - Relational Algebra and SQL Operations/schema.sql"
mysql -u <user> -p < "../HW 1 - Relational Algebra and SQL Operations/data.sql"
mysql -u <user> -p university < prereq_schema.sql
mysql -u <user> -p university < recursive_prereqs.sql
```

Change `@start_course` in [recursive_prereqs.sql](recursive_prereqs.sql) to
try other courses (e.g. `CS-101` has no prerequisites and returns no rows).
