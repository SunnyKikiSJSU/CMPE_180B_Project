"""Extra credit: render the hometown_geo parks/roads/buildings on a real
San Jose basemap, using GeoJSON pulled straight from MySQL via ST_AsGeoJSON.

Usage:
    source .venv/bin/activate
    pip install -r requirements.txt
    python3 visualize.py  # writes map.html next to this script

Connection settings are read from env vars (with sane localhost defaults)
so no credentials are hardcoded:
    MYSQL_HOST (default: 127.0.0.1)
    MYSQL_PORT (default: 3306)
    MYSQL_USER (default: root)
    MYSQL_PASSWORD (default: "")
    MYSQL_DATABASE (default: hometown_geo)
"""

import json
import os

import folium
import mysql.connector

# Downtown San Jose, roughly centered on Plaza de Cesar Chavez.
MAP_CENTER = (37.3335, -121.8897)

TABLES = [
    ("parks", "park_id", "area", "green"),
    ("roads", "road_id", "path", "blue"),
    ("buildings", "building_id", "footprint", "red"),
]


def connect():
    return mysql.connector.connect(
        host=os.environ.get("MYSQL_HOST", "127.0.0.1"),
        port=int(os.environ.get("MYSQL_PORT", "3306")),
        user=os.environ.get("MYSQL_USER", "root"),
        password=os.environ.get("MYSQL_PASSWORD", ""),
        database=os.environ.get("MYSQL_DATABASE", "hometown_geo"),
    )


def add_layer(fmap, cursor, table, id_col, geom_col, color):
    cursor.execute(f"SELECT name, ST_AsGeoJSON({geom_col}) FROM {table}")
    for name, geojson_text in cursor.fetchall():
        geometry = json.loads(geojson_text)
        folium.GeoJson(
            {"type": "Feature", "properties": {"name": name}, "geometry": geometry},
            name=f"{table}: {name}",
            style_function=lambda _f, color=color: {
                "color": color,
                "weight": 3,
                "fillColor": color,
                "fillOpacity": 0.25,
            },
            tooltip=name,
        ).add_to(fmap)


def main():
    fmap = folium.Map(location=MAP_CENTER, zoom_start=15, tiles="OpenStreetMap")
    conn = connect()
    try:
        cursor = conn.cursor()
        for table, id_col, geom_col, color in TABLES:
            add_layer(fmap, cursor, table, id_col, geom_col, color)
    finally:
        conn.close()
    folium.LayerControl().add_to(fmap)
    out_path = os.path.join(os.path.dirname(__file__), "map.html")
    fmap.save(out_path)
    print(f"Wrote {out_path}")


if __name__ == "__main__":
    main()
