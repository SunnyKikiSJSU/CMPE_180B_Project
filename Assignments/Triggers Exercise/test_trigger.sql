-- Quick manual check for update_tot_cred.
-- Run schema.sql + data.sql from HW1 and trigger.sql first.

USE university;

-- Before: student's current tot_cred.
SELECT id, tot_cred FROM student WHERE id = '00128';

-- Insert a new enrollment for that student in a course/section they haven't
-- taken yet (CS-315, 3 credits, Spring 2026 — exists in section, unused by 00128).
INSERT INTO takes (id, course_id, sec_id, semester, year, grade)
VALUES ('00128', 'CS-315', '1', 'Spring', 2026, NULL);

-- After: tot_cred should now include CS-315's 3 credits.
SELECT id, tot_cred FROM student WHERE id = '00128';
