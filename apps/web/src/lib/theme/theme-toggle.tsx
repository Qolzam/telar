'use client';

import { useState } from 'react';
import {
  Box,
  Button,
  Menu,
  MenuItem,
  ListItemIcon,
  ListItemText,
  Typography,
  Divider,
} from '@mui/material';
import {
  LightMode as LightModeIcon,
  DarkMode as DarkModeIcon,
  Settings as SystemIcon,
  KeyboardArrowDown as ArrowDownIcon,
} from '@mui/icons-material';
import { useTheme } from './use-theme';

interface ThemeToggleProps {
  variant?: 'button' | 'menu';
  size?: 'small' | 'medium' | 'large';
  fullWidth?: boolean;
}

export function ThemeToggle({ 
  variant = 'button', 
  size = 'medium',
  fullWidth = false 
}: ThemeToggleProps) {
  const { colorScheme, setColorScheme, resolvedMode } = useTheme();
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const open = Boolean(anchorEl);

  const handleClick = (event: React.MouseEvent<HTMLElement>) => {
    setAnchorEl(event.currentTarget);
  };

  const handleClose = () => {
    setAnchorEl(null);
  };

  const handleThemeChange = (scheme: 'light' | 'dark' | 'system') => {
    setColorScheme(scheme);
    handleClose();
  };

  const getCurrentIcon = () => {
    switch (colorScheme) {
      case 'light':
        return <LightModeIcon />;
      case 'dark':
        return <DarkModeIcon />;
      case 'system':
        return <SystemIcon />;
      default:
        return <SystemIcon />;
    }
  };

  const getCurrentLabel = () => {
    switch (colorScheme) {
      case 'light':
        return 'Light';
      case 'dark':
        return 'Dark';
      case 'system':
        return `System (${resolvedMode === 'dark' ? 'Dark' : 'Light'})`;
      default:
        return 'System';
    }
  };

  if (variant === 'button') {
    return (
      <Button
        variant="outlined"
        size={size}
        fullWidth={fullWidth}
        startIcon={getCurrentIcon()}
        endIcon={<ArrowDownIcon />}
        onClick={handleClick}
        sx={{
          justifyContent: 'space-between',
          textTransform: 'none',
        }}
      >
        {getCurrentLabel()}
      </Button>
    );
  }

  return (
    <>
      <Button
        variant="outlined"
        size={size}
        fullWidth={fullWidth}
        startIcon={getCurrentIcon()}
        endIcon={<ArrowDownIcon />}
        onClick={handleClick}
        sx={{
          justifyContent: 'space-between',
          textTransform: 'none',
        }}
      >
        {getCurrentLabel()}
      </Button>

      <Menu
        anchorEl={anchorEl}
        open={open}
        onClose={handleClose}
        anchorOrigin={{
          vertical: 'bottom',
          horizontal: 'left',
        }}
        transformOrigin={{
          vertical: 'top',
          horizontal: 'left',
        }}
        slotProps={{
          paper: {
            sx: {
              minWidth: 200,
              mt: 1,
            },
          },
        }}
      >
        <Box sx={{ px: 2, py: 1 }}>
          <Typography variant="subtitle2" color="text.secondary">
            Theme
          </Typography>
        </Box>
        <Divider />

        <MenuItem 
          onClick={() => handleThemeChange('light')}
          selected={colorScheme === 'light'}
        >
          <ListItemIcon>
            <LightModeIcon fontSize="small" />
          </ListItemIcon>
          <ListItemText primary="Light" />
        </MenuItem>

        <MenuItem 
          onClick={() => handleThemeChange('dark')}
          selected={colorScheme === 'dark'}
        >
          <ListItemIcon>
            <DarkModeIcon fontSize="small" />
          </ListItemIcon>
          <ListItemText primary="Dark" />
        </MenuItem>

        <MenuItem 
          onClick={() => handleThemeChange('system')}
          selected={colorScheme === 'system'}
        >
          <ListItemIcon>
            <SystemIcon fontSize="small" />
          </ListItemIcon>
          <ListItemText 
            primary="System" 
            secondary={resolvedMode === 'dark' ? 'Dark' : 'Light'}
          />
        </MenuItem>
      </Menu>
    </>
  );
}



