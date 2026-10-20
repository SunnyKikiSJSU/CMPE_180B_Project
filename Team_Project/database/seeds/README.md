# Seed Data

Place reproducible development or demonstration data in this directory as
ordered `.sql` files.

The Compose `seed` service applies every `.sql` file in this directory in
lexical order. Seed scripts should be idempotent where practical so the local
database can be refreshed without manual cleanup.
