-- Extends the `university` database (see HW1's schema.sql/data.sql) with a
-- prereq table so the recursive CTE exercise has hierarchical data to walk.
USE university;

CREATE TABLE IF NOT EXISTS prereq (
    course_id  VARCHAR(8) NOT NULL,
    prereq_id  VARCHAR(8) NOT NULL,
    PRIMARY KEY (course_id, prereq_id),
    FOREIGN KEY (course_id) REFERENCES course (course_id) ON DELETE CASCADE,
    FOREIGN KEY (prereq_id) REFERENCES course (course_id) ON DELETE CASCADE
);

-- CS-315 (Robotics) -> CS-190 -> CS-101, and CS-315 -> EE-181 -> PHY-101,
-- so looking up CS-315's prerequisites recurses two levels deep.
INSERT IGNORE INTO prereq (course_id, prereq_id) VALUES
    ('CS-190',  'CS-101'),
    ('CS-315',  'CS-190'),
    ('CS-315',  'EE-181'),
    ('CS-347',  'CS-101'),
    ('EE-181',  'PHY-101');
