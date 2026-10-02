-- Part 1: Setting Up the Geo-Spatial Database
-- Geoville: a fictional city used to practice MySQL spatial data types and functions.

CREATE DATABASE IF NOT EXISTS geoville;
USE geoville;

DROP TABLE IF EXISTS buildings;
DROP TABLE IF EXISTS roads;
DROP TABLE IF EXISTS parks;

-- Table: Parks
CREATE TABLE parks (
    park_id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100),
    area GEOMETRY NOT NULL,
    SPATIAL INDEX(area)
);

-- Table: Roads
CREATE TABLE roads (
    road_id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100),
    path LINESTRING NOT NULL,
    SPATIAL INDEX(path)
);

-- Table: Buildings
CREATE TABLE buildings (
    building_id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100),
    footprint POLYGON NOT NULL,
    SPATIAL INDEX(footprint)
);
