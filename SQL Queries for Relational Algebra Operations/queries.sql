-- SQL Queries for Relational Algebra Operations
-- Tables (ra_student, ra_cs_student, ra_math_student, ra_instructor, ra_course) are defined in schema.sql

-- A. Selection
-- Relational Algebra: σ dept_name='Computer Science' (ra_student)
-- Task: Retrieve all students from the "Computer Science" department.
SELECT *
FROM ra_student
WHERE dept_name = 'Computer Science';

-- B. Projection
-- Relational Algebra: π name, tot_cred (ra_student)
-- Task: Retrieve the list of student names and their total credits.
SELECT name, tot_cred
FROM ra_student;

-- C. Union
-- Relational Algebra: ra_cs_student ∪ ra_math_student
-- Task: Combine the lists of Computer Science and Math students.
SELECT id, name, dept_name FROM ra_cs_student
UNION
SELECT id, name, dept_name FROM ra_math_student;

-- D. Set Difference
-- Relational Algebra: ra_student − ra_cs_student
-- Task: Find students who are not in the Computer Science department.
SELECT id, name, dept_name, tot_cred
FROM ra_student
WHERE dept_name <> 'Computer Science' OR dept_name IS NULL;

-- E. Cartesian Product
-- Relational Algebra: ra_instructor × ra_course
-- Task: Pair each instructor with every course.
SELECT *
FROM ra_instructor, ra_course;

-- F. Renaming
-- Relational Algebra: ρ (student_id, student_name, department, total_credits) (ra_student)
-- Task: Rename the attributes of the Student table.
SELECT id        AS student_id,
       name      AS student_name,
       dept_name AS department,
       tot_cred  AS total_credits
FROM ra_student;

