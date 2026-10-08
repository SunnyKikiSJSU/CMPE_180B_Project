-- Database schema for SQL Queries for Relational Algebra Operations assignment
-- Tables are prefixed with ra_ to avoid clashing with other tables in the shared "university" DB.

CREATE TABLE IF NOT EXISTS ra_student (
    id        INT PRIMARY KEY,
    name      VARCHAR(50) NOT NULL,
    dept_name VARCHAR(50),
    tot_cred  INT
);

CREATE TABLE IF NOT EXISTS ra_cs_student (
    id        INT PRIMARY KEY,
    name      VARCHAR(50) NOT NULL,
    dept_name VARCHAR(50),
    FOREIGN KEY (id) REFERENCES ra_student(id)
);

CREATE TABLE IF NOT EXISTS ra_math_student (
    id        INT PRIMARY KEY,
    name      VARCHAR(50) NOT NULL,
    dept_name VARCHAR(50),
    FOREIGN KEY (id) REFERENCES ra_student(id)
);

CREATE TABLE IF NOT EXISTS ra_instructor (
    id   INT PRIMARY KEY,
    name VARCHAR(50) NOT NULL
);

CREATE TABLE IF NOT EXISTS ra_course (
    course_id VARCHAR(10) PRIMARY KEY,
    title     VARCHAR(100) NOT NULL
);

INSERT IGNORE INTO ra_student (id, name, dept_name, tot_cred) VALUES
    (1, 'John',  'Computer Science', 45),
    (2, 'Emily', 'Math',             32),
    (3, 'Alice', 'Computer Science', 78),
    (4, 'Bob',   'Math',             22);

INSERT IGNORE INTO ra_cs_student (id, name, dept_name) VALUES
    (1, 'John',  'Computer Science'),
    (3, 'Alice', 'Computer Science');

INSERT IGNORE INTO ra_math_student (id, name, dept_name) VALUES
    (2, 'Emily', 'Math'),
    (4, 'Bob',   'Math');

INSERT IGNORE INTO ra_instructor (id, name) VALUES
    (1, 'John'),
    (2, 'Alice');

INSERT IGNORE INTO ra_course (course_id, title) VALUES
    ('C101', 'Database Systems'),
    ('C102', 'Algorithms');
