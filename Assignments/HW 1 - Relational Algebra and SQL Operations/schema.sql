-- HW 1 - Relational Algebra and SQL Operations
-- Part 1.B: Database Schema Design - Table creation (MySQL)

CREATE DATABASE IF NOT EXISTS university;
USE university;

-- Drop in reverse dependency order so re-running this script is safe.
DROP TABLE IF EXISTS advisor;
DROP TABLE IF EXISTS takes;
DROP TABLE IF EXISTS teaches;
DROP TABLE IF EXISTS section;
DROP TABLE IF EXISTS course;
DROP TABLE IF EXISTS instructor;
DROP TABLE IF EXISTS student;
DROP TABLE IF EXISTS classroom;
DROP TABLE IF EXISTS department;

CREATE TABLE department (
    dept_name VARCHAR(20) NOT NULL,
    building  VARCHAR(15),
    budget    DECIMAL(12, 2) CHECK (budget > 0),
    PRIMARY KEY (dept_name)
);

CREATE TABLE classroom (
    building    VARCHAR(15) NOT NULL,
    room_number VARCHAR(7)  NOT NULL,
    capacity    NUMERIC(4, 0),
    PRIMARY KEY (building, room_number)
);

CREATE TABLE student (
    id        VARCHAR(5)  NOT NULL,
    name      VARCHAR(20) NOT NULL,
    dept_name VARCHAR(20),
    tot_cred  NUMERIC(3, 0) CHECK (tot_cred >= 0),
    PRIMARY KEY (id),
    FOREIGN KEY (dept_name) REFERENCES department (dept_name)
        ON DELETE SET NULL
);

CREATE TABLE instructor (
    id        VARCHAR(5)  NOT NULL,
    name      VARCHAR(20) NOT NULL,
    dept_name VARCHAR(20),
    salary    DECIMAL(10, 2) CHECK (salary > 29000),
    PRIMARY KEY (id),
    FOREIGN KEY (dept_name) REFERENCES department (dept_name)
        ON DELETE SET NULL
);

CREATE TABLE course (
    course_id VARCHAR(8)  NOT NULL,
    title     VARCHAR(50),
    dept_name VARCHAR(20),
    credits   NUMERIC(2, 0) CHECK (credits > 0),
    PRIMARY KEY (course_id),
    FOREIGN KEY (dept_name) REFERENCES department (dept_name)
        ON DELETE SET NULL
);

CREATE TABLE section (
    course_id     VARCHAR(8) NOT NULL,
    sec_id        VARCHAR(8) NOT NULL,
    semester      ENUM ('Fall', 'Winter', 'Spring', 'Summer') NOT NULL,
    year          NUMERIC(4, 0) CHECK (year > 1900),
    building      VARCHAR(15),
    room_number   VARCHAR(7),
    time_slot_id  VARCHAR(4),
    PRIMARY KEY (course_id, sec_id, semester, year),
    FOREIGN KEY (course_id) REFERENCES course (course_id)
        ON DELETE CASCADE,
    FOREIGN KEY (building, room_number) REFERENCES classroom (building, room_number)
        ON DELETE SET NULL
);

CREATE TABLE teaches (
    id         VARCHAR(5) NOT NULL,
    course_id  VARCHAR(8) NOT NULL,
    sec_id     VARCHAR(8) NOT NULL,
    semester   ENUM ('Fall', 'Winter', 'Spring', 'Summer') NOT NULL,
    year       NUMERIC(4, 0) NOT NULL,
    PRIMARY KEY (id, course_id, sec_id, semester, year),
    FOREIGN KEY (course_id, sec_id, semester, year)
        REFERENCES section (course_id, sec_id, semester, year)
        ON DELETE CASCADE,
    FOREIGN KEY (id) REFERENCES instructor (id)
        ON DELETE CASCADE
);

CREATE TABLE takes (
    id         VARCHAR(5) NOT NULL,
    course_id  VARCHAR(8) NOT NULL,
    sec_id     VARCHAR(8) NOT NULL,
    semester   ENUM ('Fall', 'Winter', 'Spring', 'Summer') NOT NULL,
    year       NUMERIC(4, 0) NOT NULL,
    grade      VARCHAR(2),
    PRIMARY KEY (id, course_id, sec_id, semester, year),
    FOREIGN KEY (course_id, sec_id, semester, year)
        REFERENCES section (course_id, sec_id, semester, year)
        ON DELETE CASCADE,
    FOREIGN KEY (id) REFERENCES student (id)
        ON DELETE CASCADE
);

CREATE TABLE advisor (
    s_id VARCHAR(5) NOT NULL,
    i_id VARCHAR(5),
    PRIMARY KEY (s_id),
    FOREIGN KEY (s_id) REFERENCES student (id)
        ON DELETE CASCADE,
    FOREIGN KEY (i_id) REFERENCES instructor (id)
        ON DELETE SET NULL
);
