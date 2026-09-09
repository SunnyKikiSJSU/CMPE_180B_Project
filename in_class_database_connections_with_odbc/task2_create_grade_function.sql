-- Task 2: Create a Simple MySQL Function
-- Returns a student's grade for a specific course from the `takes` table.
-- Note: parameters are named s_id/c_id (not student_id/course_id) to avoid
-- ambiguity with the `takes` table's own student_id/course_id columns.
DELIMITER $$

CREATE FUNCTION GetGrade(s_id CHAR(5), c_id CHAR(8))
RETURNS CHAR(2)
DETERMINISTIC
READS SQL DATA
BEGIN
    DECLARE student_grade CHAR(2);

    SELECT grade INTO student_grade
    FROM takes
    WHERE student_id = s_id AND course_id = c_id
    LIMIT 1;

    RETURN student_grade;
END$$

DELIMITER ;
