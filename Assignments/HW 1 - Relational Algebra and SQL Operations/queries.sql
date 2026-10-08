-- HW 1 - Relational Algebra and SQL Operations
-- Part 2: Writing SQL Queries (MySQL)
-- Run schema.sql and data.sql first: mysql -u <user> -p university < queries.sql

USE university;

-- ============================================================
-- A. Join Operations and Aggregation
-- ============================================================

-- A1: All students, their total credits, and their department name.
SELECT s.name        AS student_name,
       s.tot_cred    AS total_credits,
       d.dept_name   AS department_name
FROM student s
JOIN department d ON s.dept_name = d.dept_name
ORDER BY s.name;

-- A2: Total number of students in each department.
-- LEFT JOIN so departments with zero students still show a count of 0.
SELECT d.dept_name          AS department_name,
       COUNT(s.id)          AS num_students
FROM department d
LEFT JOIN student s ON s.dept_name = d.dept_name
GROUP BY d.dept_name
ORDER BY d.dept_name;

-- A3: Every instructor and the total number of (distinct) courses they teach.
-- LEFT JOIN so instructors currently teaching nothing show a count of 0.
SELECT i.name                          AS instructor_name,
       COUNT(DISTINCT t.course_id)     AS num_courses_taught
FROM instructor i
LEFT JOIN teaches t ON t.id = i.id
GROUP BY i.id, i.name
ORDER BY i.name;

-- ============================================================
-- B. Subqueries and Nested Queries
-- ============================================================

-- B1: Students who have taken ALL sections ever offered of "Database System Concepts" (CS-347).
-- Classic relational-division pattern: no section of the course exists that the student
-- did NOT take.
SELECT s.name AS student_name
FROM student s
WHERE NOT EXISTS (
    SELECT 1
    FROM section sec
    JOIN course c ON c.course_id = sec.course_id
    WHERE c.title = 'Database System Concepts'
      AND NOT EXISTS (
          SELECT 1
          FROM takes t
          WHERE t.id = s.id
            AND t.course_id = sec.course_id
            AND t.sec_id = sec.sec_id
            AND t.semester = sec.semester
            AND t.year = sec.year
      )
);

-- B2: Instructors who do not teach any course offered by the "Comp. Sci." department.
SELECT i.name AS instructor_name
FROM instructor i
WHERE i.id NOT IN (
    SELECT t.id
    FROM teaches t
    JOIN course c ON c.course_id = t.course_id
    WHERE c.dept_name = 'Comp. Sci.'
);

-- ============================================================
-- C. Complex Filtering and Set Operations
-- ============================================================

-- C1: Students who have taken more than one (distinct) course in the same semester.
SELECT s.name        AS student_name,
       t.semester,
       t.year,
       COUNT(DISTINCT t.course_id) AS num_courses
FROM takes t
JOIN student s ON s.id = t.id
GROUP BY t.id, t.semester, t.year
HAVING COUNT(DISTINCT t.course_id) > 1;

-- C2: Students who have never received a grade lower than 'B' (i.e. every graded
-- course is A+, A, A-, B+, or B). Students with no recorded grades yet are excluded.
SELECT s.name AS student_name
FROM student s
WHERE EXISTS (
    SELECT 1 FROM takes t WHERE t.id = s.id AND t.grade IS NOT NULL
)
AND NOT EXISTS (
    SELECT 1
    FROM takes t
    WHERE t.id = s.id
      AND t.grade IS NOT NULL
      AND t.grade NOT IN ('A+', 'A', 'A-', 'B+', 'B')
);
