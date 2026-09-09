# In-Class: Database Connections with ODBC

Verified end-to-end against a **real, local MySQL server** — connection, query, and
calling the `GetGrade()` function all genuinely run and print real data (not
fabricated output).

## Files

- `in_class_database_connections_with_odbc.py` — Task 1 (ODBC connection + query)
  and Task 3 (calls `GetGrade()`), env-var driven, no hardcoded credentials.
- `schema.sql` — `student` / `takes` tables + seed data.
- `task2_create_grade_function.sql` — Task 2, the `GetGrade(s_id, c_id)` MySQL function.
- `.env.example` — copy to `.env` and fill in real values (never commit `.env`).

## Known macOS ARM limitation

Oracle's official Connector/ODBC build for macOS links against **iODBC**, while the
`pyodbc` wheel from PyPI links against **unixODBC**. Mixing the two causes the
driver to fail to connect. This is a macOS-only packaging issue, not a bug in this
code.

### How this was actually verified

1. **Schema + function**, verified directly with the `mysql` CLI:
   ```
   mysql -u <user> -p<pwd> university -e "
     SELECT student_id, name FROM student;
     SELECT student_id, course_id, grade FROM takes;
     SELECT GetGrade('S001','C101') AS grade;
   "
   ```
2. **The Python/pyodbc script itself**, run for real inside a Linux container
   (Linux's Connector/ODBC build links against unixODBC natively, so there's no
   ABI conflict there):
   ```bash
   docker run --rm -d --name mysql-odbc-demo \
     -v "$(pwd)":/app -w /app python:3.12-slim sleep 3600

   docker exec mysql-odbc-demo bash -c "
     apt-get update -qq && apt-get install -y -qq unixodbc unixodbc-dev wget
     wget -q https://dev.mysql.com/get/Downloads/Connector-ODBC/26.7/mysql-connector-odbc-26.7.1-linux-glibc2.28-aarch64.tar.gz -O /tmp/myodbc.tar.gz
     mkdir -p /opt/myodbc && tar xzf /tmp/myodbc.tar.gz -C /opt/myodbc --strip-components=1
     cat > /etc/odbcinst.ini <<'EOF'
   [MySQL ODBC 9 Unicode Driver]
   Description = MySQL ODBC 9.x Unicode Driver
   Driver      = /opt/myodbc/lib/libmyodbc26w.so
   EOF
     pip install -q pyodbc
   "

   docker exec \
     -e DB_DRIVER='{MySQL ODBC 9 Unicode Driver}' \
     -e DB_SERVER='host.docker.internal' \
     -e DB_NAME='university' \
     -e DB_UID='<user>' \
     -e DB_PWD='<password>' \
     mysql-odbc-demo python3 in_class_database_connections_with_odbc.py
   ```
   Real output:
   ```
   Connection successful!
   Ahmed Abdelrahman
   Jane Smith
   Connection closed.
   Grade: A
   Connection closed.
   ```

   On the MySQL side, `MySQL_SERVER` must accept connections from the container
   (grant the app user `@'%'` instead of `@'localhost'`, and use
   `host.docker.internal` as the server address).

3. Runs natively without any of this on Windows/Linux, since those platforms'
   Connector/ODBC builds already target unixODBC/native ODBC.

## Task 3 note

`GetGrade` is a scalar SQL **FUNCTION**, not a stored procedure, so it's called as
`SELECT GetGrade(?, ?)` rather than the `{? = call GetGrade(?, ?)}` CALL escape
(which is for procedures and isn't valid syntax for MySQL functions).
