import { useLocalParticipant } from '@livekit/components-react';
import { LocalVideoTrack } from 'livekit-client';
import { BACKGROUND_OPTIONS, BackgroundOption } from '../lib/backgrounds';
import { useVirtualBackground } from '../hooks/useVirtualBackground';

interface BackgroundSelectorProps {
  onClose: () => void;
}

export default function BackgroundSelector({ onClose }: BackgroundSelectorProps) {
  const { localParticipant } = useLocalParticipant();

  const videoTrack = localParticipant
    .getTrackPublications()
    .find((pub) => pub.kind === 'video' && pub.source === 'camera')
    ?.track as LocalVideoTrack | undefined;

  const { currentBackground, isApplying, error, applyBackground } = useVirtualBackground({
    videoTrack,
  });

  const handleSelect = async (option: BackgroundOption) => {
    await applyBackground(option);
  };

  return (
    <div className="absolute top-12 right-0 w-72 bg-slate-800 rounded-lg shadow-xl border border-slate-700 z-50">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-slate-700">
        <h3 className="text-white font-medium text-sm">Virtual Background</h3>
        <button
          onClick={onClose}
          className="text-slate-400 hover:text-white transition-colors"
          aria-label="Close"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      {/* Error message */}
      {error && (
        <div className="px-4 py-2 bg-red-900/50 text-red-300 text-xs">
          {error}
        </div>
      )}

      {/* Options grid */}
      <div className="p-3">
        <div className="grid grid-cols-4 gap-2">
          {BACKGROUND_OPTIONS.map((option) => (
            <button
              key={option.id}
              onClick={() => handleSelect(option)}
              disabled={isApplying}
              className={`
                relative aspect-square rounded-lg transition-all
                ${currentBackground.id === option.id
                  ? 'ring-2 ring-blue-500 ring-offset-2 ring-offset-slate-800'
                  : 'hover:ring-1 hover:ring-slate-500'
                }
                ${isApplying ? 'opacity-50 cursor-wait' : 'cursor-pointer'}
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

        {/* Labels */}
        <div className="mt-3 text-center">
          <span className="text-slate-400 text-xs">
            {isApplying ? 'Applying...' : currentBackground.label}
          </span>
        </div>
      </div>

      {/* Footer hint */}
      <div className="px-4 py-2 border-t border-slate-700">
        <p className="text-slate-500 text-xs">
          Virtual backgrounds require good lighting for best results.
        </p>
      </div>
    </div>
  );
}
