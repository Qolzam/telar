import type { Metadata } from 'next';
import { roboto } from '@/lib/theme/theme';
import ThemeRegistry from '@/components/ThemeRegistry/ThemeRegistry';
import QueryProvider from '@/lib/react-query/QueryProvider';
import './globals.css';

export const metadata: Metadata = {
  title: 'Telar - Social Network Platform',
  description: 'Modern social networking platform built with Next.js',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body className={roboto.className}>
        <QueryProvider>
          <ThemeRegistry>
            {children}
          </ThemeRegistry>
        </QueryProvider>
      </body>
    </html>
  );
}
