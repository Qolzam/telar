'use client';

import { Card, CardContent, Typography } from '@mui/material';

interface ProfilePostItemProps {
  post: any;
}

export function ProfilePostItem({ post }: ProfilePostItemProps) {
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


