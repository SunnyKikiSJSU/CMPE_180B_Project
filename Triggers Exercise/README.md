# Triggers Exercise

Objective: maintain data integrity by automatically updating related data when
changes occur, using the same `university` database as
[HW 1](../HW%201%20-%20Relational%20Algebra%20and%20SQL%20Operations).

## Scenario

Whenever a student enrolls in a new course (a new row in `takes`), their
`tot_cred` in `student` should be recalculated automatically.

## Files

- [trigger.sql](trigger.sql) — `update_tot_cred`, an `AFTER INSERT ON takes`
  trigger that sums `course.credits` for every course the student (`NEW.id`)
  has taken and writes the total back to `student.tot_cred`.
- [test_trigger.sql](test_trigger.sql) — inserts a new enrollment for student
  `00128` and shows `tot_cred` before/after to verify the trigger fires.

## How to run (MySQL)

```bash
mysql -u <user> -p < "../HW 1 - Relational Algebra and SQL Operations/schema.sql"
mysql -u <user> -p < "../HW 1 - Relational Algebra and SQL Operations/data.sql"
mysql -u <user> -p university < trigger.sql
mysql -u <user> -p university < test_trigger.sql
```
