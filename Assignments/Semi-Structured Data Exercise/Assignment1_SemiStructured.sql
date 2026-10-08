-- Assignment 1: Semi-Structured Data (JSON)
-- Dialect: MySQL 8.0 (JSON column type + JSON_* functions)

CREATE DATABASE IF NOT EXISTS semi_structured_demo;
USE semi_structured_demo;

DROP TABLE IF EXISTS people;

-- 1. `people` table with a JSON column holding a nested document:
--    dept (string), children (array of {first,last}), interests (array of strings)
CREATE TABLE people (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    doc JSON NOT NULL
);

-- 2. Insert at least two rows.
INSERT INTO people (name, doc) VALUES
('Alice Johnson', JSON_OBJECT(
    'dept', 'Engineering',
    'children', JSON_ARRAY(
        JSON_OBJECT('first', 'Emma', 'last', 'Johnson'),
        JSON_OBJECT('first', 'Liam', 'last', 'Johnson')
    ),
    'interests', JSON_ARRAY('databases', 'hiking', 'chess')
)),
('Bob Smith', JSON_OBJECT(
    'dept', 'Marketing',
    'children', JSON_ARRAY(
        JSON_OBJECT('first', 'Noah', 'last', 'Smith')
    ),
    'interests', JSON_ARRAY('photography', 'travel')
));

-- 3a. Extract all children last names for a given id as a JSON array.
SELECT id, JSON_EXTRACT(doc, '$.children[*].last') AS children_last_names
FROM people
WHERE id = 1;

-- 3b. Filter rows where the interests array contains 'databases'.
SELECT id, name
FROM people
WHERE JSON_CONTAINS(doc->'$.interests', '"databases"');

-- 3c. Generate a JSON object per row: {"name": <name>, "dept": <dept>}
SELECT JSON_OBJECT('name', name, 'dept', doc->>'$.dept') AS person_summary
FROM people;
