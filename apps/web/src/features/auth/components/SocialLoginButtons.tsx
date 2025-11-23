'use client';

import { Button, Box, useTheme, useMediaQuery } from '@mui/material';
import { Google as GoogleIcon, GitHub as GitHubIcon } from '@mui/icons-material';

export interface SocialLoginButtonsProps {
  disabled?: boolean;
  layout?: 'row' | 'column' | 'responsive';
}

const SocialLoginButtons: React.FC<SocialLoginButtonsProps> = ({ 
  disabled = false,
  layout = 'responsive'
}) => {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('sm'));

  const handleGitHubLogin = () => {
    window.location.href = `/api/auth/login/github`;
  };

  const handleGoogleLogin = () => {
    window.location.href = `/api/auth/login/google`;
  };

  const flexDirection = layout === 'responsive' 
    ? (isMobile ? 'column' : 'row')
    : layout;

  return (
    <Box sx={{ display: 'flex', gap: 2, flexDirection }}>
      <Button
        fullWidth
        variant="outlined"
        startIcon={<GitHubIcon />}
        onClick={handleGitHubLogin}
        disabled={disabled}
        sx={{ py: 1.5 }}
      >
        GitHub
      </Button>
      
      <Button
        fullWidth
        variant="outlined"
        startIcon={<GoogleIcon />}
        onClick={handleGoogleLogin}
        disabled={disabled}
        sx={{ py: 1.5 }}
      >
        Google
      </Button>
    </Box>
  );
};

export default SocialLoginButtons;

