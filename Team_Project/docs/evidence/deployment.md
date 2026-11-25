# Deployment evidence (BACKEND_PLAN Phase 10, AWS_PLAN.md §4)

Deployed 2026-11-25 to account `869093593722`, region **us-east-2**, from branch
`deploy/phase10-aws` (commands sent with SSM Run Command; no SSH).

**Live site:** https://3-136-227-184.sslip.io

```text
Browser ──HTTPS :443──► EC2 t3.small  (Docker Compose, project book-metaverse-rds)
          (HTTP :80 → 308)   ├─ proxy  caddy:2.11.9-alpine   Let's Encrypt, HSTS
                             ├─ web    Next.js 16 standalone  (not published)
                             └─ api    Go API                 (not published)
                                   │ TLS (sslmode=require), port 5432
                                   ▼
                         RDS PostgreSQL 18.3, private, encrypted
```

## Steps

1. Started the stopped instance and RDS (`aws ec2 start-instances`,
   `aws rds start-db-instance`).
2. Terraform (`infra/live/dev/app-sg`): added inbound 443 to the app security
   group. Plan: 1 to add, 0 to change, 0 to destroy.
3. On the instance (SSM Run Command, root):
   - Added a 2 GB swap file (the `t3.small` has 2 GB of memory; the Next.js
     and Go builds run on the instance).
   - Cloned the repository as `ec2-user`.
   - Wrote `.env` (mode 600) from `.env.example` on the instance: database
     values from the Secrets Manager secret (read with the instance role),
     `DB_SECRET_ARN`, a new random `JWT_SECRET`, `COOKIE_SECURE=true`, and
     `SITE_ADDRESS=3-136-227-184.sslip.io`. No secret left the instance or
     appeared in command output.
   - `./scripts/dev.sh rds`: migrations, seeds, then `api` and `web` built and
     healthy; `/readyz` reported `{"database":"connected","status":"ready"}`.
   - `docker compose --profile https up -d proxy`: Caddy obtained the
     certificate on first start.

## Verification

| Check (AWS_PLAN.md §4.4) | Result |
| --- | --- |
| `/readyz` returns 200 | Yes, `{"database":"connected","status":"ready"}` |
| Schema and seeds on RDS | Migration version 8, not dirty; 201 books, 51 users, 301 reviews, 301 ratings |
| HTTPS | Let's Encrypt certificate for `3-136-227-184.sslip.io` (issuer YE1, valid to 2027-02-23); HTTP returns 308 to HTTPS; `Strict-Transport-Security`, `X-Content-Type-Options`, `Referrer-Policy` headers |
| Session cookie | `bm_session=…; Path=/; HttpOnly; Secure; SameSite=Lax` |
| Only the proxy is reachable | From the internet: 80 and 443 open; 3000, 8080, 5432, 22 closed |
| RDS is private | The endpoint resolves to `172.31.36.195`; not reachable from the internet; its security group allows 5432 only from the app security group |
| App flows on the public address | Register, dashboard, Open Library search and import (*A Wizard of Earthsea*), Reading → page 205 of 205 → Finished, review with 5 stars (summary "★ 5.0 (1 rating), 1 review"), every report tab including recommendations — all passed |
| `docker compose logs` readable | Yes: the API logs one JSON line per request (request ID, route, status, duration) |

Resource use after the test: api 19 MB, web 59 MB, Caddy 21 MB of memory;
images: api 23.5 MB, web 236 MB, Caddy 65 MB.

The `web` log shows a few `Server Reference ID did not match` errors. They are
internet scanners posting fake Next.js Server Action IDs (this app defines no
Server Actions); Next.js rejects them, and 16.4.6 includes the fixes for the
2025 Server Components advisories.

Screenshots: [home](deploy/01-home-https.jpg), [dashboard](deploy/02-dashboard.jpg),
[Open Library search](deploy/03-openlibrary-search.jpg),
[book finished and reviewed](deploy/04-book-finished-reviewed.jpg),
[SQL catalog](deploy/05-reports-sql.jpg), [my reading report](deploy/06-report-my-reading.jpg).

## Operating it

From `Team_Project` on the instance (`aws ssm start-session` or Run Command):

```bash
DC="docker compose --env-file .env -f docker-compose-rdbs.yml -p book-metaverse-rds"
git pull && ./scripts/dev.sh rds --no-seed      # redeploy api and web
$DC --profile https up -d proxy                   # (re)start HTTPS
$DC logs --tail 50 api web proxy
```

To save money between sessions, stop the instance and RDS
(`aws ec2 stop-instances`, `aws rds stop-db-instance`); the containers restart
on boot (`restart: unless-stopped`) once RDS is available again. RDS restarts
by itself after 7 days stopped.

## Hardening (2026-11-25)

| Change | Result |
| --- | --- |
| RDS automated backups (`live/dev/data`) | Retention 0 → 1 day (the AWS free plan refuses more); latest restorable time is reported, and the first automated snapshot exists |
| RDS deletion protection, tags copied to snapshots | On. `terraform destroy` needs `deletion_protection = false` applied first |
| Database restart while turning backups on | About 5 minutes; afterwards the site and `/api/books` return 200 |
| Budget (`live/dev/budget`) | `cmpe180b-team-project-monthly`, $20 a month; email at 80% actual and 100% forecast |
| Old console database `book-metaverse-database` | Deleted with the team's agreement (it was public), with its automated backups |
| VPC default security group | The rule allowing 5432 from `0.0.0.0/0` removed; nothing used the group |
