# SQL Queries for Relational Algebra Operations

Schema and seed data: see [schema.sql](./schema.sql)
Queries: see [queries.sql](./queries.sql)

## A. Selection

**Task**: Retrieve all students from the "Computer Science" department.

Relational Algebra: σ dept_name='Computer Science' (ra_student)

```sql
SELECT *
FROM ra_student
WHERE dept_name = 'Computer Science';
```

**Result**

| id   | name              | dept_name        | tot_cred |
|------|-------------------|------------------|----------|
| S001 | Ahmed Abdelrahman | Computer Science | 90       |
| S002 | Jane Smith        | Computer Science | 75       |

## B. Projection

**Task**: Retrieve the list of student names and their total credits.

Relational Algebra: π name, tot_cred (ra_student)

```sql
SELECT name, tot_cred
FROM ra_student;
```

**Result**

| name              | tot_cred |
|-------------------|----------|
| Ahmed Abdelrahman | 90       |
| Jane Smith        | 75       |
| Alice Johnson     | 60       |
| Bob Martinez      | 45       |
| Carol White       | 30       |

## C. Union

**Task**: Combine the lists of Computer Science and Math students.

Relational Algebra: ra_cs_student ∪ ra_math_student

```sql
SELECT id, name, dept_name FROM ra_cs_student
UNION
SELECT id, name, dept_name FROM ra_math_student;
```

**Result**

| id   | name              | dept_name        |
|------|-------------------|------------------|
| S001 | Ahmed Abdelrahman | Computer Science |
| S002 | Jane Smith        | Computer Science |
| S003 | Alice Johnson     | Math             |
| S004 | Bob Martinez      | Math             |

## D. Set Difference

**Task**: Find students who are not in the Computer Science department.

Relational Algebra: ra_student − ra_cs_student

```sql
SELECT id, name, dept_name, tot_cred
FROM ra_student
WHERE dept_name <> 'Computer Science' OR dept_name IS NULL;
```

**Result**

| id   | name          | dept_name | tot_cred |
|------|---------------|-----------|----------|
| S003 | Alice Johnson | Math      | 60       |
| S004 | Bob Martinez  | Math      | 45       |
| S005 | Carol White   | Physics   | 30       |

## E. Cartesian Product

**Task**: Pair each instructor with every course.

Relational Algebra: ra_instructor × ra_course

```sql
SELECT *
FROM ra_instructor, ra_course;
```

**Result**

| id   | name       | course_id | title             |
|------|------------|-----------|-------------------|
| I002 | Dr. Lee    | C101      | Database Systems  |
| I001 | Dr. Turner | C101      | Database Systems  |
| I002 | Dr. Lee    | C102      | Discrete Math     |
| I001 | Dr. Turner | C102      | Discrete Math     |

## F. Renaming

**Task**: Rename the attributes of the Student table.

Relational Algebra: ρ (student_id, student_name, department, total_credits) (ra_student)

```sql
SELECT id        AS student_id,
       name      AS student_name,
       dept_name AS department,
       tot_cred  AS total_credits
FROM ra_student;
```

**Result**

| student_id | student_name      | department        | total_credits |
|------------|-------------------|--------------------|----------------|
| S001       | Ahmed Abdelrahman | Computer Science   | 90             |
| S002       | Jane Smith        | Computer Science   | 75             |
| S003       | Alice Johnson     | Math               | 60             |
| S004       | Bob Martinez      | Math               | 45             |
| S005       | Carol White       | Physics            | 30             |
