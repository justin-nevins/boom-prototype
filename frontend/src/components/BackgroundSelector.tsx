import { useLocalParticipant } from '@livekit/components-react';
import { LocalVideoTrack } from 'livekit-client';
import { BackgroundOption } from '../lib/backgrounds';
import { useVirtualBackground } from '../hooks/useVirtualBackground';
import BackgroundOptionGrid from './BackgroundOptionGrid';

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
        <BackgroundOptionGrid
          selectedId={currentBackground.id}
          onSelect={handleSelect}
          disabled={isApplying}
        />

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
