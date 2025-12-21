import { useState, useRef, useCallback, useEffect } from 'react';
import { useDropzone } from 'react-dropzone';
import { sdk } from '@/lib/sdk';
import { uploadFileWithCompression } from '@telar/sdk';


/**
 * useImageUpload Hook
 * 
 * Shared hook for image upload functionality with drag & drop support.
 * Encapsulates upload state, progress tracking, error handling, and dropzone logic.
 * 
 * @example
 * ```tsx
 * const {
 *   imageUrl,
 *   uploadProgress,
 *   uploadError,
 *   isUploading,
 *   isDragActive,
 *   getRootProps,
 *   getInputProps,
 *   handleImageClick,
 *   handleImageChange,
 *   handleRemoveImage,
 *   resetUpload,
 * } = useImageUpload({
 *   disabled: isSubmitting,
 * });
 * ```
 */
export interface UseImageUploadOptions {
  /**
   * Whether the upload is disabled (e.g., during form submission)
   */
  disabled?: boolean;
  /**
   * Callback when upload completes successfully
   */
  onUploadSuccess?: (url: string) => void;
  /**
   * Callback when upload fails
   */
  onUploadError?: (error: Error) => void;
}

export interface UseImageUploadReturn {
  /**
   * The uploaded image URL (null if no image)
   * This is the final CDN URL after upload completes
   */
  imageUrl: string | null;
  /**
   * Optimistic preview URL (blob URL) - shown immediately on file selection
   * Use this for instant preview before upload completes
   */
  previewUrl: string | null;
  /**
   * Upload progress (0-100, null if not uploading)
   */
  uploadProgress: number | null;
  /**
   * Upload error message (null if no error)
   */
  uploadError: string | null;
  /**
   * Whether compression is currently in progress
   */
  isCompressing: boolean;
  /**
   * Whether an upload is currently in progress (includes compression phase)
   */
  isUploading: boolean;
  /**
   * Whether a file is being dragged over the dropzone
   */
  isDragActive: boolean;
  /**
   * Dropzone root props (spread on container element)
   */
  getRootProps: () => Record<string, unknown>;
  /**
   * Dropzone input props (spread on file input)
   */
  getInputProps: () => Record<string, unknown>;
  /**
   * File input ref (attach to input element)
   */
  fileInputRef: React.RefObject<HTMLInputElement | null>;
  /**
   * Handler for clicking the upload button
   */
  handleImageClick: () => void;
  /**
   * Handler for file input change event
   */
  handleImageChange: (event: React.ChangeEvent<HTMLInputElement>) => Promise<void>;
  /**
   * Handler for removing the uploaded image
   */
  handleRemoveImage: () => void;
  /**
   * Reset all upload state (useful when closing dialogs)
   */
  resetUpload: () => void;
}

export function useImageUpload(
  options: UseImageUploadOptions = {}
): UseImageUploadReturn {
  const { disabled = false, onUploadSuccess, onUploadError } = options;

  const [imageUrl, setImageUrl] = useState<string | null>(null);
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const [uploadProgress, setUploadProgress] = useState<number | null>(null);
  const [uploadError, setUploadError] = useState<string | null>(null);
  const [isCompressing, setIsCompressing] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const isUploading = uploadProgress !== null || isCompressing;

  const handleFileUpload = useCallback(
    async (file: File) => {
      setUploadError(null);
      setUploadProgress(0);
      setIsCompressing(false);

      // 1. Generate optimistic preview (blob URL) - immediate feedback
      const blobUrl = URL.createObjectURL(file);
      setPreviewUrl(blobUrl);

      try {
        const { url } = await uploadFileWithCompression(
          file,
          sdk.storage,
          (progress) => setUploadProgress(progress),
          (compressing) => setIsCompressing(compressing)
        );

        // Clean up blob URL
        if (previewUrl) {
          URL.revokeObjectURL(previewUrl);
        }

        setImageUrl(url);
        setPreviewUrl(null);
        setUploadProgress(null);
        setIsCompressing(false);
        onUploadSuccess?.(url);
      } catch (error) {
        console.error('Failed to upload image:', error);
        
        // Clean up blob URL on error
        if (blobUrl) {
          URL.revokeObjectURL(blobUrl);
        }
        
        setPreviewUrl(null);
        const errorMessage =
          error instanceof Error ? error.message : 'Failed to upload image';
        setUploadError(errorMessage);
        setUploadProgress(null);
        setIsCompressing(false);
        onUploadError?.(error instanceof Error ? error : new Error(errorMessage));
      } finally {
        // Reset file input
        if (fileInputRef.current) {
          fileInputRef.current.value = '';
        }
      }
    },
    [onUploadSuccess, onUploadError, previewUrl]
  );

  const handleImageClick = useCallback(() => {
    fileInputRef.current?.click();
  }, []);

  const handleImageChange = useCallback(
    async (event: React.ChangeEvent<HTMLInputElement>) => {
      const file = event.target.files?.[0];
      if (!file) return;
      await handleFileUpload(file);
    },
    [handleFileUpload]
  );

  const handleRemoveImage = useCallback(() => {
    // Clean up blob URL if exists
    if (previewUrl) {
      URL.revokeObjectURL(previewUrl);
    }
    setImageUrl(null);
    setPreviewUrl(null);
    setUploadError(null);
  }, [previewUrl]);

  const resetUpload = useCallback(() => {
    // Clean up blob URL if exists
    if (previewUrl) {
      URL.revokeObjectURL(previewUrl);
    }
    setImageUrl(null);
    setPreviewUrl(null);
    setUploadProgress(null);
    setUploadError(null);
    setIsCompressing(false);
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  }, [previewUrl]);

  // Drag & drop handler
  const onDrop = useCallback(
    (acceptedFiles: File[]) => {
      const file = acceptedFiles[0];
      if (file) {
        handleFileUpload(file);
      }
    },
    [handleFileUpload]
  );

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop,
    accept: {
      'image/jpeg': ['.jpg', '.jpeg'],
      'image/png': ['.png'],
      'image/webp': ['.webp'],
    },
    multiple: false,
    noClick: true, // Don't trigger file dialog on click
    disabled: disabled || isUploading,
  });

  // Cleanup blob URL on unmount
  useEffect(() => {
    return () => {
      if (previewUrl) {
        URL.revokeObjectURL(previewUrl);
      }
    };
  }, [previewUrl]);

  return {
    imageUrl,
    previewUrl,
    uploadProgress,
    uploadError,
    isCompressing,
    isUploading,
    isDragActive,
    getRootProps,
    getInputProps,
    fileInputRef,
    handleImageClick,
    handleImageChange,
    handleRemoveImage,
    resetUpload,
  };
}

