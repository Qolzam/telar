'use client';

import { useState, useEffect, useMemo } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  IconButton,
  Box,
  Stack,
  Button,
  Typography,
  TextField,
  InputAdornment,
  Snackbar,
  Alert,
  Divider,
  useTheme,
  alpha,
} from '@mui/material';
import {
  Close,
  ContentCopy,
  Share as ShareIcon,
  Twitter,
  Facebook,
  LinkedIn,
  Email,
} from '@mui/icons-material';
import { useTranslation } from 'react-i18next';
import { sdk } from '@/lib/sdk';
import type { Post } from '@telar/sdk';

interface ShareDialogProps {
  open: boolean;
  onClose: () => void;
  post: Post;
}

export function ShareDialog({ open, onClose, post }: ShareDialogProps) {
  const theme = useTheme();
  const { t } = useTranslation('posts');
  const [shareUrl, setShareUrl] = useState<string>('');
  const [isGenerating, setIsGenerating] = useState(false);
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const isDarkMode = useMemo(() => {
    if (theme.palette.mode === 'dark') return true;
    if (typeof document !== 'undefined') {
      const scheme = document.documentElement.getAttribute('data-mui-color-scheme') 
        || document.documentElement.getAttribute('data-color-scheme');
      return scheme === 'dark';
    }
    return false;
  }, [theme.palette.mode]);
  
  const darkCardBackground = '#0f172a';
  const darkBorder = '#1f2937';
  const darkTextPrimary = '#e2e8f0';
  const darkTextSecondary = '#94a3b8';
  const dialogBg = isDarkMode ? darkCardBackground : theme.palette.background.paper;
  const borderColor = isDarkMode ? darkBorder : theme.palette.divider;
  const textPrimary = isDarkMode ? darkTextPrimary : theme.palette.text.primary;
  const textSecondary = isDarkMode ? darkTextSecondary : theme.palette.text.secondary;

  useEffect(() => {
    if (open && post) {
      generateShareUrl();
    }
  }, [open, post]);

  const generateShareUrl = async () => {
    setIsGenerating(true);
    setError(null);
    
    try {
      let urlKey = post.urlKey;
      
      if (!urlKey) {
        const response = await sdk.posts.generateUrlKey(post.objectId);
        urlKey = response.urlKey;
      }
      
      const baseUrl = typeof window !== 'undefined' ? window.location.origin : '';
      const shareableUrl = `${baseUrl}/posts/${urlKey}`;
      setShareUrl(shareableUrl);
    } catch (err) {
      console.error('Failed to generate share URL:', err);
      setError(t('share.error.generateFailed'));
      const fallbackUrl = typeof window !== 'undefined' 
        ? `${window.location.origin}/posts/${post.objectId}`
        : '';
      setShareUrl(fallbackUrl);
    } finally {
      setIsGenerating(false);
    }
  };

  const handleCopyLink = async () => {
    try {
      await navigator.clipboard.writeText(shareUrl);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error('Failed to copy:', err);
      setError(t('share.error.copyFailed'));
    }
  };

  const handleNativeShare = async () => {
    if (navigator.share) {
      try {
        await navigator.share({
          title: `Post by ${post.ownerDisplayName}`,
          text: post.body.substring(0, 100),
          url: shareUrl,
        });
      } catch (err) {
        if ((err as Error).name !== 'AbortError') {
          console.error('Share failed:', err);
        }
      }
    }
  };

  const handleSocialShare = (platform: 'twitter' | 'facebook' | 'linkedin' | 'email') => {
    const encodedUrl = encodeURIComponent(shareUrl);
    const encodedText = encodeURIComponent(post.body.substring(0, 200));
    const encodedTitle = encodeURIComponent(`Post by ${post.ownerDisplayName}`);

    let shareUrlPlatform = '';

    switch (platform) {
      case 'twitter':
        shareUrlPlatform = `https://twitter.com/intent/tweet?url=${encodedUrl}&text=${encodedText}`;
        break;
      case 'facebook':
        shareUrlPlatform = `https://www.facebook.com/sharer/sharer.php?u=${encodedUrl}`;
        break;
      case 'linkedin':
        shareUrlPlatform = `https://www.linkedin.com/sharing/share-offsite/?url=${encodedUrl}`;
        break;
      case 'email':
        shareUrlPlatform = `mailto:?subject=${encodedTitle}&body=${encodedText}%20${encodedUrl}`;
        break;
    }

    if (shareUrlPlatform) {
      window.open(shareUrlPlatform, '_blank', 'noopener,noreferrer');
    }
  };


  return (
    <>
      <Dialog
        open={open}
        onClose={onClose}
        maxWidth="sm"
        fullWidth
        PaperProps={{
          sx: {
            borderRadius: '16px',
            backgroundColor: dialogBg,
            border: `1px solid ${borderColor}`,
          },
        }}
      >
        <DialogTitle
          sx={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            pb: 1,
            borderBottom: `1px solid ${borderColor}`,
            fontWeight: 600,
            color: textPrimary,
          }}
        >
          {t('share.title')}
          <IconButton
            onClick={onClose}
            size="small"
            sx={{
              color: textSecondary,
              '&:hover': {
                backgroundColor: alpha(textPrimary, 0.08),
              },
            }}
          >
            <Close />
          </IconButton>
        </DialogTitle>

        <DialogContent sx={{ pt: 3 }}>
          <Stack spacing={3}>
            <Box>
              <Typography variant="body2" sx={{ mb: 1.5, color: textSecondary }}>
                {t('share.copyLink')}
              </Typography>
              <TextField
                fullWidth
                value={shareUrl}
                disabled={isGenerating}
                size="small"
                InputProps={{
                  readOnly: true,
                  endAdornment: (
                    <InputAdornment position="end">
                      <IconButton
                        onClick={handleCopyLink}
                        disabled={!shareUrl || isGenerating}
                        size="small"
                        sx={{
                          color: copied ? theme.palette.success.main : theme.palette.primary.main,
                        }}
                      >
                        <ContentCopy fontSize="small" />
                      </IconButton>
                    </InputAdornment>
                  ),
                }}
                sx={{
                  '& .MuiOutlinedInput-root': {
                    backgroundColor: alpha(textPrimary, 0.04),
                    color: textPrimary,
                    '& fieldset': {
                      borderColor: borderColor,
                    },
                    '&:hover fieldset': {
                      borderColor: borderColor,
                    },
                    '&.Mui-focused fieldset': {
                      borderColor: theme.palette.primary.main,
                    },
                  },
                }}
              />
              {copied && (
                <Typography variant="caption" color="success.main" sx={{ mt: 0.5, display: 'block' }}>
                  {t('share.linkCopied')}
                </Typography>
              )}
            </Box>

            {typeof navigator !== 'undefined' && 'share' in navigator && (
              <>
                <Divider sx={{ borderColor: borderColor }} />
                <Box>
                  <Typography variant="body2" sx={{ mb: 1.5, color: textSecondary }}>
                    {t('share.shareVia')}
                  </Typography>
                  <Button
                    fullWidth
                    variant="outlined"
                    startIcon={<ShareIcon />}
                    onClick={handleNativeShare}
                    sx={{
                      justifyContent: 'flex-start',
                      textTransform: 'none',
                      py: 1.5,
                      borderColor: borderColor,
                      color: textPrimary,
                      '&:hover': {
                        borderColor: theme.palette.primary.main,
                        backgroundColor: alpha(theme.palette.primary.main, 0.08),
                      },
                    }}
                  >
                    {t('share.shareViaDevice')}
                  </Button>
                </Box>
              </>
            )}

            <Divider sx={{ borderColor: borderColor }} />

            <Box>
              <Typography variant="body2" sx={{ mb: 1.5, color: textSecondary }}>
                {t('share.shareToSocial')}
              </Typography>
              <Stack direction="row" spacing={1.5} flexWrap="wrap" useFlexGap>
                <Button
                  variant="outlined"
                  startIcon={<Twitter />}
                  onClick={() => handleSocialShare('twitter')}
                  sx={{
                    flex: { xs: '1 1 calc(50% - 6px)', sm: 'none' },
                    textTransform: 'none',
                    borderColor: borderColor,
                    color: textPrimary,
                    '&:hover': {
                      borderColor: '#1DA1F2',
                      backgroundColor: alpha('#1DA1F2', 0.08),
                      color: '#1DA1F2',
                    },
                  }}
                >
                  Twitter
                </Button>
                <Button
                  variant="outlined"
                  startIcon={<Facebook />}
                  onClick={() => handleSocialShare('facebook')}
                  sx={{
                    flex: { xs: '1 1 calc(50% - 6px)', sm: 'none' },
                    textTransform: 'none',
                    borderColor: borderColor,
                    color: textPrimary,
                    '&:hover': {
                      borderColor: '#1877F2',
                      backgroundColor: alpha('#1877F2', 0.08),
                      color: '#1877F2',
                    },
                  }}
                >
                  Facebook
                </Button>
                <Button
                  variant="outlined"
                  startIcon={<LinkedIn />}
                  onClick={() => handleSocialShare('linkedin')}
                  sx={{
                    flex: { xs: '1 1 calc(50% - 6px)', sm: 'none' },
                    textTransform: 'none',
                    borderColor: borderColor,
                    color: textPrimary,
                    '&:hover': {
                      borderColor: '#0A66C2',
                      backgroundColor: alpha('#0A66C2', 0.08),
                      color: '#0A66C2',
                    },
                  }}
                >
                  LinkedIn
                </Button>
                <Button
                  variant="outlined"
                  startIcon={<Email />}
                  onClick={() => handleSocialShare('email')}
                  sx={{
                    flex: { xs: '1 1 calc(50% - 6px)', sm: 'none' },
                    textTransform: 'none',
                    borderColor: borderColor,
                    color: textPrimary,
                    '&:hover': {
                      borderColor: theme.palette.primary.main,
                      backgroundColor: alpha(theme.palette.primary.main, 0.08),
                      color: theme.palette.primary.main,
                    },
                  }}
                >
                  Email
                </Button>
              </Stack>
            </Box>
          </Stack>
        </DialogContent>
      </Dialog>

      <Snackbar
        open={!!error}
        autoHideDuration={4000}
        onClose={() => setError(null)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        <Alert onClose={() => setError(null)} severity="error" variant="filled">
          {error}
        </Alert>
      </Snackbar>
    </>
  );
}

