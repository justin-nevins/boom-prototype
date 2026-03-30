export type BackgroundType = 'none' | 'blur' | 'color' | 'image';

export interface BackgroundOption {
  id: string;
  type: BackgroundType;
  label: string;
  value?: string;
  blurAmount?: number;
}

export const BACKGROUND_OPTIONS: BackgroundOption[] = [
  // None - no background processing
  { id: 'none', type: 'none', label: 'None' },

  // Blur levels
  { id: 'blur-light', type: 'blur', label: 'Light blur', blurAmount: 5 },
  { id: 'blur-medium', type: 'blur', label: 'Medium blur', blurAmount: 10 },
  { id: 'blur-heavy', type: 'blur', label: 'Heavy blur', blurAmount: 20 },

  // Images
  { id: 'image-mountains', type: 'image', label: 'Snowy Mountains', value: 'https://images.unsplash.com/photo-1519681393784-d120267933ba?w=1920&q=80' },
];

export const STORAGE_KEY = 'boom-virtual-background';

export function loadSavedBackground(): BackgroundOption {
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
}
