/**
 * Side navigation for main app sections
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
} from '@mui/material';
import { useRouter, usePathname } from 'next/navigation';
import HomeIcon from '@mui/icons-material/Home';
import PersonIcon from '@mui/icons-material/Person';
import PostAddIcon from '@mui/icons-material/PostAdd';
import ChatIcon from '@mui/icons-material/Chat';
import GroupIcon from '@mui/icons-material/Group';
import SearchIcon from '@mui/icons-material/Search';
import SettingsIcon from '@mui/icons-material/Settings';

const DRAWER_WIDTH = 240;

const MENU_ITEMS = [
  { label: 'Home', path: '/dashboard', icon: <HomeIcon /> },
  { label: 'Profile', path: '/profile', icon: <PersonIcon /> },
  { label: 'Posts', path: '/posts', icon: <PostAddIcon /> },
  { label: 'Messages', path: '/messages', icon: <ChatIcon /> },
  { label: 'Connections', path: '/connections', icon: <GroupIcon /> },
  { label: 'Search', path: '/search', icon: <SearchIcon /> },
  { label: 'Settings', path: '/settings', icon: <SettingsIcon /> },
] as const;

export default function DashboardSidebar() {
  const router = useRouter();
  const pathname = usePathname();

  const handleNavigation = (path: string) => {
    router.push(path);
  };

  return (
    <Drawer
      variant="permanent"
      sx={{
        width: DRAWER_WIDTH,
        flexShrink: 0,
        '& .MuiDrawer-paper': {
          width: DRAWER_WIDTH,
          boxSizing: 'border-box',
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
            {/* Icon/Shape - Connected People/Social Network */}
            <Box
              sx={{
                width: 48,
                height: 48,
                borderRadius: 2,
                background: 'linear-gradient(135deg, #1976d2 0%, #42a5f5 100%)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                boxShadow: '0 4px 12px rgba(25, 118, 210, 0.3)',
                position: 'relative',
                '&::after': {
                  content: '""',
                  position: 'absolute',
                  top: -2,
                  left: -2,
                  right: -2,
                  bottom: -2,
                  borderRadius: 2,
                  background: 'linear-gradient(135deg, #1976d2 0%, #42a5f5 100%)',
                  opacity: 0.3,
                  filter: 'blur(8px)',
                  zIndex: -1,
                },
              }}
            >
              {/* Connected Nodes - Symbolizing Social Network */}
              <Box
                component="svg"
                width={32}
                height={32}
                viewBox="0 0 32 32"
                fill="none"
                sx={{ color: 'white' }}
              >
                {/* Connecting Lines */}
                <path
                  d="M8 16 L24 16"
                  stroke="currentColor"
                  strokeWidth={1.5}
                  strokeLinecap="round"
                  opacity={0.3}
                />
                <path
                  d="M16 8 L16 24"
                  stroke="currentColor"
                  strokeWidth={1.5}
                  strokeLinecap="round"
                  opacity={0.3}
                />
                <path
                  d="M8 8 L24 24"
                  stroke="currentColor"
                  strokeWidth={1.5}
                  strokeLinecap="round"
                  opacity={0.2}
                />
                <path
                  d="M24 8 L8 24"
                  stroke="currentColor"
                  strokeWidth={1.5}
                  strokeLinecap="round"
                  opacity={0.2}
                />
                
                {/* Central Hub */}
                <circle
                  cx="16"
                  cy="16"
                  r="4"
                  fill="currentColor"
                />
                
                {/* Peripheral Nodes */}
                <circle
                  cx="16"
                  cy="8"
                  r="2.5"
                  fill="currentColor"
                  opacity={0.9}
                />
                <circle
                  cx="24"
                  cy="16"
                  r="2.5"
                  fill="currentColor"
                  opacity={0.9}
                />
                <circle
                  cx="16"
                  cy="24"
                  r="2.5"
                  fill="currentColor"
                  opacity={0.9}
                />
                <circle
                  cx="8"
                  cy="16"
                  r="2.5"
                  fill="currentColor"
                  opacity={0.9}
                />
                
                {/* Corner Nodes */}
                <circle
                  cx="8"
                  cy="8"
                  r="2"
                  fill="currentColor"
                  opacity={0.7}
                />
                <circle
                  cx="24"
                  cy="8"
                  r="2"
                  fill="currentColor"
                  opacity={0.7}
                />
                <circle
                  cx="8"
                  cy="24"
                  r="2"
                  fill="currentColor"
                  opacity={0.7}
                />
                <circle
                  cx="24"
                  cy="24"
                  r="2"
                  fill="currentColor"
                  opacity={0.7}
                />
              </Box>
            </Box>
          </Box>
          <Typography
            variant="h5"
            component="div"
            sx={{
              fontWeight: 700,
              background: 'linear-gradient(135deg, #1976d2 0%, #42a5f5 100%)',
              WebkitBackgroundClip: 'text',
              WebkitTextFillColor: 'transparent',
              backgroundClip: 'text',
              letterSpacing: '-0.02em',
            }}
          >
            Telar
          </Typography>
          <Typography
            variant="caption"
            sx={{
              color: 'text.secondary',
              fontSize: '0.7rem',
              fontWeight: 500,
              letterSpacing: '0.1em',
            }}
          >
            SOCIAL NETWORK
          </Typography>
        </Box>

        <Divider />

        {/* Navigation Menu */}
        <List sx={{ flexGrow: 1, pt: 1 }}>
          {MENU_ITEMS.map((item) => (
            <ListItem key={item.path} disablePadding>
              <ListItemButton
                selected={pathname === item.path}
                onClick={() => handleNavigation(item.path)}
                sx={{
                  mx: 1,
                  borderRadius: 1,
                  '&.Mui-selected': {
                    bgcolor: 'primary.main',
                    color: 'primary.contrastText',
                    '&:hover': {
                      bgcolor: 'primary.dark',
                    },
                    '& .MuiListItemIcon-root': {
                      color: 'primary.contrastText',
                    },
                  },
                }}
              >
                <ListItemIcon
                  sx={{
                    minWidth: 40,
                    color: 'inherit',
                  }}
                >
                  {item.icon}
                </ListItemIcon>
                <ListItemText primary={item.label} />
              </ListItemButton>
            </ListItem>
          ))}
        </List>
      </Box>
    </Drawer>
  );
}
