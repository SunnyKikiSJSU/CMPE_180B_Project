-- HW 1 - Relational Algebra and SQL Operations
-- Part 1.C: Populate the database with sample data (MySQL)

USE university;

-- Departments
INSERT INTO department (dept_name, building, budget) VALUES
    ('Comp. Sci.', 'Taylor',   1100000.00),
    ('Physics',    'Watson',    850000.00),
    ('Math',       'Chandler',  700000.00),
    ('Biology',    'Watson',    950000.00),
    ('History',    'Painter',   400000.00),
    ('Music',      'Packard',   350000.00),
    ('Finance',    'Painter',   600000.00),
    ('Elec. Eng.', 'Taylor',    850000.00);

-- Classrooms
INSERT INTO classroom (building, room_number, capacity) VALUES
    ('Taylor',   '3128', 70),
    ('Watson',   '100',  120),
    ('Watson',   '120',  30),
    ('Chandler', '101',  50),
    ('Painter',  '514',  10),
    ('Packard',  '101',  500);

-- Students
INSERT INTO student (id, name, dept_name, tot_cred) VALUES
    ('00128', 'Zhang',   'Comp. Sci.', 102),
    ('12345', 'Shankar', 'Comp. Sci.', 32),
    ('19991', 'Brandt',  'History',    80),
    ('23121', 'Chavez',  'Finance',    110),
    ('44553', 'Peltier', 'Physics',    56),
    ('45678', 'Levy',    'Physics',    46),
    ('54321', 'Williams','Comp. Sci.', 54),
    ('55739', 'Sanchez', 'Music',      38),
    ('70557', 'Snow',    'Physics',    0),
    ('76543', 'Brown',   'Comp. Sci.', 58),
    ('76653', 'Aoi',     'Elec. Eng.', 60),
    ('98765', 'Bourikas','Elec. Eng.', 98),
    ('98988', 'Tanaka',  'Biology',    120);

-- Instructors
INSERT INTO instructor (id, name, dept_name, salary) VALUES
    ('10101', 'Srinivasan', 'Comp. Sci.', 65000.00),
    ('12121', 'Wu',         'Finance',    90000.00),
    ('15151', 'Mozart',     'Music',      40000.00),
    ('22222', 'Einstein',   'Physics',    95000.00),
    ('32343', 'El Said',    'History',    80000.00),
    ('33456', 'Gold',       'Physics',    87000.00),
    ('45565', 'Katz',       'Comp. Sci.', 75000.00),
    ('58583', 'Califieri',  'History',    62000.00),
    ('76766', 'Crick',      'Biology',    72000.00),
    ('83821', 'Brandt',     'Comp. Sci.', 92000.00),
    ('98345', 'Kim',        'Elec. Eng.', 80000.00);

-- Courses
INSERT INTO course (course_id, title, dept_name, credits) VALUES
    ('BIO-101', 'Intro. to Biology',        'Biology',    4),
    ('BIO-301', 'Genetics',                 'Biology',    4),
    ('CS-101',  'Intro. to Computer Science','Comp. Sci.',4),
    ('CS-190',  'Game Design',              'Comp. Sci.', 4),
    ('CS-315',  'Robotics',                 'Comp. Sci.', 3),
    ('CS-347',  'Database System Concepts', 'Comp. Sci.', 3),
    ('EE-181',  'Intro. to Digital Systems','Elec. Eng.', 3),
    ('FIN-201', 'Investment Banking',       'Finance',    3),
    ('HIS-351', 'World History',            'History',    3),
    ('MU-199',  'Music Video Production',   'Music',      3),
    ('PHY-101', 'Physical Principles',      'Physics',    4);

-- Sections
INSERT INTO section (course_id, sec_id, semester, year, building, room_number, time_slot_id) VALUES
    ('BIO-101', '1', 'Summer', 2025, 'Watson',  '100', 'H'),
    ('BIO-301', '1', 'Summer', 2025, 'Watson',  '120', 'F'),
    ('CS-101',  '1', 'Fall',   2025, 'Taylor',  '3128','A'),
    ('CS-101',  '1', 'Spring', 2026, 'Taylor',  '3128','A'),
    ('CS-190',  '1', 'Spring', 2026, 'Taylor',  '3128','C'),
    ('CS-315',  '1', 'Spring', 2026, 'Watson',  '100', 'B'),
    ('CS-347',  '1', 'Fall',   2025, 'Taylor',  '3128','A'),
    ('EE-181',  '1', 'Spring', 2026, 'Taylor',  '3128','C'),
    ('FIN-201', '1', 'Spring', 2026, 'Packard', '101', 'B'),
    ('HIS-351', '1', 'Spring', 2026, 'Painter', '514', 'C'),
    ('MU-199',  '1', 'Spring', 2026, 'Packard', '101', 'D'),
    ('PHY-101', '1', 'Fall',   2025, 'Watson',  '100', 'A');

-- Teaches
INSERT INTO teaches (id, course_id, sec_id, semester, year) VALUES
    ('10101', 'CS-347', '1', 'Fall',   2025),
    ('45565', 'CS-101', '1', 'Fall',   2025),
    ('45565', 'CS-101', '1', 'Spring', 2026),
    ('83821', 'CS-190', '1', 'Spring', 2026),
    ('83821', 'CS-315', '1', 'Spring', 2026),
    ('98345', 'EE-181', '1', 'Spring', 2026),
    ('76766', 'BIO-101','1', 'Summer', 2025),
    ('76766', 'BIO-301','1', 'Summer', 2025),
    ('22222', 'PHY-101','1', 'Fall',   2025),
    ('12121', 'FIN-201','1', 'Spring', 2026),
    ('32343', 'HIS-351','1', 'Spring', 2026),
    ('15151', 'MU-199', '1', 'Spring', 2026);

-- Takes
INSERT INTO takes (id, course_id, sec_id, semester, year, grade) VALUES
    ('00128', 'CS-101', '1', 'Fall',   2025, 'A'),
    ('00128', 'CS-347', '1', 'Fall',   2025, 'A-'),
    ('12345', 'CS-101', '1', 'Spring', 2026, NULL),
    ('12345', 'CS-190', '1', 'Spring', 2026, NULL),
    ('19991', 'HIS-351','1', 'Spring', 2026, 'B'),
    ('23121', 'FIN-201','1', 'Spring', 2026, 'C+'),
    ('44553', 'PHY-101','1', 'Fall',   2025, 'B+'),
    ('45678', 'PHY-101','1', 'Fall',   2025, 'A'),
    ('54321', 'CS-101', '1', 'Fall',   2025, 'A-'),
    ('54321', 'CS-347', '1', 'Fall',   2025, 'A'),
    ('55739', 'MU-199', '1', 'Spring', 2026, 'A'),
    ('76543', 'CS-101', '1', 'Fall',   2025, 'B'),
    ('76653', 'EE-181', '1', 'Spring', 2026, NULL),
    ('98765', 'EE-181', '1', 'Spring', 2026, 'A-'),
    ('98988', 'BIO-101','1', 'Summer', 2025, 'A'),
    ('98988', 'BIO-301','1', 'Summer', 2025, 'B+');

-- Advisors
INSERT INTO advisor (s_id, i_id) VALUES
    ('00128', '45565'),
    ('12345', '10101'),
    ('19991', '32343'),
    ('23121', '12121'),
    ('44553', '22222'),
    ('45678', '33456'),
    ('54321', '83821'),
    ('55739', '15151'),
    ('76653', '98345'),
    ('98765', '98345'),
    ('98988', '76766');
