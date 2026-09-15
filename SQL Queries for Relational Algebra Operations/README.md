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

| id | name  | dept_name        | tot_cred |
|----|-------|------------------|----------|
| 1  | John  | Computer Science | 45       |
| 3  | Alice | Computer Science | 78       |

## B. Projection

**Task**: Retrieve the list of student names and their total credits.

Relational Algebra: π name, tot_cred (ra_student)

```sql
SELECT name, tot_cred
FROM ra_student;
```

**Result**

| name  | tot_cred |
|-------|----------|
| John  | 45       |
| Emily | 32       |
| Alice | 78       |
| Bob   | 22       |

## C. Union

**Task**: Combine the lists of Computer Science and Math students.

Relational Algebra: ra_cs_student ∪ ra_math_student

```sql
SELECT id, name, dept_name FROM ra_cs_student
UNION
SELECT id, name, dept_name FROM ra_math_student;
```

**Result**

| id | name  | dept_name        |
|----|-------|------------------|
| 1  | John  | Computer Science |
| 3  | Alice | Computer Science |
| 2  | Emily | Math             |
| 4  | Bob   | Math             |

## D. Set Difference

**Task**: Find students who are not in the Computer Science department.

Relational Algebra: ra_student − ra_cs_student

```sql
SELECT id, name, dept_name, tot_cred
FROM ra_student
WHERE dept_name <> 'Computer Science' OR dept_name IS NULL;
```

**Result**

| id | name  | dept_name | tot_cred |
|----|-------|-----------|----------|
| 2  | Emily | Math      | 32       |
| 4  | Bob   | Math      | 22       |

## E. Cartesian Product

**Task**: Pair each instructor with every course.

Relational Algebra: ra_instructor × ra_course

```sql
SELECT *
FROM ra_instructor, ra_course;
```

**Result**

| id | name  | course_id | title             |
|----|-------|-----------|-------------------|
| 2  | Alice | C101      | Database Systems  |
| 1  | John  | C101      | Database Systems  |
| 2  | Alice | C102      | Algorithms        |
| 1  | John  | C102      | Algorithms        |

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

| student_id | student_name | department        | total_credits |
|------------|--------------|--------------------|----------------|
| 1          | John         | Computer Science   | 45             |
| 2          | Emily        | Math               | 32             |
| 3          | Alice        | Computer Science   | 78             |
| 4          | Bob          | Math               | 22             |
