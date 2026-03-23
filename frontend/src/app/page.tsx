import { Suspense } from 'react';

import { LoginPageContent } from '@/components/login-page-content';

export default function LoginPage() {
  return (
    <Suspense fallback={<main className="auth-shell" />}>
      <LoginPageContent />
    </Suspense>
  );
}
