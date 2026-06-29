// Central whitelabel brand configuration.
//
// Everything customer-facing reads from here, so a reseller can rebrand the
// entire site by setting a handful of NEXT_PUBLIC_* env vars (or editing the
// defaults below) — no other code changes required. The backend's
// /api/v1/config.site_name overrides the name at runtime where available.

export const brand = {
  name: process.env.NEXT_PUBLIC_SITE_NAME?.trim() || 'Proxia',
  tagline:
    process.env.NEXT_PUBLIC_SITE_TAGLINE?.trim() ||
    'Premium residential, ISP, datacenter & mobile proxies — provisioned in seconds.',
  // Support contact shown in the footer.
  supportEmail: process.env.NEXT_PUBLIC_SUPPORT_EMAIL?.trim() || 'support@proxia.io',
  // Network size headline number (purely marketing copy).
  ipCount: process.env.NEXT_PUBLIC_IP_COUNT?.trim() || '32M+',
};

export type Brand = typeof brand;
