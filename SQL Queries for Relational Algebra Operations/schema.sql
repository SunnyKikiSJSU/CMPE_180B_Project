-- Database schema for SQL Queries for Relational Algebra Operations assignment
-- Tables are prefixed with ra_ to avoid clashing with other tables in the shared "university" DB.

CREATE TABLE IF NOT EXISTS ra_student (
    id        VARCHAR(10) PRIMARY KEY,
    name      VARCHAR(50) NOT NULL,
    dept_name VARCHAR(50),
    tot_cred  INT
);

CREATE TABLE IF NOT EXISTS ra_cs_student (
    id        VARCHAR(10) PRIMARY KEY,
    name      VARCHAR(50) NOT NULL,
    dept_name VARCHAR(50),
    FOREIGN KEY (id) REFERENCES ra_student(id)
);

CREATE TABLE IF NOT EXISTS ra_math_student (
    id        VARCHAR(10) PRIMARY KEY,
    name      VARCHAR(50) NOT NULL,
    dept_name VARCHAR(50),
    FOREIGN KEY (id) REFERENCES ra_student(id)
);

CREATE TABLE IF NOT EXISTS ra_instructor (
    id   VARCHAR(10) PRIMARY KEY,
    name VARCHAR(50) NOT NULL
);

CREATE TABLE IF NOT EXISTS ra_course (
    course_id VARCHAR(10) PRIMARY KEY,
    title     VARCHAR(100) NOT NULL
);

INSERT IGNORE INTO ra_student (id, name, dept_name, tot_cred) VALUES
    ('S001', 'Ahmed Abdelrahman', 'Computer Science', 90),
    ('S002', 'Jane Smith',        'Computer Science', 75),
    ('S003', 'Alice Johnson',     'Math',             60),
    ('S004', 'Bob Martinez',      'Math',             45),
    ('S005', 'Carol White',       'Physics',          30);

INSERT IGNORE INTO ra_cs_student (id, name, dept_name) VALUES
    ('S001', 'Ahmed Abdelrahman', 'Computer Science'),
    ('S002', 'Jane Smith',        'Computer Science');

INSERT IGNORE INTO ra_math_student (id, name, dept_name) VALUES
    ('S003', 'Alice Johnson', 'Math'),
    ('S004', 'Bob Martinez',  'Math');

INSERT IGNORE INTO ra_instructor (id, name) VALUES
    ('I001', 'Dr. Turner'),
    ('I002', 'Dr. Lee');

INSERT IGNORE INTO ra_course (course_id, title) VALUES
    ('C101', 'Database Systems'),
    ('C102', 'Discrete Math');
