'use client';

import { usePlugins, usePlugin } from '@/plugins';
import { Box, Typography, Card, CardContent, Chip, Stack } from '@mui/material';

export default function PluginTestPage() {
  const { plugins, isLoading, isReady, errors } = usePlugins();
  const testPlugin = usePlugin('test-plugin');

  return (
    <Box sx={{ p: 4, maxWidth: 800, mx: 'auto' }}>
      <Typography variant="h4" gutterBottom>
        Plugin System Test
      </Typography>

      <Stack spacing={3}>
        {/* Status */}
        <Card>
          <CardContent>
            <Typography variant="h6" gutterBottom>
              System Status
            </Typography>
            <Stack direction="row" spacing={1} sx={{ mb: 2 }}>
              <Chip 
                label={isLoading ? 'Loading...' : 'Loaded'} 
                color={isLoading ? 'warning' : 'success'} 
              />
              <Chip 
                label={isReady ? 'Ready' : 'Not Ready'} 
                color={isReady ? 'success' : 'error'} 
              />
            </Stack>
            <Typography>
              Plugins Loaded: {plugins.length}
            </Typography>
            <Typography>
              Errors: {errors.length}
            </Typography>
          </CardContent>
        </Card>

        {/* Loaded Plugins */}
        <Card>
          <CardContent>
            <Typography variant="h6" gutterBottom>
              Loaded Plugins
            </Typography>
            {plugins.length === 0 && !isLoading && (
              <Typography color="text.secondary">
                No plugins loaded yet.
              </Typography>
            )}
            {plugins.map(plugin => (
              <Box key={plugin.metadata.id} sx={{ mb: 2 }}>
                <Typography variant="body1" fontWeight="bold">
                  {plugin.metadata.name}
                </Typography>
                <Typography variant="caption" display="block">
                  ID: {plugin.metadata.id} | Version: {plugin.metadata.version}
                </Typography>
                {plugin.metadata.description && (
                  <Typography variant="body2" color="text.secondary">
                    {plugin.metadata.description}
                  </Typography>
                )}
              </Box>
            ))}
          </CardContent>
        </Card>

        {/* Test Plugin Details */}
        {testPlugin && (
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                Test Plugin Details
              </Typography>
              <Typography gutterBottom>
                ✅ Test plugin is loaded and accessible!
              </Typography>
              <Box
                component="pre"
                sx={{
                  bgcolor: 'grey.100',
                  p: 2,
                  borderRadius: 1,
                  overflow: 'auto',
                  fontSize: '0.875rem',
                }}
              >
                {JSON.stringify(testPlugin.metadata, null, 2)}
              </Box>
            </CardContent>
          </Card>
        )}

        {/* Errors */}
        {errors.length > 0 && (
          <Card sx={{ bgcolor: 'error.light' }}>
            <CardContent>
              <Typography variant="h6" gutterBottom color="error">
                Errors ({errors.length})
              </Typography>
              {errors.map((error, index) => (
                <Typography key={index} variant="body2" color="error">
                  {error.message}
                </Typography>
              ))}
            </CardContent>
          </Card>
        )}
      </Stack>
    </Box>
  );
}

