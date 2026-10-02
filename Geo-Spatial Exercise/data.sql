-- Part 2: Sample data for Geoville
USE geoville;

-- Parks are stored as polygons (closed rings) wrapped in GEOMETRY.
INSERT INTO parks (name, area) VALUES
('Central Park',
 ST_GeomFromText('POLYGON((2 8, 5 8, 5 11, 2 11, 2 8))')),
('Riverside Park',
 ST_GeomFromText('POLYGON((6 5, 9 5, 9 7, 8 8, 6 7, 6 5))'));

-- Roads are linestrings tracing the path of travel.
INSERT INTO roads (name, path) VALUES
('Main Street',
 ST_GeomFromText('LINESTRING(0 0, 10 10)')),
('Second Avenue',
 ST_GeomFromText('LINESTRING(0 10, 10 0)'));

-- Buildings are polygon footprints.
INSERT INTO buildings (name, footprint) VALUES
('City Hall',
 ST_GeomFromText('POLYGON((4 4, 5 4, 5 5, 4 5, 4 4))')),
('Library',
 ST_GeomFromText('POLYGON((7 7, 8 7, 8 8, 7 8, 7 7))'));

-- Verify the inserted data.
SELECT park_id, name, ST_AsText(area) AS area_wkt FROM parks;
SELECT road_id, name, ST_AsText(path) AS path_wkt FROM roads;
SELECT building_id, name, ST_AsText(footprint) AS footprint_wkt FROM buildings;
