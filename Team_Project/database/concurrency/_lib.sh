# shellcheck shell=bash
# Shared helpers for the multi-session scripts. Source it; do not run it.
#
# PSQL is the client command. Default: psql "$DATABASE_URL". With the local
# Docker database, for example:
#   PSQL="docker exec -i bm-pg psql -U book_app -d book_metaverse"
set -euo pipefail

if [[ -z "${PSQL:-}" ]]; then
  : "${DATABASE_URL:?set DATABASE_URL or PSQL}"
  PSQL="psql $DATABASE_URL"
fi

# sql runs statements in a new session and prints only values.
sql() { $PSQL -X -q -At -v ON_ERROR_STOP=1 "$@"; }

OUT=$(mktemp -d)
trap 'rm -rf "$OUT"' EXIT

# session NAME runs stdin as one psql session and saves its output; call it
# with & to run sessions side by side. Progress messages carry the server clock
# (see `ts`). Expected errors are printed by the script itself from psql's
# LAST_ERROR_SQLSTATE and LAST_ERROR_MESSAGE variables, so they stay in order;
# anything on stderr is shown separately.
session() {
  $PSQL -X -q -At -v ON_ERROR_STOP=1 >"$OUT/$1.out" 2>"$OUT/$1.err"
}

# show NAME... prints saved session output after `wait`.
show() {
  for name in "$@"; do
    sed '/^$/d; s/^/['"$name"'] /' "$OUT/$name.out"
    if [[ -s "$OUT/$name.err" ]]; then sed 's/^/['"$name"' stderr] /' "$OUT/$name.err"; fi
  done
}

# failed is a psql snippet that reports the last error with its SQLSTATE.
failed() { echo "SELECT $ts || '  $1: failed with ' || :'LAST_ERROR_SQLSTATE' || ': ' || :'LAST_ERROR_MESSAGE';"; }

# ts is the SQL expression for a millisecond server timestamp.
ts="to_char(clock_timestamp(), 'HH24:MI:SS.MS')"

# ids quotes arguments as a SQL list: ids a b -> 'a','b'
ids() { local IFS=,; local q=(); for v in "$@"; do q+=("'$v'"); done; echo "${q[*]}"; }

# observe saves the sessions that are waiting on a lock right now.
observe() {
  sql >"$OUT/observer.out" 2>"$OUT/observer.err" <<SQL
SELECT $ts || '  waiting: pid ' || pid || ' ' || wait_event_type || '/' || wait_event
       || ' blocked by ' || pg_blocking_pids(pid)::text
       || ' -> ' || left(regexp_replace(query, '\s+', ' ', 'g'), 70)
FROM pg_stat_activity
WHERE datname = current_database() AND wait_event_type = 'Lock';
SQL
}

# Throwaway fixtures: users and a book that nothing else touches.
make_user() {
  sql -c "INSERT INTO users (email, password_hash, display_name)
          VALUES ('conc-' || gen_random_uuid() || '@example.com', 'not-a-login', '$1')
          RETURNING user_id"
}
make_book() {
  sql -c "INSERT INTO books (title, source) VALUES ('$1', 'user') RETURNING book_id"
}
cleanup() {
  rm -rf "$OUT"
  sql -c "DELETE FROM books WHERE book_id IN ($1); DELETE FROM users WHERE user_id IN ($2);" >/dev/null
}

summary_line() {
  sql <<SQL
SELECT 'ratings table: count=' || count(r.*) || ' avg=' || coalesce(round(avg(r.rating_value), 2), 0)
       || ' | summary row: count=' || s.rating_count || ' avg=' || s.average_rating
       || ' reviews=' || s.review_count
FROM book_rating_summaries s LEFT JOIN ratings r ON r.book_id = s.book_id
WHERE s.book_id = '$1'
GROUP BY s.rating_count, s.average_rating, s.review_count;
SQL
}
