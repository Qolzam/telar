'use client';

import { useState, useCallback } from 'react';
import {
  IconButton,
  Tooltip,
  Menu,
  MenuItem,
  ListItemIcon,
  ListItemText,
  Box,
  Typography,
  Divider,
  Fade,
} from '@mui/material';
import {
  LightMode as LightModeIcon,
  DarkMode as DarkModeIcon,
  Settings as SystemIcon,
  Palette as PaletteIcon,
} from '@mui/icons-material';
import { useTheme } from '@/lib/theme/use-theme';

interface ThemeToggleProps {
  variant?: 'icon' | 'button';
  size?: 'small' | 'medium' | 'large';
  showLabel?: boolean;
  className?: string;
}

export function ThemeToggle({ 
  variant = 'icon', 
  size = 'medium',
  showLabel = false,
  className 
}: ThemeToggleProps) {
  const { colorScheme, setColorScheme, resolvedMode } = useTheme();
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const open = Boolean(anchorEl);

  const handleClick = useCallback((event: React.MouseEvent<HTMLElement>) => {
    setAnchorEl(event.currentTarget);
  }, []);

  const handleClose = useCallback(() => {
    setAnchorEl(null);
  }, []);

  const handleThemeChange = useCallback((scheme: 'light' | 'dark' | 'system') => {
    setColorScheme(scheme);
    handleClose();
  }, [setColorScheme, handleClose]);

  const getCurrentIcon = useCallback(() => {
    switch (colorScheme) {
      case 'light':
        return <LightModeIcon />;
      case 'dark':
        return <DarkModeIcon />;
      case 'system':
        return <SystemIcon />;
      default:
        return <PaletteIcon />;
    }
  }, [colorScheme]);

  const getTooltipTitle = useCallback(() => {
    switch (colorScheme) {
      case 'light':
        return 'Light theme';
      case 'dark':
        return 'Dark theme';
      case 'system':
        return `System theme (${resolvedMode})`;
      default:
        return 'Theme settings';
    }
  }, [colorScheme, resolvedMode]);

  const getCurrentLabel = useCallback(() => {
    switch (colorScheme) {
      case 'light':
        return 'Light';
      case 'dark':
        return 'Dark';
      case 'system':
        return `System (${resolvedMode})`;
      default:
        return 'Theme';
    }
  }, [colorScheme, resolvedMode]);

  if (variant === 'button') {
    return (
      <>
        <Tooltip title={getTooltipTitle()}>
          <IconButton
            onClick={handleClick}
            color="inherit"
            size={size}
            className={className}
            aria-label="Theme toggle"
            aria-haspopup="true"
            aria-expanded={open}
          >
            {getCurrentIcon()}
            {showLabel && (
              <Typography variant="body2" sx={{ ml: 1 }}>
                {getCurrentLabel()}
              </Typography>
            )}
          </IconButton>
        </Tooltip>

        <Menu
          anchorEl={anchorEl}
          open={open}
          onClose={handleClose}
          anchorOrigin={{
            vertical: 'bottom',
            horizontal: 'right',
          }}
          transformOrigin={{
            vertical: 'top',
            horizontal: 'right',
          }}
          slotProps={{
            paper: {
              sx: {
                minWidth: 200,
                mt: 1,
              },
            },
          }}
          TransitionComponent={Fade}
          TransitionProps={{ timeout: 150 }}
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
            aria-label="Switch to light theme"
          >
            <ListItemIcon>
              <LightModeIcon fontSize="small" />
            </ListItemIcon>
            <ListItemText primary="Light" />
          </MenuItem>

          <MenuItem 
            onClick={() => handleThemeChange('dark')}
            selected={colorScheme === 'dark'}
            aria-label="Switch to dark theme"
          >
            <ListItemIcon>
              <DarkModeIcon fontSize="small" />
            </ListItemIcon>
            <ListItemText primary="Dark" />
          </MenuItem>

          <MenuItem 
            onClick={() => handleThemeChange('system')}
            selected={colorScheme === 'system'}
            aria-label="Switch to system theme"
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

  return (
    <Tooltip title={getTooltipTitle()}>
      <IconButton
        onClick={handleClick}
        color="inherit"
        size={size}
        className={className}
        aria-label="Theme toggle"
        aria-haspopup="true"
        aria-expanded={open}
      >
        {getCurrentIcon()}
      </IconButton>
    </Tooltip>
  );
}