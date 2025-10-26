/**
 * Top navigation bar with user profile and logout
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
} from '@mui/material';
import { useState } from 'react';
import { useSession, useLogout } from '@/features/auth/client';
import NotificationsIcon from '@mui/icons-material/Notifications';
import SettingsIcon from '@mui/icons-material/Settings';
import LogoutIcon from '@mui/icons-material/Logout';
import AccountCircleIcon from '@mui/icons-material/AccountCircle';
import { useRouter } from 'next/navigation';

export default function DashboardHeader() {
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
    handleMenuClose();
    router.push('/profile');
  };

  const handleSettingsClick = () => {
    handleMenuClose();
    router.push('/settings');
  };

  const handleLogoutClick = () => {
    handleMenuClose();
    logout();
  };

  return (
    <AppBar position="static" color="default" elevation={0}>
      <Toolbar sx={{ justifyContent: 'flex-end', gap: 1 }}>
        {isLoading ? (
          <CircularProgress size={24} />
        ) : isAuthenticated && user ? (
          <>
            {/* Notifications */}
            <IconButton color="inherit" size="medium">
              <NotificationsIcon />
            </IconButton>

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
              transformOrigin={{ horizontal: 'right', vertical: 'top' }}
              anchorOrigin={{ horizontal: 'right', vertical: 'bottom' }}
            >
              <MenuItem onClick={handleProfileClick}>
                <AccountCircleIcon sx={{ mr: 1.5 }} />
                Profile
              </MenuItem>
              <MenuItem onClick={handleSettingsClick}>
                <SettingsIcon sx={{ mr: 1.5 }} />
                Settings
              </MenuItem>
              <MenuItem onClick={handleLogoutClick} disabled={isLoggingOut}>
                <LogoutIcon sx={{ mr: 1.5 }} />
                {isLoggingOut ? 'Logging out...' : 'Logout'}
              </MenuItem>
            </Menu>
          </>
        ) : null}
      </Toolbar>
    </AppBar>
  );
}
