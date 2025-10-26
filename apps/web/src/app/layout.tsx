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
    <html lang="en" suppressHydrationWarning data-scroll-behavior="smooth">
      <body className={roboto.className}>
        <script
          dangerouslySetInnerHTML={{
            __html: `
              (function() {
                try {
                  var mode = localStorage.getItem('theme-mode') || 'light';
                  document.documentElement.setAttribute('data-mui-color-scheme', mode);
                } catch (e) {}
              })();
            `,
          }}
        />
        <QueryProvider>
          <ThemeRegistry>
            {children}
          </ThemeRegistry>
        </QueryProvider>
      </body>
    </html>
  );
}
