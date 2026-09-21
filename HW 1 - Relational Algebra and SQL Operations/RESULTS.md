# HW 1 - Part 2: Query Results

Results captured by running `schema.sql`, `data.sql`, then `queries.sql` against
MySQL 8.0 (Homebrew) with the sample data included in this folder.

## A1: Students, total credits, and department name

| student_name | total_credits | department_name |
|---|---|---|
| Aoi | 60 | Elec. Eng. |
| Bourikas | 98 | Elec. Eng. |
| Brandt | 80 | History |
| Brown | 58 | Comp. Sci. |
| Chavez | 110 | Finance |
| Levy | 46 | Physics |
| Peltier | 56 | Physics |
| Sanchez | 38 | Music |
| Shankar | 32 | Comp. Sci. |
| Snow | 0 | Physics |
| Tanaka | 120 | Biology |
| Williams | 54 | Comp. Sci. |
| Zhang | 102 | Comp. Sci. |

## A2: Total number of students per department

| department_name | num_students |
|---|---|
| Biology | 1 |
| Comp. Sci. | 4 |
| Elec. Eng. | 2 |
| Finance | 1 |
| History | 1 |
| Math | 0 |
| Music | 1 |
| Physics | 3 |

## A3: Instructors and total distinct courses taught

| instructor_name | num_courses_taught |
|---|---|
| Brandt | 2 |
| Califieri | 0 |
| Crick | 2 |
| Einstein | 1 |
| El Said | 1 |
| Gold | 0 |
| Katz | 1 |
| Kim | 1 |
| Mozart | 1 |
| Srinivasan | 1 |
| Wu | 1 |

## B1: Students who have taken all sections of "Database System Concepts" (CS-347)

| student_name |
|---|
| Zhang |
| Williams |

CS-347 has a single section (Fall 2025), so this returns every student who took that section.

## B2: Instructors who do not teach any course in the "Comp. Sci." department

| instructor_name |
|---|
| Wu |
| Mozart |
| Einstein |
| El Said |
| Gold |
| Califieri |
| Crick |
| Kim |

## C1: Students who took more than one course in the same semester

| student_name | semester | year | num_courses |
|---|---|---|---|
| Zhang | Fall | 2025 | 2 |
| Shankar | Spring | 2026 | 2 |
| Williams | Fall | 2025 | 2 |
| Tanaka | Summer | 2025 | 2 |

## C2: Students who have never received a grade lower than 'B'

| student_name |
|---|
| Zhang |
| Brandt |
| Peltier |
| Levy |
| Williams |
| Sanchez |
| Brown |
| Bourikas |
| Tanaka |

Excludes students with zero graded records (e.g. Shankar, Aoi) since "never received
a grade lower than B" is vacuously true but uninformative for someone with no grades yet.
