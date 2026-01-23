'use client';

import { Chip } from '@mui/material';
import type { ModerationDetails } from '@telar/sdk';

interface ForensicBadgeProps {
  details: ModerationDetails | null;
}

/**
 * ForensicBadge
 * 
 * Visualizes the model_used field to show which moderation layer flagged the content.
 * - L2: Yellow (Fast/Regex/Keyword)
 * - L3: Blue (ONNX)
 * - L4: Purple (LLM)
 * - analysis_failed: Red
 */
export function ForensicBadge({ details }: ForensicBadgeProps) {
  if (!details) {
    return (
      <Chip
        label="Unknown"
        color="default"
        size="small"
        sx={{ fontWeight: 'medium' }}
      />
    );
  }

  const modelUsed = details.model_used || '';
  const status = details.analysis_status || '';

  // Check for analysis_failed status first
  if (status === 'analysis_failed' || modelUsed.includes('failed')) {
    return (
      <Chip
        label="Analysis Failed"
        color="error"
        size="small"
        sx={{ fontWeight: 'medium' }}
      />
    );
  }

  // L2: Keyword/Regex/Heuristic filters (Fast)
  if (
    modelUsed.includes('keyword') ||
    modelUsed.includes('Keyword') ||
    modelUsed.includes('regex') ||
    modelUsed.includes('heuristic') ||
    modelUsed.includes('L2')
  ) {
    return (
      <Chip
        label="L2: Fast Filter"
        color="warning"
        size="small"
        sx={{ fontWeight: 'medium' }}
      />
    );
  }

  // L3: ONNX models
  if (
    modelUsed.includes('ONNX') ||
    modelUsed.includes('onnx') ||
    modelUsed.includes('toxic-bert') ||
    modelUsed.includes('L3')
  ) {
    return (
      <Chip
        label="L3: ONNX Model"
        color="primary"
        size="small"
        sx={{ fontWeight: 'medium' }}
      />
    );
  }

  // L4: LLM models (Qwen, Llama, GPT, etc.)
  if (
    modelUsed.includes('qwen') ||
    modelUsed.includes('llama') ||
    modelUsed.includes('gpt') ||
    modelUsed.includes('LLM') ||
    modelUsed.includes('L4')
  ) {
    return (
      <Chip
        label="L4: LLM Analysis"
        sx={{
          backgroundColor: '#9c27b0', // Purple
          color: 'white',
          fontWeight: 'medium',
          '&:hover': {
            backgroundColor: '#7b1fa2',
          },
        }}
        size="small"
      />
    );
  }

  // Cache hits
  if (modelUsed.includes('cache') || modelUsed.includes('Cache')) {
    return (
      <Chip
        label="Cached Result"
        color="info"
        size="small"
        sx={{ fontWeight: 'medium' }}
      />
    );
  }

  // Fallback: show the model_used value
  return (
    <Chip
      label={modelUsed || 'Unknown Model'}
      color="default"
      size="small"
      sx={{ fontWeight: 'medium' }}
    />
  );
}
