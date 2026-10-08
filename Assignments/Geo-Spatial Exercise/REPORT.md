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

For each type of analysis: the query used (see `queries.sql` for the exact
SQL), the result, and the interpretation.

### 1. Querying Geo-Spatial Data

- **Query:** `SELECT name, ST_AsText(area) FROM parks;` and
  `ST_Contains(plaza_polygon, footprint)` to find buildings inside Plaza de
  Cesar Chavez.
- **Result:** All 3 parks round-trip correctly through `ST_AsText(area)`.
  `ST_Contains` returns exactly one row: **San Jose Museum of Art**.
- **Interpretation:** San Jose Museum of Art is modeled as the one building
  located inside Plaza de Cesar Chavez — matching its real-world location
  directly adjacent to the plaza.

### 2. Location Functions

- **Query:** `ST_Centroid(area)` per park; `ST_Distance` between two points;
  `ST_Distance` + `ORDER BY`/`LIMIT 1` for nearest-park lookup.
- **Result:** Centroid of Plaza de Cesar Chavez → `POINT(-121.8899
  37.33355)`. Distance between San Jose City Hall and SAP Center: **0.0158
  degrees**. Nearest park to San Jose City Hall: **Plaza de Cesar Chavez**
  (distance ≈ 0.00383 degrees).
- **Interpretation:** `ST_Centroid` gives a single representative point
  useful for map pins or proximity sorting without needing the full
  polygon. Since coordinates are in degrees rather than meters, the
  distance values are a relative/comparative measure, not a literal
  distance — at this latitude 1 degree of longitude is roughly 88 km, so
  0.0158 degrees maps to a rough real-world separation of a few kilometers,
  consistent with City Hall and SAP Center being on opposite sides of
  downtown. The nearest-park result matches reality — City Hall is only a
  few blocks from the Plaza.

### 3. Distance Calculations

- **Query:** `ST_Distance(r.path, p.area) < 0.005` for roads vs. Plaza de
  Cesar Chavez.
- **Result:** **All three** roads (The Alameda, Santa Clara Street, Almaden
  Boulevard) are within 0.005 degrees.
- **Interpretation:** Expected, since all three streets run through or near
  downtown San Jose where the plaza sits.

### 4. Area and Perimeter

- **Query:** `ST_Area(area)` per park; `ST_Length(ST_ExteriorRing(footprint))`
  per building (MySQL has no native `ST_Perimeter()`).
- **Result:**

  | Feature | Area (deg²) |
  |---|---|
  | Plaza de Cesar Chavez | 0.0000024 |
  | Guadalupe River Park | 0.000033 |
  | Kelley Park | 0.00002 |

  Building perimeters: City Hall and Museum of Art ≈ 0.002 deg, SAP Center
  ≈ 0.004 deg.
- **Interpretation:** Plaza de Cesar Chavez is smallest, matching its
  compact single-block footprint; Guadalupe River Park is largest, matching
  its long, linear shape along the river; Kelley Park is mid-sized, matching
  its larger multi-block real footprint. SAP Center's larger perimeter
  matches its real status as a full arena vs. the smaller office/museum
  buildings.

### 5. Intersection and Containment

- **Query:** `ST_Intersects(footprint, path)` for buildings vs. roads;
  `ST_Contains(area, POINT)` for a test point vs. parks.
- **Result:** **San Jose City Hall** intersects a road path. The test point
  is contained only by **Plaza de Cesar Chavez**.
- **Interpretation:** City Hall's real address (200 E Santa Clara St) sits
  right on Santa Clara Street, so the intersection result is the expected
  real-world match. The containment result correctly excludes the other two
  parks.

### 6. Buffering

- **Query:** `ST_Buffer(area, 0.001)` per park, then `ST_Contains` of the
  buffered polygon against building footprints.
- **Result:** San Jose Museum of Art is the only building within the
  buffered Plaza de Cesar Chavez boundary.
- **Interpretation:** A 0.001-degree buffer is approximately 88–111 meters
  in San Jose, depending on direction; since the data uses SRID 0 and degree
  coordinates, this is only an approximation. The result is consistent with
  the Museum of Art being the closest building to the plaza.

### 7. Analysis Functions

- **Query:** `ST_Union` and `ST_Difference` on Plaza de Cesar Chavez +
  Guadalupe River Park.
- **Result:** `ST_Union` produced a `MULTIPOLYGON`. `ST_Difference` returned
  the full Plaza polygon unchanged.
- **Interpretation:** The two parks are disjoint/non-adjacent in this model
  (matching their real separation of roughly half a mile), so the union
  can't merge into one polygon and the difference confirms no overlap.

### 8. Relationship Functions

- **Query:** `ST_Touches`, `ST_Within`, `ST_Crosses` across
  buildings/parks/roads.
- **Result:** `ST_Touches` returns no rows. `ST_Within` confirms San Jose
  Museum of Art is within Plaza de Cesar Chavez. `ST_Crosses` returns Santa
  Clara Street against the Plaza boundary.
- **Interpretation:** No modeled footprints share an edge, so `ST_Touches`
  is empty as expected. For the `ST_Crosses` result: because the modeled
  road follows the plaza edge, `ST_Intersects` or `ST_Touches` is more
  semantically appropriate than `ST_Crosses` to describe this relationship —
  `ST_Crosses` technically matched here only because the modeled linestring
  clips through the polygon boundary rather than running cleanly alongside
  it.

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

