'use client';

import { Activity, LayoutDashboard, Network, ShieldCheck } from 'lucide-react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import type { ReactNode } from 'react';

import { LogoutButton } from '@/components/logout-button';
import { cn } from '@/lib/cn';

const navigation = [
  { href: '/dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { href: '/graph', label: 'Graph explorer', icon: Network },
];

type AppShellProps = {
  title: string;
  description: string;
  children: ReactNode;
  actions?: ReactNode;
};

export function AppShell({ title, description, actions, children }: AppShellProps) {
  const pathname = usePathname();

  return (
    <div className="app-shell">
      <div className="app-shell-inner">
        <header className="topbar">
          <div className="brand-block">
            <Link className="brand" href="/dashboard">
              <span className="brand-mark">
                <Activity size={18} />
              </span>
              <span>
                <strong>Open Context</strong>
                <small>Protected admin console</small>
              </span>
            </Link>
            <nav className="nav-links" aria-label="Primary">
              {navigation.map(({ href, label, icon: Icon }) => (
                <Link
                  key={href}
                  className={cn('nav-link', pathname === href && 'nav-link-active')}
                  href={href}
                >
                  <Icon size={16} />
                  {label}
                </Link>
              ))}
            </nav>
          </div>
          <div className="topbar-actions">
            <div className="status-chip">
              <ShieldCheck size={16} />
              Signed admin session
            </div>
            <LogoutButton />
          </div>
        </header>

        <section className="hero-panel">
          <div>
            <p className="eyebrow">Admin workspace</p>
            <h1>{title}</h1>
            <p className="hero-description">{description}</p>
          </div>
          {actions ? <div className="hero-actions">{actions}</div> : null}
        </section>

        <main className="page-content">{children}</main>
      </div>
    </div>
  );
}
