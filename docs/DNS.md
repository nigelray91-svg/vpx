# DNS Setup

Everything you need to add at your domain registrar / DNS provider for
`nullvault.net`. Nothing here requires changes on the rented server itself.

## The mental model

Your domain does **two unrelated jobs**, and they point at different machines:

```
                    ┌─────────────────────────────────────────┐
  Customer's        │  panel.nullvault.net  ─┐                │
  browser  ────────▶│  api.nullvault.net    ─┴─▶ YOUR SERVER  │   A records
                    │                            (the VPS)    │
                    └─────────────────────────────────────────┘

                    ┌─────────────────────────────────────────┐
  Customer's        │  resi-gb.proxies.nullvault.net ─┐       │
  scraper  ────────▶│  mobile.proxies.nullvault.net  ─┴─▶     │   CNAME records
  / browser         │           VAULTPROXIES' GATEWAYS        │
                    └─────────────────────────────────────────┘
```

1. **The website** (panel + API) runs on the server you rent. Those names need
   **A records** pointing at your server's IP address.
2. **The proxies** run on VaultProxies' infrastructure. Those names need
   **CNAME records** pointing at their gateway hostnames.

Proxy traffic goes straight from your customer to the upstream gateway. It does
**not** pass through your server, so your VPS needs no extra ports, no extra
config, and carries none of that bandwidth. The CNAMEs exist purely so the
hostname your customer sees says `nullvault.net` instead of the supplier's name.

## Records to create

Replace `203.0.113.10` with your server's real public IP.

### 1. The website — A records → your server

| Type | Name    | Value          | Proxy status |
|------|---------|----------------|--------------|
| A    | `panel` | `203.0.113.10` | Proxied OK   |
| A    | `api`   | `203.0.113.10` | Proxied OK   |

(If you prefer the bare domain, add an A record for `@` as well.)

### 2. The proxies — CNAME records → upstream gateways

Only create the ones for plans you actually sell.

| Type  | Name              | Value                     | Proxy status  |
|-------|-------------------|---------------------------|---------------|
| CNAME | `resi-gb.proxies` | `resi-gb.vaultproxies.com` | **DNS only** |
| CNAME | `resi.proxies`    | `resi.vaultproxies.com`    | **DNS only** |
| CNAME | `eu-dc.proxies`   | `eu-dc.vaultproxies.com`   | **DNS only** |
| CNAME | `na-dc.proxies`   | `na-dc.vaultproxies.com`   | **DNS only** |
| CNAME | `dc-gb.proxies`   | `dc-gb.vaultproxies.com`   | **DNS only** |
| CNAME | `mobile.proxies`  | `mobile.vaultproxies.com`  | **DNS only** |
| CNAME | `na.proxies`      | `na.vaultproxies.com`      | **DNS only** |
| CNAME | `isp.proxies`     | `isp.vaultproxies.com`     | **DNS only** |
| CNAME | `eu-isp.proxies`  | `eu-isp.vaultproxies.com`  | **DNS only** |
| CNAME | `ipv6.proxies`    | `ipv6.vaultproxies.com`    | **DNS only** |

Most DNS panels append your domain automatically, so entering `resi-gb.proxies`
produces `resi-gb.proxies.nullvault.net`. If yours wants the whole thing, type
the full name.

> **Cloudflare users:** the cloud icon next to each proxy CNAME must be **grey
> ("DNS only")**, not orange. Orange sends traffic through Cloudflare's HTTP
> proxy, which does not speak the proxy protocol — every proxy line would break.
> The `panel` and `api` A records may stay orange.

## Matching backend configuration

```ini
PROXY_BRAND_DOMAIN=proxies.nullvault.net
PROXY_BRAND_STRICT=true

PUBLIC_BASE_URL=https://panel.nullvault.net
API_BASE_URL=https://api.nullvault.net
COOKIE_DOMAIN=.nullvault.net
NEXT_PUBLIC_API_BASE_URL=https://api.nullvault.net
NEXT_PUBLIC_SITE_URL=https://panel.nullvault.net
```

`panel` and `api` share the registrable domain `nullvault.net`, which is what
lets the auth cookies work across both.

The backend keeps the gateway label and swaps the domain, so
`resi-gb.vaultproxies.com` becomes `resi-gb.proxies.nullvault.net` — which is
exactly the CNAME you created. Ports are untouched: a CNAME maps names, not
ports, so one record per gateway covers every port it listens on (80, 777, 666,
10808, 30, 31, …).

## Verifying

DNS changes take a few minutes to propagate (occasionally up to an hour).

```bash
# Should print the upstream hostname, then an IP.
dig +short resi-gb.proxies.nullvault.net

# The website should resolve to your server's IP.
dig +short panel.nullvault.net
```

Then prove a real proxy works end to end, using credentials generated in the
dashboard:

```bash
curl -x http://USERNAME:PASSWORD@resi-gb.proxies.nullvault.net:80 https://api.ipify.org
```

That should print an exit IP that is **not** your server's. If it does, the
whitelabel chain is working.

## Troubleshooting

| Symptom | Cause |
|---|---|
| `dig` returns nothing | Record not created, wrong name, or still propagating. |
| Proxy times out / TLS errors | The CNAME is orange-clouded in Cloudflare. Set it to DNS only. |
| `curl` returns your server's IP | You pointed the proxy CNAME at your own server instead of the upstream gateway. |
| 407 / auth failure | Credentials are wrong or expired — regenerate them in the dashboard. |
| Customers still see `vaultproxies.com` | `PROXY_BRAND_DOMAIN` is unset, or the backend was not restarted after setting it. |
