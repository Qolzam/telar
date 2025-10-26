'use client';

import { useCallback } from 'react';
import {
  AppBar,
  Toolbar,
  Typography,
  IconButton,
  Stack,
  useTheme,
  useMediaQuery,
} from '@mui/material';
import {
  Menu as MenuIcon,
  AccountCircle as AccountIcon,
} from '@mui/icons-material';
import { ThemeToggle } from '../ThemeToggle';

interface HeaderProps {
  title?: string;
  onMenuClick?: () => void;
  onAccountClick?: () => void;
  className?: string;
  elevation?: number;
}

export function Header({ 
  title = 'Telar Social', 
  onMenuClick,
  onAccountClick,
  className,
  elevation = 1
}: HeaderProps) {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('md'));

  const handleMenuClick = useCallback(() => {
    onMenuClick?.();
  }, [onMenuClick]);

  const handleAccountClick = useCallback(() => {
    onAccountClick?.();
  }, [onAccountClick]);

  return (
    <AppBar 
      position="static" 
      elevation={elevation}
      className={className}
      sx={{
        zIndex: theme.zIndex.appBar,
      }}
    >
      <Toolbar>
        {onMenuClick && (
          <IconButton
            edge="start"
            color="inherit"
            aria-label="Open navigation menu"
            onClick={handleMenuClick}
            sx={{ 
              mr: 2,
              display: { xs: 'flex', md: 'flex' }
            }}
          >
            <MenuIcon />
          </IconButton>
        )}

        <Typography 
          variant={isMobile ? "h6" : "h5"} 
          component="div" 
          sx={{ 
            flexGrow: 1,
            fontWeight: 600,
            letterSpacing: '-0.025em',
          }}
        >
          {title}
        </Typography>

        <Stack 
          direction="row" 
          spacing={1} 
          alignItems="center"
          sx={{ ml: 2 }}
        >
          <ThemeToggle 
            variant="icon" 
            size={isMobile ? "small" : "medium"}
            aria-label="Toggle theme"
          />
          
          {onAccountClick && (
            <IconButton
              color="inherit"
              aria-label="Account menu"
              onClick={handleAccountClick}
              size={isMobile ? "small" : "medium"}
            >
              <AccountIcon />
            </IconButton>
          )}
        </Stack>
      </Toolbar>
    </AppBar>
  );
}