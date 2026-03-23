'use client';

import { LockKeyhole, Network, ShieldCheck, Sparkles } from 'lucide-react';
import { useRouter, useSearchParams } from 'next/navigation';
import { useState } from 'react';

import { Button } from '@/components/button';
import { Card } from '@/components/card';
import { TextField } from '@/components/text-field';

const highlights = [
  {
    title: 'Protected sessions',
    description: 'Signed admin cookies and tighter route checks keep the UI behind a real session.',
    icon: <ShieldCheck size={18} />,
  },
  {
    title: 'Graph-first workflow',
    description: 'Jump straight from sign-in to exploring user or group graph relationships.',
    icon: <Network size={18} />,
  },
  {
    title: 'Operational clarity',
    description: 'Focused admin tooling without the clutter of a full customer-facing product shell.',
    icon: <Sparkles size={18} />,
  },
  {
    title: 'Simple access model',
    description: 'A compact admin surface with one protected entry point and one job: graph inspection.',
    icon: <LockKeyhole size={18} />,
  },
];

export function LoginPageContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    setIsSubmitting(true);

    try {
      const res = await fetch('/api/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      });
      const payload = (await res.json().catch(() => null)) as { message?: string } | null;

      if (!res.ok) {
        if (res.status === 401) {
          setError('Incorrect username or password.');
          return;
        }

        setError(payload?.message ?? 'Unable to sign in right now.');
        return;
      }

      router.push(searchParams.get('next') || '/dashboard');
      router.refresh();
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <main className="auth-shell">
      <section className="auth-copy">
        <p className="eyebrow">Open Context admin</p>
        <h1 className="auth-title">Professional graph operations, not a placeholder page.</h1>
        <p>
          This admin console is now designed to feel like a real internal tool: clear hierarchy,
          protected access, and a faster path into graph inspection workflows.
        </p>

        <div className="auth-card-grid">
          {highlights.map((item) => (
            <Card
              key={item.title}
              description={item.description}
              icon={item.icon}
              title={item.title}
            />
          ))}
        </div>
      </section>

      <section className="auth-panel">
        <p className="kicker">Admin sign in</p>
        <div className="stack">
          <div>
            <h2 style={{ marginTop: 0, marginBottom: 8 }}>Access the protected workspace</h2>
            <p className="muted">
              Sign in with your configured admin credentials to inspect user and group graphs.
            </p>
          </div>

          <form className="stack" onSubmit={onSubmit}>
            <TextField
              autoComplete="username"
              label="Username"
              name="username"
              onChange={(event) => setUsername(event.target.value)}
              placeholder="admin"
              value={username}
            />
            <TextField
              autoComplete="current-password"
              error={error}
              label="Password"
              name="password"
              onChange={(event) => setPassword(event.target.value)}
              placeholder="Enter your password"
              type="password"
              value={password}
            />
            <Button disabled={isSubmitting} type="submit">
              {isSubmitting ? 'Signing in...' : 'Sign in to admin'}
            </Button>
          </form>

          <dl className="meta-list">
            <div>
              <dt>Session model</dt>
              <dd>Signed, expiring admin cookie</dd>
            </div>
            <div>
              <dt>Protected routes</dt>
              <dd>`/dashboard` and `/graph`</dd>
            </div>
          </dl>
        </div>
      </section>
    </main>
  );
}
