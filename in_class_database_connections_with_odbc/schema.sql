-- Sample schema and seed data for the in-class ODBC activity's `university` database.
CREATE TABLE IF NOT EXISTS student (
    student_id CHAR(5) PRIMARY KEY,
    name       VARCHAR(100) NOT NULL
);

CREATE TABLE IF NOT EXISTS takes (
    student_id CHAR(5) NOT NULL,
    course_id  CHAR(8) NOT NULL,
    grade      CHAR(2),
    PRIMARY KEY (student_id, course_id),
    FOREIGN KEY (student_id) REFERENCES student(student_id)
);

INSERT IGNORE INTO student (student_id, name) VALUES
    ('S001', 'Ahmed Abdelrahman'),
    ('S002', 'Jane Smith');

INSERT IGNORE INTO takes (student_id, course_id, grade) VALUES
    ('S001', 'C101', 'A'),
    ('S001', 'C102', 'B+'),
    ('S002', 'C101', 'B');
