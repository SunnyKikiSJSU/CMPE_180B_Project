-- San Jose (hometown) spatial queries, following the exercise's Part 2-10 structure.
-- Run after schema.sql and data.sql.
USE hometown_geo;

-- =====================================================================
-- Part 2: Populating the Tables with Sample Data (applied by data.sql)
-- =====================================================================

-- 2.1 Inserting Data into Parks
-- INSERT INTO parks (name, area) VALUES
-- ('Plaza de Cesar Chavez', ST_GeomFromText('POLYGON((-121.8907 37.3328, -121.8891 37.3328, -121.8891 37.3343, -121.8907 37.3343, -121.8907 37.3328))')),
-- ('Guadalupe River Park', ST_GeomFromText('POLYGON((-121.9005 37.3290, -121.8975 37.3290, -121.8975 37.3400, -121.9005 37.3400, -121.9005 37.3290))')),
-- ('Kelley Park', ST_GeomFromText('POLYGON((-121.8530 37.3200, -121.8480 37.3200, -121.8480 37.3240, -121.8530 37.3240, -121.8530 37.3200))'));

-- 2.2 Inserting Data into Roads
-- INSERT INTO roads (name, path) VALUES
-- ('The Alameda', ST_GeomFromText('LINESTRING(-121.8888 37.3368, -121.8950 37.3400, -121.9020 37.3430, -121.9115 37.3487)')),
-- ('Santa Clara Street', ST_GeomFromText('LINESTRING(-121.8820 37.3382, -121.8950 37.3360, -121.9020 37.3345)')),
-- ('Almaden Boulevard', ST_GeomFromText('LINESTRING(-121.8912 37.3270, -121.8912 37.3330, -121.8912 37.3400)'));

-- 2.3 Inserting Data into Buildings
-- INSERT INTO buildings (name, footprint) VALUES
-- ('San Jose City Hall', ST_GeomFromText('POLYGON((-121.8866 37.3372, -121.8860 37.3372, -121.8860 37.3376, -121.8866 37.3376, -121.8866 37.3372))')),
-- ('SAP Center', ST_GeomFromText('POLYGON((-121.9020 37.3323, -121.9008 37.3323, -121.9008 37.3331, -121.9020 37.3331, -121.9020 37.3323))')),
-- ('San Jose Museum of Art', ST_GeomFromText('POLYGON((-121.8903 37.3335, -121.8897 37.3335, -121.8897 37.3339, -121.8903 37.3339, -121.8903 37.3335))'));

-- =====================================================================
-- Part 3: Querying Geo-Spatial Data
-- =====================================================================

-- 3.1 Selecting All Parks
SELECT name, ST_AsText(area) FROM parks;

-- 3.2 Finding Buildings Within a Specific Area (Plaza de Cesar Chavez's footprint)
SELECT name FROM buildings
WHERE ST_Contains(ST_GeomFromText('POLYGON((-121.8907 37.3328, -121.8891 37.3328, -121.8891 37.3343, -121.8907 37.3343, -121.8907 37.3328))'), footprint);

-- =====================================================================
-- Part 4: Using Location Functions
-- =====================================================================

-- 4.1 Coordinate Retrieval and Conversion: centroid of each park
SELECT name, ST_AsText(ST_Centroid(area)) AS centroid
FROM parks;

-- 4.2 Calculating Distance Between Two Points (San Jose City Hall vs SAP Center, in degrees)
SELECT ST_Distance(
    ST_GeomFromText('POINT(-121.8863 37.3374)'),
    ST_GeomFromText('POINT(-121.9014 37.3327)')) AS distance;

-- 4.3 Finding the Nearest Park to a Building
SELECT p.name, ST_Distance(b.footprint, p.area) AS distance
FROM buildings b, parks p
WHERE b.name = 'San Jose City Hall'
ORDER BY distance ASC
LIMIT 1;

-- =====================================================================
-- Part 5: Distance Calculations
-- =====================================================================

-- 5.1 Listing Roads Within a Certain Distance from a Park
-- (Coordinates are in degrees; ~0.005 degrees is roughly 500m at this latitude.)
SELECT r.name
FROM roads r, parks p
WHERE p.name = 'Plaza de Cesar Chavez' AND ST_Distance(r.path, p.area) < 0.005;

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
WHERE ST_Contains(area, ST_GeomFromText('POINT(-121.8900 37.3335)'));

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
    (SELECT area FROM parks WHERE name = 'Plaza de Cesar Chavez'),
    (SELECT area FROM parks WHERE name = 'Guadalupe River Park'))) AS combined_area;

-- 9.2 Finding the Difference Between Two Areas
SELECT ST_AsText(ST_Difference(
    (SELECT area FROM parks WHERE name = 'Plaza de Cesar Chavez'),
    (SELECT area FROM parks WHERE name = 'Guadalupe River Park'))) AS difference_area;

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
WHERE ST_Within(b.footprint, ST_GeomFromText('POLYGON((-121.8907 37.3328, -121.8891 37.3328, -121.8891 37.3343, -121.8907 37.3343, -121.8907 37.3328))'));

-- Crosses
SELECT r.name
FROM roads r, parks p
WHERE ST_Crosses(r.path, p.area);
