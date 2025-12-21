/**
 * Storage Utilities
 * 
 * Client-side compression utilities for file uploads.
 * Compression must happen BEFORE calling initializeUpload to reduce
 * file sizes and save server CPU and R2 storage costs.
 */

import imageCompression from 'browser-image-compression';

// Type definition for compression options
type CompressionOptions = {
  maxSizeMB?: number;
  maxWidthOrHeight?: number;
  useWebWorker?: boolean;
  fileType?: string;
  initialQuality?: number;
};

/**
 * Compression options for images
 * Optimized for web display while maintaining reasonable quality
 */
const DEFAULT_COMPRESSION_OPTIONS: CompressionOptions = {
  maxSizeMB: 2, // Maximum file size after compression (matches backend limit)
  maxWidthOrHeight: 1920, // Maximum dimension (good for web display)
  useWebWorker: true, // Use web worker for better performance
  fileType: 'image/jpeg', // Convert to JPEG for better compression
  initialQuality: 0.85, // 85% quality (good balance)
};

/**
 * Compress an image file before upload
 * 
 * Smart compression that:
 * - Skips non-image files
 * - Skips small files (< 1MB) to avoid unnecessary processing
 * - Uses Web Workers to prevent UI freeze
 * - Targets 1.9MB to stay safely below 2MB backend limit
 * 
 * @param file - The image file to compress
 * @param options - Optional compression options (defaults to DEFAULT_COMPRESSION_OPTIONS)
 * @param onProgress - Optional progress callback (0-100)
 * @returns Compressed file blob (or original if skipped)
 * 
 * @example
 * ```typescript
 * const originalFile = event.target.files[0];
 * const compressedFile = await compressImage(originalFile, undefined, (p) => {
 *   console.log(`Compressing: ${p}%`);
 * });
 * 
 * // Now upload the compressed file
 * const uploadResponse = await sdk.storage.initializeUpload({
 *   name: compressedFile.name,
 *   contentType: compressedFile.type,
 *   size: compressedFile.size,
 * });
 * ```
 */
export async function compressImage(
  file: File,
  options?: Partial<CompressionOptions>,
  onProgress?: (progress: number) => void
): Promise<File> {
  // 1. Skip non-images
  if (!file.type.startsWith('image/')) {
    return file; // Return original for non-images
  }

  // 2. Skip small files (Don't compress a 50KB icon)
  const fileSizeMB = file.size / 1024 / 1024;
  if (fileSizeMB < 1) {
    return file; // Return original for small files
  }

  // 3. Merge user options with defaults (target 1.9MB to stay below 2MB limit)
  const compressionOptions: CompressionOptions = {
    maxSizeMB: 1.9, // Target slightly below the 2MB backend limit
    maxWidthOrHeight: 1920, // 1080p is usually enough for web
    useWebWorker: true, // Prevent UI freeze
    ...DEFAULT_COMPRESSION_OPTIONS,
    ...options,
  };

  try {
    onProgress?.(10);
    
    const compressedFile = await imageCompression(file, {
      ...compressionOptions,
      onProgress: (p) => {
        // Map compression progress (0-100) to overall progress range
        // Compression phase is 0-50% of total upload process
        onProgress?.(10 + (p * 0.4)); // 10-50%
      },
    });
    
    // Log compression ratio for debugging
    const compressedSize = compressedFile.size;
    const compressedSizeMB = compressedSize / 1024 / 1024;
    const ratio = ((1 - compressedSize / file.size) * 100).toFixed(1);
    
    onProgress?.(50);
    
    return compressedFile;
  } catch (error) {
    throw new Error('Unable to optimize image. Please try a smaller file.');
  }
}

/**
 * Validate file before upload
 * 
 * @param file - The file to validate
 * @param maxSizeMB - Maximum file size in MB (default: 2)
 * @param allowedTypes - Allowed MIME types (default: image/jpeg, image/png, image/webp)
 * @throws Error if file is invalid
 */
export function validateFile(
  file: File,
  maxSizeMB: number = 2,
  allowedTypes: string[] = ['image/jpeg', 'image/png', 'image/webp']
): void {
  // Check file size
  const maxSizeBytes = maxSizeMB * 1024 * 1024;
  if (file.size > maxSizeBytes) {
    throw new Error(`File too large: ${(file.size / 1024 / 1024).toFixed(2)}MB. Maximum size: ${maxSizeMB}MB. Please compress the file.`);
  }

  // Check MIME type
  if (!allowedTypes.includes(file.type)) {
    throw new Error(`File type not allowed: ${file.type}. Allowed types: ${allowedTypes.join(', ')}`);
  }
}

/**
 * Upload file with automatic compression
 * 
 * This is a convenience function that:
 * 1. Validates the file
 * 2. Compresses it (if it's an image) - Progress: 0-50%
 * 3. Initializes the upload - Progress: 50%
 * 4. Uploads to the presigned URL - Progress: 50-100%
 * 5. Confirms the upload
 * 6. Gets the public URL
 * 
 * @param file - The file to upload
 * @param storageApi - The storage API client
 * @param onProgress - Optional progress callback (0-100)
 * @param onCompressionState - Optional callback to indicate compression phase (true = compressing, false = uploading)
 * @returns The file ID, key, and public URL
 */
export async function uploadFileWithCompression(
  file: File,
  storageApi: {
    initializeUpload: (req: { name: string; contentType: string; size: number }) => Promise<{ uploadUrl: string; fileId: string; key: string }>;
    confirmUpload: (req: { fileId: string }) => Promise<{ message: string }>;
    getFileURL: (fileId: string) => Promise<{ url: string }>;
  },
  onProgress?: (progress: number) => void,
  onCompressionState?: (isCompressing: boolean) => void
): Promise<{ fileId: string; key: string; url: string }> {
  // 1. Validate MIME type only (don't validate size - compression will handle it)
  const allowedTypes = ['image/jpeg', 'image/png', 'image/webp'];
  if (!file.type.startsWith('image/') || !allowedTypes.includes(file.type)) {
    throw new Error(`File type not allowed: ${file.type}. Allowed types: ${allowedTypes.join(', ')}`);
  }

  // 2. Client-Side Compression (Progress: 0-50%)
  onCompressionState?.(true);
  let fileToUpload = file;
  
  if (file.type.startsWith('image/')) {
    try {
      fileToUpload = await compressImage(file, undefined, (p) => {
        // Compression progress is 0-50% of total
        onProgress?.(p);
      });
    } catch (e) {
      onCompressionState?.(false);
      // If compression fails, throw error (don't try raw file - backend will reject)
      throw e;
    }
  } else {
    // For non-images, still report progress during "compression" phase
    onProgress?.(50);
  }
  
  onCompressionState?.(false);

  // 3. Initialize Upload (Get Presigned URL) - Progress: 50%
  onProgress?.(50);
  
  let uploadResponse: { uploadUrl: string; fileId: string; key: string };
  try {
    uploadResponse = await storageApi.initializeUpload({
      name: fileToUpload.name,
      contentType: fileToUpload.type,
      size: fileToUpload.size,
    });
  } catch (error) {
    // Handle quota errors with friendly messages
    if (error && typeof error === 'object') {
      const apiError = error as { statusCode?: number; message?: string; code?: string };
      const errorMessage = apiError.message || '';
      
      // Check for daily limit reached
      if (
        apiError.statusCode === 403 ||
        errorMessage.includes('daily upload limit reached') ||
        errorMessage.includes('Daily limit reached')
      ) {
        throw new Error('Daily upload limit reached. Please try again tomorrow.');
      }
      
      // Check for global limit reached
      if (
        apiError.statusCode === 503 ||
        errorMessage.includes('system storage busy') ||
        errorMessage.includes('System storage busy')
      ) {
        throw new Error('System storage busy, try again later.');
      }
      
      // Use the API error message if available
      if (errorMessage) {
        throw new Error(errorMessage);
      }
    }
    throw error;
  }

  const { uploadUrl, fileId, key } = uploadResponse;

  // 4. Upload Directly to R2 (PUT) - Progress: 50-100%
  await new Promise<void>((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open('PUT', uploadUrl);
    xhr.setRequestHeader('Content-Type', fileToUpload.type);
    
    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable) {
        // Map 0-100% of upload to 50-100% of total progress
        const percentComplete = 50 + ((e.loaded / e.total) * 50);
        onProgress?.(percentComplete);
      }
    };
    
    xhr.onload = () => {
      if (xhr.status === 200 || xhr.status === 204) {
        resolve();
      } else {
        reject(new Error(`Upload failed with status ${xhr.status}`));
      }
    };
    
    xhr.onerror = () => reject(new Error('Network error during upload'));
    xhr.send(fileToUpload);
  });

  // 5. Confirm Upload
  await storageApi.confirmUpload({ fileId });

  // 6. Get public URL
  const { url } = await storageApi.getFileURL(fileId);

  onProgress?.(100);

  return { fileId, key, url };
}

