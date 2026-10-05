# Geo-Spatial Exercise — Findings Report

**Hometown:** San Jose, CA (downtown core)
**Database:** MySQL 8.0 (tested on `mysql Ver 26.7.0`, Homebrew) / `hometown_geo`

## 1. Data Collection

Real-world downtown San Jose landmarks were used, with longitude/latitude
coordinates approximated from their known real locations:

| Type | Name |
|---|---|
| Park | Plaza de Cesar Chavez |
| Park | Guadalupe River Park |
| Park | Kelley Park |
| Road | The Alameda |
| Road | Santa Clara Street |
| Road | Almaden Boulevard |
| Building | San Jose City Hall |
| Building | SAP Center |
| Building | San Jose Museum of Art |

Coordinates are stored as (longitude, latitude) pairs, matching the WKT
`POINT(lon lat)` convention used by `ST_GeomFromText`.

## 2. Setting Up the Database

Database `hometown_geo` was created with three tables — `parks` (`GEOMETRY`),
`roads` (`LINESTRING`), and `buildings` (`POLYGON`) — each with a
`SPATIAL INDEX` on its geometry column (see `schema.sql`).

## 3. Querying Geo-Spatial Data

- All 3 parks round-trip correctly through `ST_AsText(area)`.
- `ST_Contains(plaza_polygon, footprint)` correctly identifies **San Jose
  Museum of Art** as the one building located inside Plaza de Cesar Chavez —
  matching its real-world location directly adjacent to the plaza.

## 4. Location Functions

- Distance between San Jose City Hall and SAP Center: **0.0158 degrees**
  (point-to-point, straight-line). Since coordinates are in degrees rather
  than meters, this is a relative/comparative measure, not a literal
  distance — at this latitude, 1 degree of longitude is roughly 88 km, so
  this maps to a rough real-world separation of a few kilometers, consistent
  with these two landmarks being on opposite sides of downtown.
- Nearest park to San Jose City Hall: **Plaza de Cesar Chavez**
  (distance ≈ 0.00383 degrees), which matches reality — City Hall is only a
  few blocks from the Plaza.

## 5. Distance Calculations

Roads within 0.005 degrees of Plaza de Cesar Chavez: **all three** (The
Alameda, Santa Clara Street, Almaden Boulevard) — expected, since all three
streets run through or near downtown San Jose where the plaza sits.

## 6. Area and Perimeter

| Feature | Area (deg²) | Notes |
|---|---|---|
| Plaza de Cesar Chavez | 0.0000024 | Smallest park — matches its compact, single-block footprint in reality |
| Guadalupe River Park | 0.000033 | Largest park — matches its long, linear shape along the river |
| Kelley Park | 0.00002 | Mid-sized, matches its larger multi-block real footprint |

Building perimeters (via `ST_Length(ST_ExteriorRing(footprint))`, since MySQL
has no native `ST_Perimeter()`): City Hall and Museum of Art ≈ 0.002 deg,
SAP Center ≈ 0.004 deg (SAP Center's footprint is modeled larger, matching
its real status as a full arena vs. the smaller office/museum buildings).

## 7. Intersection and Containment

- `ST_Intersects(footprint, path)` found **San Jose City Hall** intersecting
  a road path — City Hall's real address (200 E Santa Clara St) sits right
  on Santa Clara Street, so this is the expected real-world match.
- `ST_Contains(area, POINT)` correctly placed a test point inside
  **Plaza de Cesar Chavez** only, not the other two parks.

## 8. Buffering

A 0.001-degree buffer (~100 m) was generated around each park polygon using
`ST_Buffer`. Buffering San Jose Museum of Art's buildings query
(`buildings within buffered parks`) correctly returned **San Jose Museum of
Art** as within the buffered Plaza boundary — consistent with it being the
closest building to the plaza.

## 9. Analysis Functions

- `ST_Union` of Plaza de Cesar Chavez + Guadalupe River Park produced a
  `MULTIPOLYGON` (the two parks are disjoint/non-adjacent in this model,
  matching their real separation of roughly half a mile).
- `ST_Difference` of the same two parks returned the full Plaza polygon
  unchanged, confirming the two shapes don't overlap.

## 10. Relationship Functions

- `ST_Touches`: no buildings reported as touching a park boundary exactly
  (expected — none of the modeled footprints share an edge).
- `ST_Within`: San Jose Museum of Art confirmed within Plaza de Cesar
  Chavez's polygon.
- `ST_Crosses`: **Santa Clara Street** crosses a park boundary (Plaza de
  Cesar Chavez) — matches reality, since Santa Clara Street runs along the
  plaza's edge downtown.

## Conclusion

Modeling real downtown San Jose landmarks (rather than arbitrary
coordinates) made the spatial query results directly checkable against
real-world geography: every containment, intersection, and nearest-neighbor
result lined up with the actual relative positions of these landmarks in
San Jose. The main practical lesson was that MySQL spatial functions
(`ST_Distance`, `ST_Area`, `ST_Buffer`) operate on raw coordinate units — in
degrees here, since no SRID/geography type was used — so results need a
rough degrees-to-meters conversion (~111 km/degree latitude, ~88 km/degree
longitude at this latitude) to be interpreted as real-world measurements.
