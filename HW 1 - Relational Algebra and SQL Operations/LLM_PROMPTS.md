# HW 1 - LLM Prompts Used

This file logs the prompts given to the AI assistant (GitHub Copilot / Claude) used
to help produce the deliverables in this folder, per the assignment's disclosure
requirement.

1. "lets work in this assignment together it is not related to cisco it is personal
   school assignment. **Part 1: Database Schema Design** ... design a database schema
   for a university system with the following entities: Student, Instructor, Course,
   Section, Teaches, Takes, Department, Classroom, Advisor ... Create Tables ... write
   SQL statements to create these tables in MySQL, considering primary keys, foreign
   keys, and data types ... Populate the Database ... write SQL statements to insert a
   variety of realistic records ... make a new folder in this repo with assignment name."
   (Full prompt included the CMPE_180B_Project GitHub repo URL.)

2. "u already have my credential to commit and push to this repo" — asked the
   assistant to push the commit despite a permission error.

3. "ahmedabdelrahman-del / Ahmed_" — provided the correct GitHub account to use for
   pushing (forking) the assignment.

4. "**Part 2: Writing SQL Queries** ... A. Join Operations and Aggregation ...
   B. Subqueries and Nested Queries ... C. Complex Filtering and Set Operations"
   — full text of the Part 2 task list (join/aggregation queries, subquery/division
   queries, filtering and grade-comparison queries), requesting SQL for each task.

5. "**Deliverables:** Students should submit their SQL scripts used to create the
   tables, populate the data, and execute the queries, along with all the results,
   and any LLM prompts used to help you generate your deliverables." — requested this
   prompts log plus the results write-up.

## What the assistant did with each prompt

- Designed and wrote `schema.sql` (table DDL with PK/FK constraints and types) and
  `data.sql` (sample INSERT statements) for prompt 1.
- Diagnosed and resolved a GitHub push permission error (repo write access issue) for
  prompts 2–3, by forking the repo under `ahmedabdelrahman-del` and opening a pull
  request back to `SunnyKikiSJSU/CMPE_180B_Project`.
- Wrote `queries.sql` implementing each Part 2 task (joins/aggregation, division-style
  subqueries, `NOT EXISTS`/`NOT IN` subqueries, `GROUP BY ... HAVING`, and a grade
  comparison using an explicit letter-grade allow-list) for prompt 4.
- Ran all scripts against a local MySQL instance to verify they execute without error
  and produce sensible results, and wrote up `RESULTS.md` plus this file for prompt 5.
