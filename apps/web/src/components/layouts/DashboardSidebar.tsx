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
} from '@mui/material';
import { useRouter, usePathname } from 'next/navigation';
import HomeIcon from '@mui/icons-material/Home';
import PersonIcon from '@mui/icons-material/Person';
import PostAddIcon from '@mui/icons-material/PostAdd';
import ChatIcon from '@mui/icons-material/Chat';
import GroupIcon from '@mui/icons-material/Group';
import SearchIcon from '@mui/icons-material/Search';

const DRAWER_WIDTH = 240;

const MENU_ITEMS = [
  { label: 'Home', path: '/dashboard', icon: <HomeIcon /> },
  { label: 'Profile', path: '/profile', icon: <PersonIcon /> },
  { label: 'Posts', path: '/posts', icon: <PostAddIcon /> },
  { label: 'Messages', path: '/messages', icon: <ChatIcon /> },
  { label: 'Connections', path: '/connections', icon: <GroupIcon /> },
  { label: 'Search', path: '/search', icon: <SearchIcon /> },
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
      <Box sx={{ overflow: 'auto' }}>
        {/* Logo/Title */}
        <Box sx={{ p: 2, textAlign: 'center' }}>
          <h2 style={{ margin: 0, color: '#1976d2' }}>Telar</h2>
        </Box>

        <Divider />

        {/* Navigation Menu */}
        <List>
          {MENU_ITEMS.map((item) => (
            <ListItem key={item.path} disablePadding>
              <ListItemButton
                selected={pathname === item.path}
                onClick={() => handleNavigation(item.path)}
              >
                <ListItemIcon>{item.icon}</ListItemIcon>
                <ListItemText primary={item.label} />
              </ListItemButton>
            </ListItem>
          ))}
        </List>
      </Box>
    </Drawer>
  );
}
