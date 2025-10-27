'use client';

import { Card, CardContent, Typography } from '@mui/material';

interface ProfilePostItemProps {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  post: any;
}

export function ProfilePostItem({ post: _post }: ProfilePostItemProps) {
  return (
    <Card>
      <CardContent>
        <Typography variant="body2" color="text.secondary">
          Post feature coming soon
        </Typography>
      </CardContent>
    </Card>
  );
}


