# Geo-Spatial Exercise

Objective: practice creating and querying MySQL spatial data types and
functions using real geospatial data for my hometown, **San Jose, CA**, with
`parks`, `roads`, and `buildings` tables.

## Files

- [schema.sql](schema.sql) — Part 1: creates the `hometown_geo` database and
  the `parks` (`GEOMETRY`), `roads` (`LINESTRING`), and `buildings`
  (`POLYGON`) tables, each with a `SPATIAL INDEX`.
- [data.sql](data.sql) — Part 2: real downtown San Jose landmarks (Plaza de
  Cesar Chavez, Guadalupe River Park, Kelley Park, The Alameda, Santa Clara
  Street, Almaden Boulevard, San Jose City Hall, SAP Center, San Jose Museum
  of Art), inserted with `ST_GeomFromText` using approximate real-world
  longitude/latitude coordinates.
- [queries.sql](queries.sql) — Parts 3-10 spatial queries:
  - Part 3 — Querying geo-spatial data (`ST_AsText`, `ST_Contains`)
  - Part 4 — Location functions (`ST_Distance` between points, nearest park)
  - Part 5 — Distance calculations (roads within a distance of a park)
  - Part 6 — Area and perimeter (`ST_Area`, `ST_Length` on `ST_ExteriorRing`
    since MySQL has no built-in `ST_Perimeter`)
  - Part 7 — Intersection and containment (`ST_Intersects`, `ST_Contains`)
  - Part 8 — Buffering (`ST_Buffer`)
  - Part 9 — Analysis functions (`ST_Union`, `ST_Difference`)
  - Part 10 — Relationship functions (`ST_Touches`, `ST_Within`, `ST_Crosses`)
- [REPORT.md](REPORT.md) — findings report documenting the results of each
  query and what they reveal about San Jose's spatial layout.

Note: MySQL does not have a native `ST_Perimeter()` function (unlike
`ST_Area()`); Part 6.2 uses `ST_Length(ST_ExteriorRing(footprint))` as the
equivalent for polygon perimeter. Coordinates are in degrees (longitude,
latitude), so `ST_Distance`/`ST_Area`/`ST_Buffer` values are in degree units,
not meters.

## How to run (MySQL 5.7+)

```bash
mysql -u <user> -p < schema.sql
mysql -u <user> -p < data.sql
mysql -u <user> -p hometown_geo < queries.sql
```

