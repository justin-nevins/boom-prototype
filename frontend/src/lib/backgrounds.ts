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

  // Solid colors
  { id: 'color-navy', type: 'color', label: 'Navy', value: '#1e3a5f' },
  { id: 'color-slate', type: 'color', label: 'Slate', value: '#334155' },
  { id: 'color-green', type: 'color', label: 'Green', value: '#166534' },

  // Images
  { id: 'image-mountains', type: 'image', label: 'Snowy Mountains', value: 'https://images.unsplash.com/photo-1519681393784-d120267933ba?w=1920&q=80' },
];

export const STORAGE_KEY = 'boom-virtual-background';
