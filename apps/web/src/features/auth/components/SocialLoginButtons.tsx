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
    const authApiUrl = process.env.NEXT_PUBLIC_AUTH_API_URL || 'http://localhost:8080';
    window.location.href = `${authApiUrl}/auth/login/github`;
  };

  const handleGoogleLogin = () => {
    const authApiUrl = process.env.NEXT_PUBLIC_AUTH_API_URL || 'http://localhost:8080';
    window.location.href = `${authApiUrl}/auth/login/google`;
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

