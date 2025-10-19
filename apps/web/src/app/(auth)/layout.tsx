/**
 * Auth Layout
 * 
 * Simple layout for authentication pages
 */

import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Authentication | Telar',
};

export default function AuthLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="auth-layout">
      {children}
    </div>
  );
}
