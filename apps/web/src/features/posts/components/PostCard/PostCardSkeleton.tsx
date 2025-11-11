import { Card, CardHeader, CardContent, Skeleton } from '@mui/material';

export function PostCardSkeleton() {
  return (
    <Card sx={{ mb: 2 }}>
      <CardHeader
        avatar={<Skeleton variant="circular" width={40} height={40} />}
        title={<Skeleton width="30%" />}
        subheader={<Skeleton width="20%" />}
      />
      <CardContent>
        <Skeleton />
        <Skeleton />
        <Skeleton width="60%" />
      </CardContent>
    </Card>
  );
}


