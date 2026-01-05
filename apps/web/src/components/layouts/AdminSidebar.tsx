/**
 * Admin Sidebar Navigation
 * 
 * Side navigation for admin sections
 */

'use client';

import {
  Drawer,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Divider,
  Box,
  Typography,
  Chip,
} from '@mui/material';
import { useRouter, usePathname } from 'next/navigation';
import DashboardIcon from '@mui/icons-material/Dashboard';
import PeopleIcon from '@mui/icons-material/People';
import GavelIcon from '@mui/icons-material/Gavel';
import SettingsIcon from '@mui/icons-material/Settings';
import AdminPanelSettingsIcon from '@mui/icons-material/AdminPanelSettings';
import { useEffect, useState } from 'react';
import { isRTL } from '@/lib/i18n/utils';
import { useTranslation } from 'react-i18next';

const DRAWER_WIDTH = 240;

export default function AdminSidebar() {
  const { t, i18n } = useTranslation('common');
  const router = useRouter();
  const pathname = usePathname();
  const [direction, setDirection] = useState<'ltr' | 'rtl'>('ltr');
  
  useEffect(() => {
    const getLanguage = () => {
      if (typeof document !== 'undefined') {
        const getCookieValue = (name: string): string | null => {
          const value = `; ${document.cookie}`;
          const parts = value.split(`; ${name}=`);
          if (parts.length === 2) {
            return parts.pop()?.split(';').shift() || null;
          }
          return null;
        };
        
        const cookieLang = getCookieValue('i18next') || 'en';
        return cookieLang;
      }
      return i18n.language || 'en';
    };
    
    const lang = getLanguage();
    setDirection(isRTL(lang) ? 'rtl' : 'ltr');
  }, [i18n.language]);

  const MENU_ITEMS = [
    { label: 'Dashboard', path: '/admin', icon: <DashboardIcon /> },
    { label: 'Members', path: '/admin/members', icon: <PeopleIcon /> },
    { label: 'Moderation', path: '/admin/moderation', icon: <GavelIcon /> },
    { label: 'Settings', path: '/settings', icon: <SettingsIcon /> },
  ];

  const handleNavigation = (path: string) => {
    router.push(path);
  };

  return (
    <Drawer
      variant="permanent"
      anchor={direction === 'rtl' ? 'right' : 'left'}
      sx={{
        width: DRAWER_WIDTH,
        flexShrink: 0,
        '& .MuiDrawer-paper': {
          width: DRAWER_WIDTH,
          boxSizing: 'border-box',
          borderRight: '1px solid',
          borderColor: 'divider',
        },
      }}
    >
      <Box sx={{ overflow: 'auto', display: 'flex', flexDirection: 'column', height: '100%' }}>
        {/* Logo/Title */}
        <Box sx={{ p: 3, textAlign: 'center' }}>
          <Box
            sx={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              gap: 1,
              mb: 2,
            }}
          >
            <Box
              sx={{
                width: 48,
                height: 48,
                borderRadius: 2,
                background: 'linear-gradient(135deg, #d32f2f 0%, #f44336 100%)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                boxShadow: '0 4px 12px rgba(211, 47, 47, 0.3)',
              }}
            >
              <AdminPanelSettingsIcon sx={{ color: 'white', fontSize: 28 }} />
            </Box>
          </Box>
          <Typography variant="h6" sx={{ fontWeight: 700, mb: 0.5 }}>
            Admin Panel
          </Typography>
          <Chip
            label="Administrator"
            size="small"
            color="error"
            sx={{ fontSize: '0.7rem', height: 20 }}
          />
        </Box>

        <Divider />

        {/* Navigation Items */}
        <List sx={{ flexGrow: 1, pt: 1 }}>
          {MENU_ITEMS.map((item) => {
            const isActive = pathname === item.path || (item.path !== '/admin' && pathname?.startsWith(item.path));
            return (
              <ListItem key={item.path} disablePadding sx={{ mb: 0.5, px: 1.5 }}>
                <ListItemButton
                  onClick={() => handleNavigation(item.path)}
                  selected={isActive}
                  sx={{
                    borderRadius: 2,
                    '&.Mui-selected': {
                      backgroundColor: 'error.main',
                      color: 'white',
                      '&:hover': {
                        backgroundColor: 'error.dark',
                      },
                      '& .MuiListItemIcon-root': {
                        color: 'white',
                      },
                    },
                    '&:hover': {
                      backgroundColor: 'action.hover',
                    },
                  }}
                >
                  <ListItemIcon
                    sx={{
                      minWidth: 40,
                      color: isActive ? 'white' : 'text.secondary',
                    }}
                  >
                    {item.icon}
                  </ListItemIcon>
                  <ListItemText
                    primary={item.label}
                    primaryTypographyProps={{
                      fontWeight: isActive ? 600 : 500,
                      fontSize: '0.95rem',
                    }}
                  />
                </ListItemButton>
              </ListItem>
            );
          })}
        </List>

        <Divider />

        {/* Footer */}
        <Box sx={{ p: 2, textAlign: 'center' }}>
          <Typography variant="caption" color="text.secondary">
            Telar Admin v1.0
          </Typography>
        </Box>
      </Box>
    </Drawer>
  );
}


