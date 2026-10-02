-- Geoville spatial queries, following the exercise's Part 2-10 structure.
-- Run after schema.sql and data.sql.
USE geoville;

-- =====================================================================
-- Part 2: Populating the Tables with Sample Data (applied by data.sql)
-- =====================================================================

-- 2.1 Inserting Data into Parks
-- INSERT INTO parks (name, area) VALUES
-- ('Central Park', ST_GeomFromText('POLYGON((2 8, 5 8, 5 11, 2 11, 2 8))')),
-- ('Riverside Park', ST_GeomFromText('POLYGON((6 5, 9 5, 9 7, 8 8, 6 7, 6 5))'));

-- 2.2 Inserting Data into Roads
-- INSERT INTO roads (name, path) VALUES
-- ('Main Street', ST_GeomFromText('LINESTRING(0 0, 10 10)')),
-- ('Second Avenue', ST_GeomFromText('LINESTRING(0 10, 10 0)'));

-- 2.3 Inserting Data into Buildings
-- INSERT INTO buildings (name, footprint) VALUES
-- ('City Hall', ST_GeomFromText('POLYGON((4 4, 5 4, 5 5, 4 5, 4 4))')),
-- ('Library', ST_GeomFromText('POLYGON((7 7, 8 7, 8 8, 7 8, 7 7))'));

-- =====================================================================
-- Part 3: Querying Geo-Spatial Data
-- =====================================================================

-- 3.1 Selecting All Parks
SELECT name, ST_AsText(area) FROM parks;

-- 3.2 Finding Buildings Within a Specific Area (Central Park's footprint)
SELECT name FROM buildings
WHERE ST_Contains(ST_GeomFromText('POLYGON((2 8, 5 8, 5 11, 2 11, 2 8))'), footprint);

-- =====================================================================
-- Part 4: Using Location Functions
-- =====================================================================

-- 4.1 Calculating Distance Between Two Points
SELECT ST_Distance(
    ST_GeomFromText('POINT(0 0)'),
    ST_GeomFromText('POINT(3 4)')) AS distance;

-- 4.2 Finding the Nearest Park to a Building
SELECT p.name, ST_Distance(b.footprint, p.area) AS distance
FROM buildings b, parks p
WHERE b.name = 'City Hall'
ORDER BY distance ASC
LIMIT 1;

-- =====================================================================
-- Part 5: Distance Calculations
-- =====================================================================

-- 5.1 Listing Roads Within a Certain Distance from a Park
-- (Assuming coordinates are in degrees, and distance is in degrees.)
SELECT r.name
FROM roads r, parks p
WHERE p.name = 'Central Park' AND ST_Distance(r.path, p.area) < 0.005;

-- =====================================================================
-- Part 6: Area and Perimeter Calculations
-- =====================================================================

-- 6.1 Calculating the Area of Each Park
SELECT name, ST_Area(area) AS area
FROM parks;

-- 6.2 Calculating the Perimeter of Each Building
-- MySQL has no built-in ST_Perimeter(); derive it from the exterior ring's
-- length instead.
SELECT name, ST_Length(ST_ExteriorRing(footprint)) AS perimeter
FROM buildings;

-- =====================================================================
-- Part 7: Intersection and Containment
-- =====================================================================

-- 7.1 Finding Buildings that Intersect with Roads
SELECT b.name
FROM buildings b, roads r
WHERE ST_Intersects(b.footprint, r.path);

-- 7.2 Checking if a Point is Within a Park
SELECT name
FROM parks
WHERE ST_Contains(area, ST_GeomFromText('POINT(3 9)'));

-- =====================================================================
-- Part 8: Buffering
-- =====================================================================

-- 8.1 Creating Buffers Around Parks
SELECT name, ST_AsText(ST_Buffer(area, 0.001)) AS buffer_area
FROM parks;

-- 8.2 Finding Buildings Within Buffered Areas of Parks
SELECT b.name
FROM buildings b, parks p
WHERE ST_Contains(ST_Buffer(p.area, 0.001), b.footprint);

-- =====================================================================
-- Part 9: Analysis Functions
-- =====================================================================

-- 9.1 Computing the Union of Two Parks
SELECT ST_AsText(ST_Union(
    (SELECT area FROM parks WHERE name = 'Central Park'),
    (SELECT area FROM parks WHERE name = 'Riverside Park'))) AS combined_area;

-- 9.2 Finding the Difference Between Two Areas
SELECT ST_AsText(ST_Difference(
    (SELECT area FROM parks WHERE name = 'Central Park'),
    (SELECT area FROM parks WHERE name = 'Riverside Park'))) AS difference_area;

-- =====================================================================
-- Part 10: Relationship Functions
-- =====================================================================

-- 10.1 Checking Spatial Relationships

-- Touches
SELECT b.name
FROM buildings b, parks p
WHERE ST_Touches(b.footprint, p.area);

-- Within
SELECT b.name
FROM buildings b
WHERE ST_Within(b.footprint, ST_GeomFromText('POLYGON((2 8, 5 8, 5 11, 2 11, 2 8))'));

-- Crosses
SELECT r.name
FROM roads r, parks p
WHERE ST_Crosses(r.path, p.area);
