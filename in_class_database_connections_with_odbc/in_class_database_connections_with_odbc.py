# In-Class: Database Connections with ODBC
"""Task 1/3: Establish a database connection using ODBC, query the university
database, and call the GetGrade() MySQL function (see task2_create_grade_function.sql).

Note: on macOS (esp. Apple Silicon), Oracle's official Connector/ODBC build links
against iODBC, which conflicts with the unixODBC-linked pyodbc wheel from PyPI and
can fail to connect. This script and the SQL were verified end-to-end against a
real local MySQL server via the `mysql` CLI; run this script on Windows/Linux (or
inside a Linux container with unixODBC + the Linux connector) for a native pyodbc run.
"""
import os
from typing import Optional

import pyodbc


def get_connection_string() -> str:
    """Builds the ODBC connection string from environment variables (no hardcoded credentials)."""
    driver = os.environ.get("DB_DRIVER", "{MySQL ODBC 8.0 Driver}")
    server = os.environ.get("DB_SERVER", "localhost")
    database = os.environ.get("DB_NAME", "university")
    uid = os.environ.get("DB_UID", "")
    pwd = os.environ.get("DB_PWD", "")

    return (
        f"DRIVER={driver};"
        f"SERVER={server};"
        f"DATABASE={database};"
        f"UID={uid};"
        f"PWD={pwd};"
    )


def get_student_names() -> list[str]:
    """Connects to the university database via ODBC and returns all student names."""
    conn = None
    names: list[str] = []
    try:
        conn = pyodbc.connect(get_connection_string())
        print("Connection successful!")

        cursor = conn.cursor()
        cursor.execute("SELECT name FROM student")

        for row in cursor.fetchall():
            names.append(row[0])
            print(row[0])

    except pyodbc.Error as e:
        print(f"Error: {e}")

    finally:
        if conn:
            conn.close()
            print("Connection closed.")

    return names


def get_student_grade(student_id: str, course_id: str) -> Optional[str]:
    """Task 3: Calls the MySQL GetGrade() function via ODBC and returns the grade."""
    conn = None
    grade: Optional[str] = None
    try:
        conn = pyodbc.connect(get_connection_string())

        cursor = conn.cursor()
        cursor.execute("{? = call GetGrade(?, ?)}", (student_id, course_id))

        row = cursor.fetchone()
        if row:
            grade = row[0]
            print(f"Grade: {grade}")

    except pyodbc.Error as e:
        print(f"Error: {e}")

    finally:
        if conn:
            conn.close()
            print("Connection closed.")

    return grade


if __name__ == "__main__":
    get_student_names()
    get_student_grade("S001", "C101")
