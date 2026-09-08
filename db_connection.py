"""Task 1: Establish a database connection using ODBC and query the university database."""
import os

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


if __name__ == "__main__":
    get_student_names()
