import { z } from 'zod';

export const createCommentSchema = z.object({
  text: z
    .string()
    .min(1, 'Comment cannot be empty')
    .max(1000, 'Comment must be 1000 characters or less'),
  postId: z.string().uuid('Invalid post ID'),
  parentCommentId: z.string().uuid('Invalid parent ID').optional(),
});

export const updateCommentSchema = z.object({
  text: z
    .string()
    .min(1, 'Comment cannot be empty')
    .max(1000, 'Comment must be 1000 characters or less'),
  objectId: z.string().uuid('Invalid comment ID'),
});


