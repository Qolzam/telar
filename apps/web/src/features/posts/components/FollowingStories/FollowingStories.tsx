'use client';

import { Avatar, Box, Typography } from '@mui/material';
import { useSession } from '@/features/auth/client';

type UserStatus = 'online' | 'away' | 'offline';

type FollowingStoryUser = {
  id: string;
  name: string;
  handle: string;
  avatar?: string;
  status: UserStatus;
};

const STORY_RING_GRADIENT =
  'conic-gradient(#4F46E5 0deg, #C622FF 120deg, #FF2222 240deg, #FFA439 360deg)';

const STATUS_COLORS: Record<UserStatus, string> = {
  online: '#22C55E',
  away: '#F59E0B',
  offline: '#94A3B8',
};

const DEMO_STORIES: FollowingStoryUser[] = [
  { id: '1', name: 'Alice Johnson', handle: 'x_ae-23b', status: 'online' },
  { id: '2', name: 'Mai Senpai', handle: 'maisenpai', status: 'online' },
  { id: '3', name: 'Sammie Hayes', handle: 'sammiehay', status: 'away' },
  { id: '4', name: 'Kara Stone', handle: 'kara.st', status: 'online' },
  { id: '5', name: 'Ibrahim Diallo', handle: 'ibrahimd', status: 'offline' },
  { id: '6', name: 'Elise Reed', handle: 'elise.reed', status: 'online' },
  { id: '7', name: 'James Park', handle: 'jamesp', status: 'away' },
  { id: '8', name: 'Leah Gomez', handle: 'leahg', status: 'online' },
];

export function FollowingStories() {
  const { user } = useSession();
  const stories = DEMO_STORIES;

  if (!user || stories.length === 0) {
    return null;
  }

  return (
    <Box
      sx={{
        width: '100%',
        backgroundColor: 'transparent',
        borderRadius: 0,
        boxShadow: 'none',
        px: 2,
        py: 1.5,
        overflow: 'hidden',
        display: 'flex',
        alignItems: 'center',
      }}
    >
      <Box
        sx={{
          display: 'flex',
          gap: 2,
          overflowX: 'auto',
          pb: 1,
          pr: 1,
        }}
      >
        {stories.map((story) => (
          <Box
            key={story.id}
            sx={{
              width: 72,
              minWidth: 72,
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              gap: 1,
            }}
          >
            <Box
              sx={{
                position: 'relative',
                width: 72,
                height: 72,
                borderRadius: '50%',
                backgroundImage: STORY_RING_GRADIENT,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <Box
                sx={{
                  width: 64,
                  height: 64,
                  borderRadius: '50%',
                  backgroundColor: 'common.white',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  boxShadow: '0px 1px 2px rgba(16, 24, 40, 0.08)',
                }}
              >
                <Avatar
                  src={story.avatar}
                  alt={story.name}
                  sx={{
                    width: 56,
                    height: 56,
                    border: '2px solid #fff',
                    fontWeight: 600,
                    fontSize: 16,
                  }}
                >
                  {story.name.charAt(0).toUpperCase()}
                </Avatar>
              </Box>
              <Box
                sx={{
                  position: 'absolute',
                  bottom: 6,
                  right: 6,
                  width: 12,
                  height: 12,
                  borderRadius: '50%',
                  backgroundColor: STATUS_COLORS[story.status],
                  border: '2px solid #fff',
                  boxShadow: '0px 1px 2px rgba(16, 24, 40, 0.12)',
                }}
              />
            </Box>
            <Typography
              variant="caption"
              sx={{
                width: '100%',
                fontSize: 12,
                fontWeight: 600,
                color: 'text.secondary',
                textAlign: 'center',
                lineHeight: '16px',
                letterSpacing: '-0.06px',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
              }}
            >
              {story.handle}
            </Typography>
          </Box>
        ))}
      </Box>
    </Box>
  );
}

