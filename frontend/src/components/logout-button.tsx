'use client';

import { LogOut } from 'lucide-react';
import { useRouter } from 'next/navigation';
import { useState } from 'react';

import { Button } from '@/components/button';

export function LogoutButton() {
  const router = useRouter();
  const [isPending, setIsPending] = useState(false);

  async function onLogout() {
    try {
      setIsPending(true);
      await fetch('/api/auth/logout', { method: 'POST' });
    } finally {
      router.push('/');
      router.refresh();
      setIsPending(false);
    }
  }

  return (
    <Button
      aria-label="Sign out"
      disabled={isPending}
      onClick={onLogout}
      size="sm"
      variant="ghost"
    >
      <LogOut size={16} />
      {isPending ? 'Signing out...' : 'Sign out'}
    </Button>
  );
}
