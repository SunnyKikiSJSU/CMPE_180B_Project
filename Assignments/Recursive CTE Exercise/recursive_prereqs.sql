-- Recursive CTE: find all direct and indirect prerequisites for a course.
-- Run prereq_schema.sql first (uses the same `university` database as HW1).
USE university;

SET @start_course := 'CS-315';

WITH RECURSIVE prereq_chain AS (
    -- Base case: direct prerequisites of the target course.
    SELECT course_id, prereq_id
    FROM prereq
    WHERE course_id = @start_course

    UNION ALL

    -- Recursive step: prerequisites of the prerequisites found so far.
    SELECT p.course_id, p.prereq_id
    FROM prereq p
    JOIN prereq_chain pc ON p.course_id = pc.prereq_id
)
SELECT DISTINCT c.course_id, c.title
FROM prereq_chain pc
JOIN course c ON c.course_id = pc.prereq_id;
