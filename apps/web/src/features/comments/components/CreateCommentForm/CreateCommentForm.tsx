'use client';

import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Avatar, Box, IconButton, TextField } from '@mui/material';
import SendIcon from '@mui/icons-material/Send';
import { createCommentSchema } from '../../constants/validation';
import { useCreateCommentMutation } from '../../client';
import { useSession } from '@/features/auth/client';

interface CreateCommentFormProps {
  postId: string;
  parentCommentId?: string;
  onSuccess?: () => void;
  autoFocus?: boolean;
}

type CreateCommentFormValues = {
  text: string;
};

export function CreateCommentForm({ postId, parentCommentId, onSuccess, autoFocus }: CreateCommentFormProps) {
  const [submitting, setSubmitting] = useState(false);
  const createMutation = useCreateCommentMutation(postId);
  const { user } = useSession();

  const { register, handleSubmit, reset, formState, watch } = useForm<CreateCommentFormValues>({
    resolver: zodResolver(createCommentSchema.pick({ text: true })),
    defaultValues: { text: '' },
  });

  const commentText = watch('text');
  const hasText = commentText?.trim().length > 0;

  const onSubmit = handleSubmit(async (values) => {
    if (!hasText) return;
    setSubmitting(true);
    try {
      await createMutation.mutateAsync({ text: values.text, parentCommentId });
      reset();
      onSuccess?.();
    } finally {
      setSubmitting(false);
    }
  });

  return (
    <Box
      component="form"
      onSubmit={onSubmit}
      sx={{
        display: 'flex',
        gap: '16px',
        alignItems: 'flex-start',
        mb: 0,
        pt: parentCommentId ? 0 : '20px',
      }}
    >
      <Avatar
        src={user?.avatar}
        alt={user?.displayName || user?.socialName}
        sx={{ width: 40, height: 40, flexShrink: 0 }}
      >
        {(user?.displayName || user?.socialName)?.[0]?.toUpperCase()}
      </Avatar>
      <Box sx={{ flexGrow: 1, position: 'relative' }}>
        <TextField
          {...register('text')}
          placeholder="Write your comments here..."
          multiline
          rows={3}
          autoFocus={autoFocus}
          error={!!formState.errors.text}
          helperText={formState.errors.text?.message}
          fullWidth
          sx={{
            '& .MuiOutlinedInput-root': {
              borderRadius: '16px',
              backgroundColor: '#F8FAFC',
              borderColor: '#E2E8F0',
              fontFamily: 'PlusJakartaSans, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
              fontSize: '14px',
              fontWeight: 400,
              lineHeight: '22.4px',
              color: '#1E293B',
              '&:hover': {
                borderColor: '#E2E8F0',
              },
              '&.Mui-focused': {
                borderColor: '#4F46E5',
                borderWidth: '1px',
                '& fieldset': {
                  borderColor: '#4F46E5',
                  borderWidth: '1px',
                },
              },
              '& fieldset': {
                borderColor: '#E2E8F0',
                borderWidth: '1px',
              },
              '& input::placeholder': {
                color: '#1E293B',
                opacity: 1,
                fontFamily: 'PlusJakartaSans, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
                fontSize: '14px',
                fontWeight: 400,
                lineHeight: '22.4px',
              },
            },
          }}
        />
        </Box>
      <IconButton
        type="submit"
        disabled={submitting || !hasText}
        sx={{
          bgcolor: '#4F46E5',
          color: 'white',
          width: 40,
          height: 40,
          borderRadius: '50%',
          flexShrink: 0,
          mt: '14px',
          '&:hover': {
            bgcolor: '#4338CA',
          },
          '&:disabled': {
            bgcolor: '#CBD5E1',
            color: '#94A3B8',
          },
        }}
      >
        <SendIcon sx={{ fontSize: '20px' }} />
      </IconButton>
    </Box>
  );
}


