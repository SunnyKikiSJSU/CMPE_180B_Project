# Geo-Spatial Exercise — Findings Report

**Hometown:** San Jose, CA (downtown core)
**Database:** MySQL Server 8.0, tested using the Homebrew MySQL client / `hometown_geo`

## Introduction

San Jose, CA is my hometown and the largest city in the South Bay/Silicon
Valley. For this exercise I modeled civic and recreational landmarks
around San Jose: three San Jose parks selected for spatial analysis (Plaza
de Cesar Chavez, Guadalupe River Park, Kelley Park — note that Kelley Park
is not part of the downtown core, unlike the other two), three major
downtown streets (The Alameda, Santa Clara Street, Almaden Boulevard), and
three prominent buildings (San Jose City Hall, SAP Center, San Jose Museum
of Art). These were chosen because several are geographically clustered in
and around downtown, which makes spatial relationships between them
(containment, intersection, proximity) meaningful and checkable against
real-world geography, rather than arbitrary/disconnected points.

## Data Collection

**How/where obtained:** Coordinates (longitude, latitude) were derived from
each landmark's known real-world location in downtown San Jose (e.g. San
Jose City Hall at 200 E Santa Clara St, SAP Center at 525 W Santa Clara St,
Plaza de Cesar Chavez between Market St and Almaden Blvd). Parks and
buildings are represented as small bounding-box polygons around their real
footprint; roads are represented as multi-point linestrings tracing their
real path through downtown.

**Data formats:** All geometry is stored as WKT (Well-Known Text) and
inserted via `ST_GeomFromText`, the standard MySQL spatial input format.
Three geometry types are used: `POLYGON` for parks/buildings, `LINESTRING`
for roads, and `POINT` for ad hoc test locations in queries.

**Preprocessing:** Landmark locations were cross-checked using OpenStreetMap
and official San Jose city location information. The geometries were then
simplified into approximate WKT polygons and linestrings for this exercise
(4 decimal places, ~11m resolution) rather than imported wholesale from a
shapefile/GeoJSON source — so each polygon is an approximate bounding box
around the real footprint, not an exact real-world boundary. No SRID/
geography type was used (SRID 0), so all spatial functions operate on plain
Cartesian degree coordinates (see the Database Design and Spatial Queries
notes below on what this means for distance/area units).

Because the polygons are approximate bounding boxes rather than exact
footprints, conclusions like "the Museum of Art is inside the plaza" are
conclusions about this simplified model, not verified real-world survey
facts.

## Database Design

Three tables, each with one geometry column backed by a `SPATIAL INDEX`
(see `schema.sql`):

| Table | Geometry column | Type | Why |
|---|---|---|---|
| `parks` | `area` | `GEOMETRY` | Declared generically (not `POLYGON`) since a future park could be modeled as a `MULTIPOLYGON` (e.g. a park split by a road) without a schema change |
| `roads` | `path` | `LINESTRING` | Roads are inherently linear features, not areas |
| `buildings` | `footprint` | `POLYGON` | Building footprints are always a single closed ring |

Each geometry column is `NOT NULL` (every row must have a location — there's
no use case for a park/road/building without one) and has a `SPATIAL INDEX`
so relationship queries like `ST_Contains`/`ST_Intersects`/`ST_Within` can use
a spatial R-tree index lookup instead of a full table scan as the tables grow.
Note this doesn't automatically speed up every query — e.g. a plain
`ST_Distance` between two arbitrary geometries still has to evaluate both
argument geometries directly and isn't accelerated by the index the same way
bounding-box relationship checks are.

## Spatial Queries and Results

### 1. Querying Geo-Spatial Data

- All 3 parks round-trip correctly through `ST_AsText(area)`.
- `ST_Contains(plaza_polygon, footprint)` correctly identifies **San Jose
  Museum of Art** as the one building located inside Plaza de Cesar Chavez —
  matching its real-world location directly adjacent to the plaza.

### 2. Location Functions

- `ST_Centroid` (coordinate retrieval/conversion) on each park's polygon
  returns its geometric center point, e.g. Plaza de Cesar Chavez →
  `POINT(-121.8899 37.33355)` — a single representative point useful for
  map pins or proximity sorting without needing the full polygon.
- Distance between San Jose City Hall and SAP Center: **0.0158 degrees**
  (point-to-point, straight-line). Since coordinates are in degrees rather
  than meters, this is a relative/comparative measure, not a literal
  distance — at this latitude, 1 degree of longitude is roughly 88 km, so
  this maps to a rough real-world separation of a few kilometers, consistent
  with these two landmarks being on opposite sides of downtown.
- Nearest park to San Jose City Hall: **Plaza de Cesar Chavez**
  (distance ≈ 0.00383 degrees), which matches reality — City Hall is only a
  few blocks from the Plaza.

### 3. Distance Calculations

Roads within 0.005 degrees of Plaza de Cesar Chavez: **all three** (The
Alameda, Santa Clara Street, Almaden Boulevard) — expected, since all three
streets run through or near downtown San Jose where the plaza sits.

### 4. Area and Perimeter

| Feature | Area (deg²) | Notes |
|---|---|---|
| Plaza de Cesar Chavez | 0.0000024 | Smallest park — matches its compact, single-block footprint in reality |
| Guadalupe River Park | 0.000033 | Largest park — matches its long, linear shape along the river |
| Kelley Park | 0.00002 | Mid-sized, matches its larger multi-block real footprint |

Building perimeters (via `ST_Length(ST_ExteriorRing(footprint))`, since MySQL
has no native `ST_Perimeter()`): City Hall and Museum of Art ≈ 0.002 deg,
SAP Center ≈ 0.004 deg (SAP Center's footprint is modeled larger, matching
its real status as a full arena vs. the smaller office/museum buildings).

### 5. Intersection and Containment

- `ST_Intersects(footprint, path)` found **San Jose City Hall** intersecting
  a road path — City Hall's real address (200 E Santa Clara St) sits right
  on Santa Clara Street, so this is the expected real-world match.
- `ST_Contains(area, POINT)` correctly placed a test point inside
  **Plaza de Cesar Chavez** only, not the other two parks.

### 6. Buffering

A 0.001-degree buffer was generated around each park polygon using
`ST_Buffer`. A 0.001-degree buffer is approximately 88–111 meters in San
Jose, depending on direction. Since the data uses SRID 0 and degree
coordinates, this is only an approximation. Buffering San Jose Museum of
Art's buildings query (`buildings within buffered parks`) correctly
returned **San Jose Museum of Art** as within the buffered Plaza
boundary — consistent with it being the closest building to the plaza.

### 7. Analysis Functions

- `ST_Union` of Plaza de Cesar Chavez + Guadalupe River Park produced a
  `MULTIPOLYGON` (the two parks are disjoint/non-adjacent in this model,
  matching their real separation of roughly half a mile).
- `ST_Difference` of the same two parks returned the full Plaza polygon
  unchanged, confirming the two shapes don't overlap.

### 8. Relationship Functions

- `ST_Touches`: no buildings reported as touching a park boundary exactly
  (expected — none of the modeled footprints share an edge).
- `ST_Within`: San Jose Museum of Art confirmed within Plaza de Cesar
  Chavez's polygon.
- `ST_Crosses`: Santa Clara Street intersects the boundary of Plaza de
  Cesar Chavez. Because the modeled road follows the plaza edge,
  `ST_Intersects` or `ST_Touches` is more appropriate than `ST_Crosses` to
  describe this relationship — `ST_Crosses` technically matched here only
  because the modeled linestring clips through the polygon boundary rather
  than running cleanly alongside it.

## Extra Credit: Visualizations

Not completed in this pass — no map/diagram was generated. A follow-up could
plot these polygons/linestrings on a real San Jose basemap (e.g. via QGIS or
a Python `folium`/`geopandas` script reading `ST_AsGeoJSON()` output from
each table) to visually confirm the containment/intersection results above.

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

