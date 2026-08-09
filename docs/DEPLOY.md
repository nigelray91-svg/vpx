# Deploying to your server

Every command here runs **on the server**, over SSH — not on your laptop and not
in your DNS panel. Do the DNS records first ([`DNS.md`](DNS.md)); certificates
cannot be issued until `panel` and `api` resolve.

## 1. Connect

```bash
ssh root@YOUR_SERVER_IP
```

## 2. Install Docker (skip if already installed)

`docker --version` tells you. On Ubuntu/Debian:

```bash
curl -fsSL https://get.docker.com | sh
```

## 3. Get the code

```bash
apt-get update && apt-get install -y git
git clone https://github.com/nigelray91-svg/vpx.git
cd vpx
git checkout claude/whitelabel-proxy-reseller-aw35l8
```

From here on, **stay in this `vpx` directory** — every command below is relative
to it, and `docker compose` only works from here.

## 4. Configure

```bash
cp .env.example .env
openssl rand -base64 48   # copy the output for JWT_SECRET
nano .env
```

At minimum, set:

| Variable | Value |
|---|---|
| `JWT_SECRET` | the random string you just generated |
| `POSTGRES_PASSWORD` | any long random string (never needs typing again) |
| `VAULTPROXIES_API_KEY` | your reseller key |
| `PANEL_DOMAIN` / `API_DOMAIN` | already `panel.nullvault.net` / `api.nullvault.net` |
| `PROXY_BRAND_DOMAIN` | already `proxies.nullvault.net` |
| `TRUST_PROXY` | `true` (you are behind the Caddy edge) |

Payment and OAuth keys can stay blank for now — those features simply show as
unavailable until filled in. Save with `Ctrl+O`, `Enter`, `Ctrl+X`.

## 4b. If panel/api are proxied by Cloudflare (orange cloud)

Cloudflare's proxy and Let's Encrypt deadlock on first issue: the ACME challenge
is answered by Cloudflare, which in Full (strict) mode must reach your origin
over HTTPS — but the origin has no certificate yet, because that is the thing
being issued.

Use a **Cloudflare Origin Certificate** instead. It never expires in any
practical sense (15 years), needs no challenge, and no renewal.

1. Cloudflare dashboard -> **SSL/TLS -> Origin Server -> Create Certificate**.
   Accept the defaults; make sure the hostnames cover `panel.` and `api.`
   (a wildcard `*.nullvault.net` plus `nullvault.net` covers both).
2. Cloudflare shows two text blocks. Save them on the server:

```bash
nano deploy/certs/origin.pem   # paste the "Origin Certificate" block
nano deploy/certs/origin.key   # paste the "Private Key" block
chmod 600 deploy/certs/origin.key
```

   The private key is displayed **once** — save it before closing the dialog.

3. Point the edge at the matching config:

```bash
echo 'CADDYFILE=./deploy/Caddyfile.cloudflare' >> .env
```

4. In Cloudflare, set **SSL/TLS mode to Full (strict)**. Origin certificates are
   trusted by Cloudflare but not by browsers, so any weaker mode either fails or
   silently downgrades the connection to your server.

Optionally lock the origin down so nobody can bypass Cloudflare by hitting the
IP directly (worth doing — it is the only thing that makes hiding the IP
meaningful):

```bash
for ip in 173.245.48.0/20 103.21.244.0/22 103.22.200.0/22 103.31.4.0/22 \
          141.101.64.0/18 108.162.192.0/18 190.93.240.0/20 188.114.96.0/20 \
          197.234.240.0/22 198.41.128.0/17 162.158.0.0/15 104.16.0.0/13 \
          104.24.0.0/14 172.64.0.0/13 131.0.72.0/22; do
  ufw allow from $ip to any port 80,443 proto tcp
done
ufw allow 22/tcp        # keep SSH open, or you will lock yourself out
ufw --force enable
```

## 5. Check the reverse-proxy config parses

```bash
docker run --rm -v "$PWD/deploy/Caddyfile:/etc/caddy/Caddyfile:ro" \
  caddy:2-alpine caddy validate --config /etc/caddy/Caddyfile
```

Using the Cloudflare config instead? Validate that one:

```bash
docker run --rm -v "$PWD/deploy/Caddyfile.cloudflare:/etc/caddy/Caddyfile:ro" \
  caddy:2-alpine caddy validate --config /etc/caddy/Caddyfile
```

Expect `Valid configuration`. Fix any error before continuing — a broken config
means no certificates and no site.

## 6. Start

```bash
docker compose --profile edge up -d --build
```

First run takes a few minutes: it builds both images, starts Postgres and Redis,
applies database migrations, and requests TLS certificates from Let's Encrypt.

Watch it come up:

```bash
docker compose logs -f
```

Look for `server listening` from the backend and a certificate line from Caddy.
`Ctrl+C` stops watching (it does not stop the stack).

## 7. Verify

```bash
curl -s https://api.nullvault.net/healthz     # {"status":"ok"}
curl -s https://api.nullvault.net/readyz      # database + upstream reachable
```

Then open `https://panel.nullvault.net` and register an account.

## 8. Make yourself admin

Put your registered email in `.env` as `ADMIN_EMAIL=you@nullvault.net`, then:

```bash
docker compose restart backend
```

The account is promoted on the next start.

## 9. Confirm the whitelabel chain

Order a small plan in the dashboard, generate a proxy line, and run it:

```bash
curl -x http://USERNAME:PASSWORD@resi-gb.proxies.nullvault.net:80 https://api.ipify.org
```

You should get an exit IP that is neither your server's nor your own. Confirm
the generated line shows **`nullvault.net`** and never `vaultproxies.com`.

Also check the logs once for branding problems:

```bash
docker compose logs backend | grep -i "unrecognised upstream gateway"
```

Anything there names a CNAME you still need to create.

## Everyday commands

| Task | Command |
|---|---|
| Logs | `docker compose logs -f` |
| Backend logs only | `docker compose logs -f backend` |
| Restart after `.env` change | `docker compose up -d` |
| Deploy new code | `git pull && docker compose up -d --build` |
| Stop | `docker compose down` |
| Database backup | `docker compose exec postgres pg_dump -U vpx vpx > backup-$(date +%F).sql` |

Changing any `NEXT_PUBLIC_*` value requires a rebuild, not just a restart —
those are compiled into the frontend bundle:

```bash
docker compose up -d --build frontend
```

## Payment webhooks

Once you have Stripe / NOWPayments keys in `.env`, register these URLs in each
provider's dashboard:

- Stripe: `https://api.nullvault.net/api/v1/webhooks/stripe` — event
  `checkout.session.completed`. Copy the signing secret into
  `STRIPE_WEBHOOK_SECRET`.
- NOWPayments IPN: `https://api.nullvault.net/api/v1/webhooks/nowpayments`.
  Copy the IPN secret into `NOWPAYMENTS_IPN_SECRET`.

Then `docker compose up -d` to apply.

## Troubleshooting

| Symptom | Fix |
|---|---|
| No certificate / site not loading | DNS must resolve to this server before Caddy can issue certs. Check `dig +short panel.nullvault.net`. Ports 80 and 443 must be open in any provider firewall. |
| Cert issuance hangs while orange-clouded | Expected — see step 4b, use a Cloudflare Origin Certificate. |
| Cloudflare error 526 | Origin cert missing/mismatched, or SSL mode is not Full (strict). |
| Redirect loop | Cloudflare SSL/TLS mode is "Flexible". Set it to **Full (strict)**. |
| `POSTGRES_PASSWORD is required` | `.env` is missing or you are not in the `vpx` directory. |
| Login always says rate limited | `TRUST_PROXY` is wrong for your setup, or `api` is proxied without the Cloudflare ranges in the Caddyfile. |
| Customers see `vaultproxies.com` | `PROXY_BRAND_DOMAIN` unset, or backend not restarted after setting it. |
| Out of disk | `docker system prune -af` clears old build layers. |
