import type { Metadata } from 'next';
import { roboto } from '@/lib/theme/theme';
import ThemeRegistry from '@/components/ThemeRegistry/ThemeRegistry';
import QueryProvider from '@/lib/react-query/QueryProvider';
import { I18nProvider } from '@/lib/provider/I18nProvider';
import { cookies } from 'next/headers';
import { cookieName, fallbackLng } from '@/lib/i18n/settings';
import { getDirection } from '@/lib/i18n/utils';
import './globals.css';

export const metadata: Metadata = {
  title: 'Telar - Social Network Platform',
  description: 'Modern social networking platform built with Next.js',
};

export default async function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  // Read locale from cookie (set by middleware)
  const cookieStore = await cookies();
  const locale = cookieStore.get(cookieName)?.value || fallbackLng;
  const direction = getDirection(locale);

  return (
    <html lang={locale} dir={direction} suppressHydrationWarning data-scroll-behavior="smooth">
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
        <I18nProvider>
          <QueryProvider>
            <ThemeRegistry direction={direction}>
              {children}
            </ThemeRegistry>
          </QueryProvider>
        </I18nProvider>
      </body>
    </html>
  );
}
