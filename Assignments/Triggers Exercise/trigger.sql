-- Triggers Exercise
-- Objective: maintain data integrity by auto-updating student.tot_cred
-- whenever a new enrollment row is inserted into takes.
-- Uses the same `university` database as HW 1 (schema.sql/data.sql in
-- "../HW 1 - Relational Algebra and SQL Operations").

USE university;

DELIMITER $$

CREATE TRIGGER update_tot_cred
AFTER INSERT ON takes
FOR EACH ROW
BEGIN
    UPDATE student
    SET tot_cred = (
        SELECT COALESCE(SUM(c.credits), 0)
        FROM takes t
        JOIN course c ON c.course_id = t.course_id
        WHERE t.id = NEW.id
    )
    WHERE id = NEW.id;
END$$

DELIMITER ;
