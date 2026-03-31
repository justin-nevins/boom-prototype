import { useRef } from 'react';
import { BACKGROUND_OPTIONS, BackgroundOption } from '../lib/backgrounds';

const MAX_WIDTH = 1920;
const MAX_HEIGHT = 1080;

function resizeImage(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      const img = new Image();
      img.onload = () => {
        let { width, height } = img;
        if (width > MAX_WIDTH || height > MAX_HEIGHT) {
          const scale = Math.min(MAX_WIDTH / width, MAX_HEIGHT / height);
          width = Math.round(width * scale);
          height = Math.round(height * scale);
        }
        const canvas = document.createElement('canvas');
        canvas.width = width;
        canvas.height = height;
        const ctx = canvas.getContext('2d');
        if (!ctx) { reject(new Error('Canvas not available')); return; }
        ctx.drawImage(img, 0, 0, width, height);
        resolve(canvas.toDataURL('image/jpeg', 0.85));
      };
      img.onerror = () => reject(new Error('Failed to load image'));
      img.src = reader.result as string;
    };
    reader.onerror = () => reject(new Error('Failed to read file'));
    reader.readAsDataURL(file);
  });
}

interface BackgroundOptionGridProps {
  selectedId: string;
  onSelect: (option: BackgroundOption) => void;
  disabled?: boolean;
  customBackgrounds?: BackgroundOption[];
  onUpload?: (option: BackgroundOption) => void;
}

export default function BackgroundOptionGrid({ selectedId, onSelect, disabled, customBackgrounds, onUpload }: BackgroundOptionGridProps) {
  const fileInputRef = useRef<HTMLInputElement>(null!);

  const allOptions = [...BACKGROUND_OPTIONS, ...(customBackgrounds || [])];

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file || !onUpload) return;
    try {
      const dataUrl = await resizeImage(file);
      const option: BackgroundOption = {
        id: `custom-${Date.now()}`,
        type: 'image',
        label: file.name.replace(/\.[^.]+$/, ''),
        value: dataUrl,
      };
      onUpload(option);
    } catch (err) {
      console.error('Failed to process uploaded image:', err);
    }
    e.target.value = '';
  };

  return (
    <div className="grid grid-cols-4 gap-2">
      {allOptions.map((option) => (
        <button
          key={option.id}
          onClick={() => onSelect(option)}
          disabled={disabled}
          className={`
            relative aspect-square rounded-lg transition-all
            ${selectedId === option.id
              ? 'ring-2 ring-[#2B88D9] ring-offset-2 ring-offset-slate-800'
              : 'hover:ring-1 hover:ring-slate-500'
            }
            ${disabled ? 'opacity-50 cursor-wait' : 'cursor-pointer'}
          `}
          title={option.label}
        >
          {option.type === 'none' && (
            <div className="w-full h-full bg-slate-700 rounded-lg flex items-center justify-center">
              <svg className="w-5 h-5 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />
              </svg>
            </div>
          )}
          {option.type === 'blur' && (
            <div className="w-full h-full bg-slate-600 rounded-lg flex items-center justify-center relative overflow-hidden">
              <div
                className="absolute inset-0 bg-gradient-to-br from-slate-500 to-slate-700"
                style={{ filter: `blur(${(option.blurAmount || 10) / 5}px)` }}
              />
              <span className="relative text-xs text-white font-medium">
                {option.blurAmount}
              </span>
            </div>
          )}
          {option.type === 'image' && (
            <div
              className="w-full h-full rounded-lg bg-cover bg-center"
              style={{ backgroundImage: `url(${option.value})` }}
            />
          )}
        </button>
      ))}

      {/* Upload button */}
      {onUpload && (
        <button
          onClick={() => fileInputRef.current.click()}
          disabled={disabled}
          className="relative aspect-square rounded-lg transition-all hover:ring-1 hover:ring-slate-500 cursor-pointer"
          title="Upload custom background"
        >
          <div className="w-full h-full bg-slate-700 rounded-lg flex flex-col items-center justify-center border-2 border-dashed border-slate-500">
            <svg className="w-5 h-5 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
          </div>
        </button>
      )}

      <input
        ref={fileInputRef}
        type="file"
        accept="image/*"
        onChange={handleFileChange}
        className="hidden"
      />
    </div>
  );
}
