-- Part 2: Sample data for San Jose (hometown)
USE hometown_geo;

-- Parks are stored as polygons (closed rings) wrapped in GEOMETRY.
-- Coordinates are (longitude, latitude) approximations of real downtown
-- San Jose parks.
INSERT INTO parks (name, area) VALUES
('Plaza de Cesar Chavez',
 ST_GeomFromText('POLYGON((-121.8907 37.3328, -121.8891 37.3328, -121.8891 37.3343, -121.8907 37.3343, -121.8907 37.3328))')),
('Guadalupe River Park',
 ST_GeomFromText('POLYGON((-121.9005 37.3290, -121.8975 37.3290, -121.8975 37.3400, -121.9005 37.3400, -121.9005 37.3290))')),
('Kelley Park',
 ST_GeomFromText('POLYGON((-121.8530 37.3200, -121.8480 37.3200, -121.8480 37.3240, -121.8530 37.3240, -121.8530 37.3200))'));

-- Roads are linestrings tracing the path of travel.
INSERT INTO roads (name, path) VALUES
('The Alameda',
 ST_GeomFromText('LINESTRING(-121.8888 37.3368, -121.8950 37.3400, -121.9020 37.3430, -121.9115 37.3487)')),
('Santa Clara Street',
 ST_GeomFromText('LINESTRING(-121.8820 37.3382, -121.8950 37.3360, -121.9020 37.3345)')),
('Almaden Boulevard',
 ST_GeomFromText('LINESTRING(-121.8912 37.3270, -121.8912 37.3330, -121.8912 37.3400)'));

-- Buildings are polygon footprints.
INSERT INTO buildings (name, footprint) VALUES
('San Jose City Hall',
 ST_GeomFromText('POLYGON((-121.8866 37.3372, -121.8860 37.3372, -121.8860 37.3376, -121.8866 37.3376, -121.8866 37.3372))')),
('SAP Center',
 ST_GeomFromText('POLYGON((-121.9020 37.3323, -121.9008 37.3323, -121.9008 37.3331, -121.9020 37.3331, -121.9020 37.3323))')),
('San Jose Museum of Art',
 ST_GeomFromText('POLYGON((-121.8903 37.3335, -121.8897 37.3335, -121.8897 37.3339, -121.8903 37.3339, -121.8903 37.3335))'));

-- Verify the inserted data.
SELECT park_id, name, ST_AsText(area) AS area_wkt FROM parks;
SELECT road_id, name, ST_AsText(path) AS path_wkt FROM roads;
SELECT building_id, name, ST_AsText(footprint) AS footprint_wkt FROM buildings;
