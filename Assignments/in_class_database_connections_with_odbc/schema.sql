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
    ('S002', 'Jane Smith'),
    ('S003', 'Alice Johnson'),
    ('S004', 'Bob Martinez'),
    ('S005', 'Carol White'),
    ('S006', 'David Brown'),
    ('S007', 'Eva Green'),
    ('S008', 'Frank Miller'),
    ('S009', 'Grace Lee'),
    ('S010', 'Henry Wilson'),
    ('S011', 'Ivy Chen'),
    ('S012', 'Jack Davis'),
    ('S013', 'Karen Scott'),
    ('S014', 'Liam Turner'),
    ('S015', 'Maya Patel');

INSERT IGNORE INTO takes (student_id, course_id, grade) VALUES
    ('S001', 'C101', 'A'),
    ('S001', 'C102', 'B+'),
    ('S002', 'C101', 'B'),
    ('S003', 'C101', 'A-'),
    ('S004', 'C101', 'B+'),
    ('S005', 'C101', 'C+'),
    ('S006', 'C101', 'A'),
    ('S007', 'C101', 'B'),
    ('S008', 'C101', 'A-'),
    ('S009', 'C101', 'B+'),
    ('S010', 'C101', 'A'),
    ('S011', 'C101', 'C'),
    ('S012', 'C101', 'B'),
    ('S013', 'C101', 'A-'),
    ('S014', 'C101', 'B+'),
    ('S015', 'C101', 'A');
