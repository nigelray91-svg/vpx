import { ImageResponse } from 'next/og';
import { brand } from '@/lib/brand';

export const alt = `${brand.name} — Premium Proxies`;
export const size = { width: 1200, height: 630 };
export const contentType = 'image/png';

export default function OgImage() {
  return new ImageResponse(
    (
      <div
        style={{
          width: '100%',
          height: '100%',
          display: 'flex',
          flexDirection: 'column',
          justifyContent: 'center',
          padding: '80px',
          background:
            'radial-gradient(1000px 500px at 20% -10%, #312e81 0%, transparent 60%), linear-gradient(135deg, #06060d 0%, #0f0f1a 100%)',
          color: 'white',
          fontFamily: 'sans-serif',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 22 }}>
          <div
            style={{
              width: 64,
              height: 64,
              borderRadius: 18,
              background: 'linear-gradient(135deg, #818cf8, #6366f1, #22d3ee)',
              display: 'flex',
            }}
          />
          <div style={{ fontSize: 40, fontWeight: 700 }}>{brand.name}</div>
        </div>
        <div style={{ display: 'flex', marginTop: 48, fontSize: 76, fontWeight: 800, lineHeight: 1.05, maxWidth: 900 }}>
          The proxy network built for scale & trust
        </div>
        <div style={{ display: 'flex', marginTop: 28, fontSize: 32, color: '#94a3b8', maxWidth: 920 }}>
          {`Residential · ISP · Datacenter · IPv6 · Mobile — ${brand.ipCount} IPs, instant provisioning.`}
        </div>
      </div>
    ),
    { ...size },
  );
}
