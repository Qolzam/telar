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
    <AppBar position="static" color="default" elevation={1}>
      <Toolbar>
        <Typography variant="h6" component="div" sx={{ flexGrow: 1 }}>
          Telar
        </Typography>

        {isLoading ? (
          <CircularProgress size={24} />
        ) : isAuthenticated && user ? (
          <>
            {/* Notifications */}
            <IconButton color="inherit" sx={{ mr: 1 }}>
              <NotificationsIcon />
            </IconButton>

            {/* Settings */}
            <IconButton color="inherit" sx={{ mr: 2 }} onClick={handleSettingsClick}>
              <SettingsIcon />
            </IconButton>

            {/* User Profile */}
            <Box
              sx={{
                display: 'flex',
                alignItems: 'center',
                cursor: 'pointer',
              }}
              onClick={handleMenuOpen}
            >
              <Avatar
                alt={user.displayName}
                src={user.avatar}
                sx={{ width: 36, height: 36, mr: 1 }}
              />
              <Typography variant="body1">{user.displayName}</Typography>
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
                <AccountCircleIcon sx={{ mr: 1 }} />
                Profile
              </MenuItem>
              <MenuItem onClick={handleSettingsClick}>
                <SettingsIcon sx={{ mr: 1 }} />
                Settings
              </MenuItem>
              <MenuItem onClick={handleLogoutClick} disabled={isLoggingOut}>
                <LogoutIcon sx={{ mr: 1 }} />
                {isLoggingOut ? 'Logging out...' : 'Logout'}
              </MenuItem>
            </Menu>
          </>
        ) : null}
      </Toolbar>
    </AppBar>
  );
}
