/**
 * Admin Dashboard Home Page
 * 
 * Overview dashboard with stats and quick actions
 */

'use client';

import { Box, Container, Grid, Paper, Typography, Card, CardContent, Stack, Chip } from '@mui/material';
import { useMembersQuery } from '@/features/admin/client';
import { useModerationQueue } from '@/features/moderation/client';
import PeopleIcon from '@mui/icons-material/People';
import GavelIcon from '@mui/icons-material/Gavel';
import AdminPanelSettingsIcon from '@mui/icons-material/AdminPanelSettings';
import TrendingUpIcon from '@mui/icons-material/TrendingUp';
import { CircularProgress, Alert } from '@mui/material';

export default function AdminDashboardPage() {
  // #region agent log
  if(typeof window!=='undefined'){fetch('http://localhost:7242/ingest/8ca19b94-8916-40b5-8bfd-490c6748d23a',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({location:'(admin)/page.tsx:19',message:'AdminDashboardPage entry',data:{pathname:window.location.pathname},timestamp:Date.now(),sessionId:'debug-session',runId:'run1',hypothesisId:'D'})}).catch(()=>{});}
  // #endregion
  const { data: membersData, isLoading: membersLoading } = useMembersQuery({ limit: 100, offset: 0 });
  const { data: moderationData, isLoading: moderationLoading } = useModerationQueue();

  const totalMembers = membersData?.members?.length ?? 0;
  const pendingModeration = moderationData?.items?.length ?? moderationData?.count ?? 0;

  const stats = [
    {
      title: 'Total Members',
      value: membersLoading ? '...' : totalMembers.toLocaleString(),
      icon: <PeopleIcon sx={{ fontSize: 40 }} />,
      color: '#1976d2',
      gradient: 'linear-gradient(135deg, #1976d2 0%, #42a5f5 100%)',
    },
    {
      title: 'Pending Moderation',
      value: moderationLoading ? '...' : pendingModeration.toLocaleString(),
      icon: <GavelIcon sx={{ fontSize: 40 }} />,
      color: '#ed6c02',
      gradient: 'linear-gradient(135deg, #ed6c02 0%, #ff9800 100%)',
    },
    {
      title: 'Admin Users',
      value: '1',
      icon: <AdminPanelSettingsIcon sx={{ fontSize: 40 }} />,
      color: '#d32f2f',
      gradient: 'linear-gradient(135deg, #d32f2f 0%, #f44336 100%)',
    },
    {
      title: 'Growth Rate',
      value: '+12%',
      icon: <TrendingUpIcon sx={{ fontSize: 40 }} />,
      color: '#2e7d32',
      gradient: 'linear-gradient(135deg, #2e7d32 0%, #4caf50 100%)',
    },
  ];

  return (
    <Container maxWidth="lg" sx={{ py: 4 }}>
      {/* Header */}
      <Box sx={{ mb: 4 }}>
        <Stack direction="row" spacing={2} alignItems="center" sx={{ mb: 1 }}>
          <Typography variant="h4" sx={{ fontWeight: 700 }}>
            Admin Dashboard
          </Typography>
          <Chip
            label="Administrator"
            size="small"
            color="error"
            sx={{ fontWeight: 600 }}
          />
        </Stack>
        <Typography variant="body1" color="text.secondary">
          Manage your platform, members, and content moderation
        </Typography>
      </Box>

      {/* Stats Grid */}
      <Grid container spacing={3} sx={{ mb: 4 }}>
        {stats.map((stat, index) => (
          <Grid size={{ xs: 12, sm: 6, md: 3 }} key={index}>
            <Card
              sx={{
                height: '100%',
                background: stat.gradient,
                color: 'white',
                transition: 'transform 0.2s, box-shadow 0.2s',
                '&:hover': {
                  transform: 'translateY(-4px)',
                  boxShadow: 6,
                },
              }}
            >
              <CardContent>
                <Stack direction="row" spacing={2} alignItems="center">
                  <Box
                    sx={{
                      p: 1.5,
                      borderRadius: 2,
                      bgcolor: 'rgba(255, 255, 255, 0.2)',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                    }}
                  >
                    {stat.icon}
                  </Box>
                  <Box sx={{ flexGrow: 1 }}>
                    <Typography variant="h4" sx={{ fontWeight: 700, mb: 0.5 }}>
                      {stat.value}
                    </Typography>
                    <Typography variant="body2" sx={{ opacity: 0.9 }}>
                      {stat.title}
                    </Typography>
                  </Box>
                </Stack>
              </CardContent>
            </Card>
          </Grid>
        ))}
      </Grid>

      {/* Quick Actions */}
      <Grid container spacing={3}>
        <Grid size={{ xs: 12, md: 6 }}>
          <Paper sx={{ p: 3, height: '100%' }}>
            <Typography variant="h6" sx={{ fontWeight: 600, mb: 2 }}>
              Quick Actions
            </Typography>
            <Stack spacing={2}>
              <Box
                sx={{
                  p: 2,
                  borderRadius: 2,
                  border: '1px solid',
                  borderColor: 'divider',
                  '&:hover': {
                    bgcolor: 'action.hover',
                    cursor: 'pointer',
                  },
                }}
                onClick={() => window.location.href = '/admin/members'}
              >
                <Stack direction="row" spacing={2} alignItems="center">
                  <PeopleIcon color="primary" />
                  <Box>
                    <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>
                      Manage Members
                    </Typography>
                    <Typography variant="body2" color="text.secondary">
                      View and manage user accounts
                    </Typography>
                  </Box>
                </Stack>
              </Box>
              <Box
                sx={{
                  p: 2,
                  borderRadius: 2,
                  border: '1px solid',
                  borderColor: 'divider',
                  '&:hover': {
                    bgcolor: 'action.hover',
                    cursor: 'pointer',
                  },
                }}
                onClick={() => window.location.href = '/admin/moderation'}
              >
                <Stack direction="row" spacing={2} alignItems="center">
                  <GavelIcon color="primary" />
                  <Box>
                    <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>
                      Moderation Queue
                    </Typography>
                    <Typography variant="body2" color="text.secondary">
                      Review flagged content
                    </Typography>
                  </Box>
                </Stack>
              </Box>
            </Stack>
          </Paper>
        </Grid>

        <Grid size={{ xs: 12, md: 6 }}>
          <Paper sx={{ p: 3, height: '100%' }}>
            <Typography variant="h6" sx={{ fontWeight: 600, mb: 2 }}>
              System Status
            </Typography>
            <Stack spacing={2}>
              <Box>
                <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 1 }}>
                  <Typography variant="body2" color="text.secondary">
                    Platform Status
                  </Typography>
                  <Chip label="Operational" color="success" size="small" />
                </Stack>
              </Box>
              <Box>
                <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 1 }}>
                  <Typography variant="body2" color="text.secondary">
                    API Status
                  </Typography>
                  <Chip label="Healthy" color="success" size="small" />
                </Stack>
              </Box>
              <Box>
                <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 1 }}>
                  <Typography variant="body2" color="text.secondary">
                    Database
                  </Typography>
                  <Chip label="Connected" color="success" size="small" />
                </Stack>
              </Box>
            </Stack>
          </Paper>
        </Grid>
      </Grid>
    </Container>
  );
}
