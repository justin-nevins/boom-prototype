import { useState, useEffect, useCallback } from 'react';
import { BackgroundBlur, VirtualBackground, ProcessorWrapper } from '@livekit/track-processors';
import type { LocalVideoTrack } from 'livekit-client';
import { BackgroundOption, BACKGROUND_OPTIONS, STORAGE_KEY } from '../lib/backgrounds';

// Preload image with CORS support for canvas processing
const preloadImage = (url: string): Promise<HTMLImageElement> => {
  return new Promise((resolve, reject) => {
    const img = new Image();
    img.crossOrigin = 'anonymous';
    img.onload = () => resolve(img);
    img.onerror = () => reject(new Error(`Failed to load image: ${url}`));
    img.src = url;
  });
};

// Generate a data URL for a solid color (VirtualBackground only accepts image paths)
const createColorDataUrl = (hexColor: string): string => {
  const canvas = document.createElement('canvas');
  canvas.width = 1;
  canvas.height = 1;
  const ctx = canvas.getContext('2d');
  if (ctx) {
    ctx.fillStyle = hexColor;
    ctx.fillRect(0, 0, 1, 1);
  }
  return canvas.toDataURL('image/png');
};

interface UseVirtualBackgroundOptions {
  videoTrack: LocalVideoTrack | undefined;
}

interface UseVirtualBackgroundReturn {
  currentBackground: BackgroundOption;
  isApplying: boolean;
  error: string | null;
  applyBackground: (option: BackgroundOption) => Promise<void>;
  clearBackground: () => Promise<void>;
}

export function useVirtualBackground({
  videoTrack,
}: UseVirtualBackgroundOptions): UseVirtualBackgroundReturn {
  const [currentBackground, setCurrentBackground] = useState<BackgroundOption>(() => {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved) {
      try {
        const parsed = JSON.parse(saved);
        return BACKGROUND_OPTIONS.find((opt) => opt.id === parsed.id) || BACKGROUND_OPTIONS[0];
      } catch {
        return BACKGROUND_OPTIONS[0];
      }
    }
    return BACKGROUND_OPTIONS[0];
  });
  const [isApplying, setIsApplying] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [currentProcessor, setCurrentProcessor] = useState<ProcessorWrapper<any> | null>(null);

  const clearBackground = useCallback(async () => {
    if (!videoTrack) return;

    try {
      await videoTrack.stopProcessor();
      setCurrentProcessor(null);
    } catch (err) {
      console.error('Failed to clear background processor:', err);
    }
  }, [videoTrack]);

  const applyBackground = useCallback(
    async (option: BackgroundOption) => {
      if (!videoTrack) {
        setError('No video track available');
        return;
      }

      setIsApplying(true);
      setError(null);

      try {
        // Clear existing processor
        if (currentProcessor) {
          await videoTrack.stopProcessor();
          setCurrentProcessor(null);
        }

        if (option.type === 'none') {
          setCurrentBackground(option);
          localStorage.setItem(STORAGE_KEY, JSON.stringify({ id: option.id }));
          setIsApplying(false);
          return;
        }

        let processor: ProcessorWrapper<any>;

        if (option.type === 'blur') {
          processor = BackgroundBlur(option.blurAmount || 10);
        } else if (option.type === 'color') {
          // VirtualBackground needs an image path, so create a 1x1 color data URL
          const colorDataUrl = createColorDataUrl(option.value || '#334155');
          processor = VirtualBackground(colorDataUrl);
        } else if (option.type === 'image') {
          // VirtualBackground with image URL - preload with CORS first
          if (!option.value) {
            throw new Error('Image URL is required for image backgrounds');
          }
          // Preload the image to verify it's accessible and CORS-enabled
          try {
            await preloadImage(option.value);
          } catch (preloadErr) {
            throw new Error('Image could not be loaded. It may be blocked by CORS policy.');
          }
          processor = VirtualBackground(option.value);
        } else {
          throw new Error('Unknown background type: ' + option.type);
        }

        // Set processor with timeout to catch hanging operations
        const timeoutPromise = new Promise((_, reject) =>
          setTimeout(() => reject(new Error('Background processor timed out')), 10000)
        );

        await Promise.race([
          videoTrack.setProcessor(processor),
          timeoutPromise
        ]);
        setCurrentProcessor(processor);
        setCurrentBackground(option);
        localStorage.setItem(STORAGE_KEY, JSON.stringify({ id: option.id }));
      } catch (err) {
        console.error('Failed to apply background:', err);
        setError(err instanceof Error ? err.message : 'Failed to apply background');
      } finally {
        setIsApplying(false);
      }
    },
    [videoTrack, currentProcessor]
  );

  // Apply saved background when video track becomes available
  // Only auto-apply blur backgrounds - colors/images can cause dark video issues
  useEffect(() => {
    if (videoTrack && currentBackground.type === 'blur' && !currentProcessor) {
      applyBackground(currentBackground).catch((err) => {
        console.error('Failed to auto-apply background, clearing:', err);
        // Clear saved preference if auto-apply fails
        localStorage.removeItem(STORAGE_KEY);
        setCurrentBackground(BACKGROUND_OPTIONS[0]); // Reset to 'none'
      });
    }
  }, [videoTrack]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (videoTrack && currentProcessor) {
        videoTrack.stopProcessor().catch(console.error);
      }
    };
  }, [videoTrack, currentProcessor]);

  return {
    currentBackground,
    isApplying,
    error,
    applyBackground,
    clearBackground,
  };
}
