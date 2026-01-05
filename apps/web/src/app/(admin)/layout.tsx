/**
 * Admin Layout
 * 
 * Layout for admin pages with sidebar and header
 * 
 * Note: Authentication and role checks are handled by middleware.
 * This layout is a server component for proper Next.js route resolution.
 */

import type { Metadata } from 'next';
import { Box } from '@mui/material';
import AdminSidebar from '@/components/layouts/AdminSidebar';
import AdminHeader from '@/components/layouts/AdminHeader';
import { TechPreviewBanner } from '@/components/TechPreviewBanner';

export const metadata: Metadata = {
  title: 'Admin Dashboard | Telar',
};

export default function AdminLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <Box sx={{ display: 'flex', minHeight: '100vh' }}>
      {/* Sidebar */}
      <AdminSidebar />
      
      {/* Main content area */}
      <Box sx={{ flexGrow: 1, display: 'flex', flexDirection: 'column' }}>
        <TechPreviewBanner />

        {/* Header */}
        <AdminHeader />

        {/* Page content */}
        <Box 
          component="main" 
          sx={{ 
            flexGrow: 1, 
            p: 3, 
            bgcolor: 'background.default' 
          }}
        >
          {children}
        </Box>
      </Box>
    </Box>
  );
}






