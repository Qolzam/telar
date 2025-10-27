
'use client';

import React from 'react';
import { Box, Typography, useTheme, useMediaQuery } from '@mui/material';

export interface AuthLayoutProps {
  /** Content to render in the right panel (form area) */
  children: React.ReactNode;
  /** Title to display in the left panel */
  title: string;
  /** Subtitle/description to display in the left panel */
  subtitle: string;
  /** Illustration content for the left panel */
  illustration?: React.ReactNode;
}

const AuthLayout: React.FC<AuthLayoutProps> = ({
  children,
  title,
  subtitle,
  illustration
}) => {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('md'));

  const defaultIllustration = (
    <Box
      sx={{
        width: 300,
        height: 200,
        mx: 'auto',
        backgroundColor: 'rgba(255, 255, 255, 0.1)',
        borderRadius: 3,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        border: '2px dashed rgba(255, 255, 255, 0.3)',
        position: 'relative',
        overflow: 'hidden',
      }}
    >
      <Box
        sx={{
          position: 'absolute',
          top: '20%',
          left: '20%',
          width: 40,
          height: 40,
          borderRadius: '50%',
          backgroundColor: 'rgba(255, 255, 255, 0.3)',
        }}
      />
      <Box
        sx={{
          position: 'absolute',
          bottom: '30%',
          right: '25%',
          width: 30,
          height: 30,
          borderRadius: '4px',
          backgroundColor: 'rgba(255, 255, 255, 0.25)',
          transform: 'rotate(45deg)',
        }}
      />
      <Box
        sx={{
          position: 'absolute',
          top: '50%',
          left: '50%',
          transform: 'translate(-50%, -50%)',
          width: 60,
          height: 60,
          borderRadius: '50%',
          border: '3px solid rgba(255, 255, 255, 0.4)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        }}
      >
        <Box
          sx={{
            width: 20,
            height: 20,
            borderRadius: '50%',
            backgroundColor: 'rgba(255, 255, 255, 0.6)',
          }}
        />
      </Box>
    </Box>
  );

  return (
    <Box sx={{ minHeight: '100vh', display: 'flex', backgroundColor: theme.palette.background.default }}>
      <Box sx={{ display: 'flex', width: '100%', minHeight: '100vh' }}>
        {/* Left Panel - Branding/Illustration */}
        {!isMobile && (
          <Box
            sx={{
              flex: '1 1 58%', // ~7/12 width (lg=7)
              background: `linear-gradient(135deg, ${theme.palette.primary.main} 0%, ${theme.palette.secondary.main} 100%)`,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              position: 'relative',
              overflow: 'hidden',
            }}
          >
            {/* Floating Animation Elements */}
            <Box
              sx={{
                position: 'absolute',
                top: '20%',
                left: '10%',
                width: 60,
                height: 60,
                borderRadius: '50%',
                backgroundColor: 'rgba(255, 255, 255, 0.1)',
                animation: 'float 6s ease-in-out infinite',
                '@keyframes float': {
                  '0%, 100%': { transform: 'translateY(0px)' },
                  '50%': { transform: 'translateY(-20px)' },
                },
              }}
            />
            <Box
              sx={{
                position: 'absolute',
                bottom: '20%',
                right: '15%',
                width: 80,
                height: 80,
                borderRadius: '50%',
                backgroundColor: 'rgba(255, 255, 255, 0.08)',
                animation: 'float 8s ease-in-out infinite reverse',
              }}
            />
            
            {/* Main Content */}
            <Box sx={{ textAlign: 'center', px: 4, zIndex: 1 }}>
              <Typography
                variant="h2"
                component="h1"
                sx={{
                  color: 'white',
                  fontWeight: 'bold',
                  mb: 3,
                  fontSize: { md: '3rem', lg: '3.5rem' },
                }}
              >
                {title}
              </Typography>
              <Typography
                variant="h5"
                sx={{
                  color: 'rgba(255, 255, 255, 0.9)',
                  mb: 4,
                  maxWidth: 400,
                  mx: 'auto',
                  lineHeight: 1.6,
                }}
              >
                {subtitle}
              </Typography>
              
              {illustration || defaultIllustration}
            </Box>
          </Box>
        )}
        
        {/* Right Panel - Form Content */}
        <Box
          sx={{
            flex: isMobile ? '1 1 100%' : '1 1 42%', // ~5/12 width (lg=5), full width on mobile
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            backgroundColor: theme.palette.mode === 'dark' 
              ? theme.palette.background.default 
              : theme.palette.background.paper,
            p: { xs: 3, sm: 4, md: 5 },
          }}
        >
          <Box
            sx={{
              width: '100%',
              maxWidth: 480,
              mx: 2,
            }}
          >
            {children}
          </Box>
        </Box>
      </Box>
    </Box>
  );
};

export default AuthLayout;
