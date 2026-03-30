import { BACKGROUND_OPTIONS, BackgroundOption } from '../lib/backgrounds';

interface BackgroundOptionGridProps {
  selectedId: string;
  onSelect: (option: BackgroundOption) => void;
  disabled?: boolean;
}

export default function BackgroundOptionGrid({ selectedId, onSelect, disabled }: BackgroundOptionGridProps) {
  return (
    <div className="grid grid-cols-4 gap-2">
      {BACKGROUND_OPTIONS.map((option) => (
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
          {option.type === 'color' && (
            <div
              className="w-full h-full rounded-lg"
              style={{ backgroundColor: option.value }}
            />
          )}
          {option.type === 'image' && (
            <div
              className="w-full h-full rounded-lg bg-cover bg-center"
              style={{ backgroundImage: `url(${option.value})` }}
            />
          )}
        </button>
      ))}
    </div>
  );
}
