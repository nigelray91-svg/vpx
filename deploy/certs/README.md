# Origin certificates

Only used when serving hostnames that Cloudflare proxies (orange cloud), with
`CADDYFILE=./deploy/Caddyfile.cloudflare`.

Create the pair in the Cloudflare dashboard under
**SSL/TLS -> Origin Server -> Create Certificate**, then save them here on the
server as:

- `origin.pem` — the certificate ("Origin Certificate" block)
- `origin.key` — the private key ("Private Key" block)

```bash
chmod 600 origin.key
```

Both are gitignored. The private key is shown by Cloudflare exactly once, so
save it before closing the dialog. Nothing else in this directory is read.
