/**
 * Dashboard Layout
 * 
 * Layout for authenticated pages with header and sidebar
 */

import type { Metadata } from 'next';
import DashboardHeader from '@/components/layouts/DashboardHeader';
import DashboardSidebar from '@/components/layouts/DashboardSidebar';
import { Box } from '@mui/material';

export const metadata: Metadata = {
  title: 'Dashboard | Telar',
};

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <Box sx={{ display: 'flex', minHeight: '100vh' }}>
      {/* Sidebar */}
      <DashboardSidebar />
      
      {/* Main content area */}
      <Box sx={{ flexGrow: 1, display: 'flex', flexDirection: 'column' }}>
        {/* Header */}
        <DashboardHeader />
        
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
