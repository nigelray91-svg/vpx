'use client';

import { useRouter } from 'next/navigation';
import { Badge, Button, Card } from '@/components/ui';
import { useAuth } from '@/components/AuthProvider';
import { titleCase } from '@/lib/format';

export default function SettingsPage() {
  const { user, logout } = useAuth();
  const router = useRouter();

  if (!user) return null;

  const onLogout = async () => {
    await logout();
    router.replace('/login');
  };

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-white">Settings</h1>

      <Card className="max-w-2xl">
        <h2 className="text-base font-semibold text-white">Profile</h2>
        <dl className="mt-4 divide-y divide-ink-700 text-sm">
          <Row label="Full name" value={user.full_name || '—'} />
          <Row label="Email" value={user.email} />
          <Row
            label="Role"
            value={<Badge tone={user.role === 'admin' ? 'info' : 'default'}>{titleCase(user.role)}</Badge>}
          />
        </dl>
      </Card>

      <Card className="max-w-2xl">
        <h2 className="text-base font-semibold text-white">Security</h2>
        <p className="mt-2 text-sm text-slate-400">
          Signing out revokes this device&apos;s session. Refresh tokens rotate
          automatically on every use.
        </p>
        <Button variant="danger" className="mt-4" onClick={onLogout}>
          Sign out
        </Button>
      </Card>
    </div>
  );
}

function Row({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between py-3">
      <dt className="text-slate-400">{label}</dt>
      <dd className="font-medium text-slate-100">{value}</dd>
    </div>
  );
}
