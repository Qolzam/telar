/**
 * Admin Header
 * 
 * Top navigation bar for admin panel
 */

'use client';

import {
  AppBar,
  Toolbar,
  Typography,
  Avatar,
  IconButton,
  Menu,
  MenuItem,
  Box,
  CircularProgress,
  Button,
} from '@mui/material';
import { useState } from 'react';
import { useSession, useLogout } from '@/features/auth/client';
import SettingsIcon from '@mui/icons-material/Settings';
import LogoutIcon from '@mui/icons-material/Logout';
import AccountCircleIcon from '@mui/icons-material/AccountCircle';
import HomeIcon from '@mui/icons-material/Home';
import { useRouter } from 'next/navigation';
import { useTranslation } from 'react-i18next';

export default function AdminHeader() {
  const { t } = useTranslation('common');
  const { user, isAuthenticated, isLoading } = useSession();
  const { logout, isLoading: isLoggingOut } = useLogout();
  const router = useRouter();
  
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const open = Boolean(anchorEl);

  const handleMenuOpen = (event: React.MouseEvent<HTMLElement>) => {
    setAnchorEl(event.currentTarget);
  };

  const handleMenuClose = () => {
    setAnchorEl(null);
  };

  const handleProfileClick = () => {
    setAnchorEl(null);
    router.push('/profile');
  };

  const handleSettingsClick = () => {
    setAnchorEl(null);
    router.push('/settings');
  };

  const handleHomeClick = () => {
    router.push('/dashboard');
  };

  const handleLogoutClick = () => {
    setAnchorEl(null);
    logout();
  };

  return (
    <AppBar position="static" color="default" elevation={1} sx={{ bgcolor: 'background.paper' }}>
      <Toolbar sx={{ justifyContent: 'space-between', gap: 2, px: { xs: 2, sm: 3 } }}>
        {isLoading ? (
          <CircularProgress size={24} />
        ) : isAuthenticated && user ? (
          <>
            {/* Left side: Title and Home Button */}
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
              <Typography variant="h6" sx={{ fontWeight: 700, color: 'text.primary' }}>
                Admin Dashboard
              </Typography>
              <Button
                variant="outlined"
                size="small"
                startIcon={<HomeIcon />}
                onClick={handleHomeClick}
                sx={{ ml: 2 }}
              >
                Back to App
              </Button>
            </Box>

            {/* Right side: User Profile */}
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              {/* Settings */}
              <IconButton color="inherit" size="medium" onClick={handleSettingsClick}>
                <SettingsIcon />
              </IconButton>

              {/* User Profile */}
              <Box
                sx={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 1,
                  cursor: 'pointer',
                  px: 1,
                  py: 0.5,
                  borderRadius: 1,
                  '&:hover': {
                    bgcolor: 'action.hover',
                  },
                }}
                onClick={handleMenuOpen}
              >
                <Avatar
                  alt={user.displayName}
                  src={user.avatar}
                  sx={{ width: 32, height: 32 }}
                />
                <Typography variant="body2" sx={{ display: { xs: 'none', sm: 'block' } }}>
                  {user.displayName}
                </Typography>
              </Box>

              {/* User Menu */}
              <Menu
                anchorEl={anchorEl}
                open={open}
                onClose={handleMenuClose}
                onClick={handleMenuClose}
                autoFocus={false}
                disableAutoFocusItem={true}
                transformOrigin={{ horizontal: 'right', vertical: 'top' }}
                anchorOrigin={{ horizontal: 'right', vertical: 'bottom' }}
              >
                <MenuItem onClick={handleProfileClick}>
                  <AccountCircleIcon sx={{ mr: 1.5 }} />
                  {t('navigation.profile')}
                </MenuItem>
                <MenuItem onClick={handleSettingsClick}>
                  <SettingsIcon sx={{ mr: 1.5 }} />
                  {t('navigation.settings')}
                </MenuItem>
                <MenuItem onClick={handleLogoutClick} disabled={isLoggingOut}>
                  <LogoutIcon sx={{ mr: 1.5 }} />
                  {isLoggingOut ? t('states.processing') : t('navigation.logout')}
                </MenuItem>
              </Menu>
            </Box>
          </>
        ) : null}
      </Toolbar>
    </AppBar>
  );
}


