/**
 * Top navigation bar with search, add post button, user profile and logout
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
  TextField,
  InputAdornment,
  Button,
  ClickAwayListener,
  Divider,
  List,
  ListItemAvatar,
  ListItemButton,
  ListItemText,
  ListSubheader,
  Paper,
  Stack,
} from '@mui/material';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useSession, useLogout } from '@/features/auth/client';
import NotificationsIcon from '@mui/icons-material/Notifications';
import SettingsIcon from '@mui/icons-material/Settings';
import LogoutIcon from '@mui/icons-material/Logout';
import AccountCircleIcon from '@mui/icons-material/AccountCircle';
import SearchIcon from '@mui/icons-material/Search';
import AddIcon from '@mui/icons-material/Add';
import { useRouter } from 'next/navigation';
import { PostDialog } from '@/features/posts/components';
import { useGlobalSearch } from '@/features/search';

export default function DashboardHeader() {
  const { t } = useTranslation('common');
  const { user, isAuthenticated, isLoading } = useSession();
  const { logout, isLoading: isLoggingOut } = useLogout();
  const router = useRouter();
  
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const [postDialogOpen, setPostDialogOpen] = useState(false);
  const [searchValue, setSearchValue] = useState('');
  const [isSearchOpen, setIsSearchOpen] = useState(false);
  const open = Boolean(anchorEl);

  const { profiles: searchProfiles, posts: searchPosts, isLoading: isSearchLoading } = useGlobalSearch(searchValue);
  const trimmedSearch = searchValue.trim();
  const showSearchDropdown = isSearchOpen && trimmedSearch.length > 0;

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

  const handleLogoutClick = () => {
    setAnchorEl(null);
    logout();
  };

  const handleOpenPostDialog = () => {
    setPostDialogOpen(true);
  };

  const handleClosePostDialog = () => {
    setPostDialogOpen(false);
  };

  const handleSearchChange = (value: string) => {
    setSearchValue(value);
    setIsSearchOpen(true);
  };

  const handleSearchClose = () => {
    setIsSearchOpen(false);
  };

  const handleSelectProfile = (id: string) => {
    setIsSearchOpen(false);
    setSearchValue('');
    router.push(`/profile/${id}`);
  };

  const handleSelectPost = (id: string) => {
    setIsSearchOpen(false);
    setSearchValue('');
    router.push(`/posts/${id}`);
  };

  return (
    <>
      <AppBar position="static" color="default" elevation={0}>
        <Toolbar sx={{ justifyContent: 'space-between', gap: 2, px: { xs: 2, sm: 3 } }}>
          {isLoading ? (
            <CircularProgress size={24} />
          ) : isAuthenticated && user ? (
            <>
              {/* Left side: Search and Add Post Button */}
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, flex: 1, maxWidth: { xs: '100%', md: '600px' } }}>
                {/* Search Input */}
                <ClickAwayListener onClickAway={handleSearchClose}>
                  <Box sx={{ position: 'relative', flex: 1, maxWidth: { xs: '100%', sm: '400px' } }}>
                    <TextField
                      placeholder="Search for people or posts"
                      value={searchValue}
                      onFocus={() => setIsSearchOpen(true)}
                      onChange={(e) => handleSearchChange(e.target.value)}
                      size="small"
                      sx={{
                        width: '100%',
                        '& .MuiOutlinedInput-root': {
                          borderRadius: '123px',
                          backgroundColor: 'background.paper',
                          height: '40px',
                          '& fieldset': {
                            borderColor: 'rgb(203, 213, 225)',
                          },
                          '&:hover fieldset': {
                            borderColor: 'rgb(203, 213, 225)',
                          },
                          '&.Mui-focused fieldset': {
                            borderColor: 'primary.main',
                          },
                        },
                        '& .MuiInputBase-input': {
                          py: 1,
                          fontSize: '14px',
                          fontWeight: 500,
                          letterSpacing: '-0.112px',
                        },
                      }}
                      InputProps={{
                        endAdornment: (
                          <InputAdornment position="end">
                            {isSearchLoading ? (
                              <CircularProgress size={18} />
                            ) : (
                              <SearchIcon sx={{ color: 'rgb(71, 85, 105)', fontSize: 18 }} />
                            )}
                          </InputAdornment>
                        ),
                      }}
                    />

                    {showSearchDropdown && (
                      <Paper
                        elevation={4}
                        sx={{
                          position: 'absolute',
                          top: '46px',
                          left: 0,
                          right: 0,
                          zIndex: 10,
                          borderRadius: 2,
                          overflow: 'hidden',
                        }}
                      >
                        {trimmedSearch.length < 3 ? (
                          <Box sx={{ px: 2, py: 1.5 }}>
                            <Typography variant="body2" color="text.secondary">
                              Keep typing to search (min 3 characters)
                            </Typography>
                          </Box>
                        ) : (
                          <Stack spacing={1}>
                            <Box>
                              <List
                                dense
                                subheader={
                                  <ListSubheader component="div" disableSticky>
                                    People
                                  </ListSubheader>
                                }
                              >
                                {searchProfiles.length === 0 && !isSearchLoading ? (
                                  <ListItemText sx={{ px: 2, py: 1 }} primary="No people found" />
                                ) : (
                                  searchProfiles.map((profile) => (
                                    <ListItemButton key={profile.objectId} onClick={() => handleSelectProfile(profile.objectId)}>
                                      <ListItemAvatar>
                                        <Avatar src={profile.avatar} alt={profile.fullName || profile.socialName} sx={{ width: 32, height: 32 }} />
                                      </ListItemAvatar>
                                      <ListItemText
                                        primary={profile.fullName || profile.socialName}
                                        secondary={`@${profile.socialName}`}
                                        primaryTypographyProps={{ fontWeight: 600, fontSize: 14 }}
                                        secondaryTypographyProps={{ fontSize: 12 }}
                                      />
                                    </ListItemButton>
                                  ))
                                )}
                              </List>
                            </Box>

                            <Divider />

                            <Box>
                              <List
                                dense
                                subheader={
                                  <ListSubheader component="div" disableSticky>
                                    Posts
                                  </ListSubheader>
                                }
                              >
                                {searchPosts.length === 0 && !isSearchLoading ? (
                                  <ListItemText sx={{ px: 2, py: 1 }} primary="No posts found" />
                                ) : (
                                  searchPosts.map((post) => (
                                    <ListItemButton key={post.objectId} onClick={() => handleSelectPost(post.objectId)}>
                                      <ListItemAvatar>
                                        <Avatar src={post.ownerAvatar} alt={post.ownerDisplayName} sx={{ width: 32, height: 32 }} />
                                      </ListItemAvatar>
                                      <ListItemText
                                        primary={post.ownerDisplayName}
                                        secondary={
                                          post.body.length > 120 ? `${post.body.slice(0, 120)}…` : post.body || 'Post'
                                        }
                                        primaryTypographyProps={{ fontWeight: 600, fontSize: 14 }}
                                        secondaryTypographyProps={{ fontSize: 12 }}
                                      />
                                    </ListItemButton>
                                  ))
                                )}
                              </List>
                            </Box>
                          </Stack>
                        )}
                      </Paper>
                    )}
                  </Box>
                </ClickAwayListener>

                {/* Add New Post Button */}
                <Button
                  variant="contained"
                  onClick={handleOpenPostDialog}
                  startIcon={<AddIcon sx={{ fontSize: 18 }} />}
                  size="small"
                  sx={{
                    borderRadius: '1234px',
                    height: '40px',
                    px: 2,
                    backgroundColor: 'rgb(79, 70, 229)',
                    color: 'white',
                    fontWeight: 600,
                    fontSize: '14px',
                    letterSpacing: '-0.112px',
                    textTransform: 'none',
                    '&:hover': {
                      backgroundColor: 'rgb(67, 56, 202)',
                    },
                    display: { xs: 'none', sm: 'flex' },
                  }}
                >
                  Add New Post
                </Button>
              </Box>

              {/* Right side: Notifications, Settings, User Profile */}
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
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

      {/* Post Dialog */}
      <PostDialog open={postDialogOpen} onClose={handleClosePostDialog} />
    </>
  );
}
