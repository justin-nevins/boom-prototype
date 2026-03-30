import { useState, useCallback } from 'react';
import { useParams, useNavigate, useLocation } from 'react-router-dom';
import { useMediaPreview } from '../hooks/useMediaPreview';
import { STORAGE_KEY, BackgroundOption, loadSavedBackground } from '../lib/backgrounds';
import BackgroundOptionGrid from '../components/BackgroundOptionGrid';

interface LobbyLocationState {
  suggestedName?: string;
}

export default function Lobby() {
  const { roomName } = useParams<{ roomName: string }>();
  const navigate = useNavigate();
  const location = useLocation();
  const state = location.state as LobbyLocationState | null;

  const [name, setName] = useState(() => {
    return state?.suggestedName || sessionStorage.getItem('participantName') || '';
  });

  const { videoRef, stream, isMicOn, isCamOn, toggleMic, toggleCam, error: mediaError } = useMediaPreview();

  const [selectedBg, setSelectedBg] = useState<BackgroundOption>(loadSavedBackground);

  const handleSelectBg = (option: BackgroundOption) => {
    setSelectedBg(option);
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ id: option.id }));
  };

  const joinRoom = useCallback(() => {
    if (!name.trim()) return;
    // Release camera/mic before LiveKit tries to acquire them
    if (stream) {
      stream.getTracks().forEach((t) => t.stop());
    }
    sessionStorage.setItem('participantName', name.trim());
    sessionStorage.setItem('boom-mic-enabled', String(isMicOn));
    sessionStorage.setItem('boom-cam-enabled', String(isCamOn));
    navigate(`/room/${roomName}`);
  }, [name, stream, isMicOn, isCamOn, navigate, roomName]);

  return (
    <div className="min-h-screen bg-slate-900 flex items-center justify-center px-4">
      <div className="w-full max-w-3xl">
        {/* Logo */}
        <div className="flex items-center justify-center gap-2 mb-8">
          <svg className="w-8 h-8 text-[#2B88D9]" fill="currentColor" viewBox="0 0 24 24">
            <path d="M17 10.5V7c0-.55-.45-1-1-1H4c-.55 0-1 .45-1 1v10c0 .55.45 1 1 1h12c.55 0 1-.45 1-1v-3.5l4 4v-11l-4 4z"/>
          </svg>
          <span className="text-xl font-semibold text-white">Meet</span>
          <span className="text-slate-500 text-sm ml-2">Joining: {roomName}</span>
        </div>

        <div className="bg-slate-800 rounded-xl shadow-lg border border-slate-700 overflow-hidden">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-0">
            {/* Left - Camera Preview */}
            <div className="p-6">
              <div className="relative aspect-video bg-slate-900 rounded-lg overflow-hidden">
                {mediaError ? (
                  <div className="absolute inset-0 flex items-center justify-center">
                    <p className="text-red-400 text-sm text-center px-4">{mediaError}</p>
                  </div>
                ) : !isCamOn ? (
                  <div className="absolute inset-0 flex items-center justify-center">
                    <div className="w-16 h-16 bg-slate-700 rounded-full flex items-center justify-center">
                      <svg className="w-8 h-8 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                      </svg>
                    </div>
                  </div>
                ) : (
                  <video
                    ref={videoRef}
                    autoPlay
                    muted
                    playsInline
                    className="w-full h-full object-cover mirror"
                    style={{ transform: 'scaleX(-1)' }}
                  />
                )}
              </div>

              {/* Mic / Camera toggles */}
              <div className="flex justify-center gap-3 mt-4">
                <button
                  onClick={toggleMic}
                  className={`p-3 rounded-full transition-colors ${
                    isMicOn
                      ? 'bg-slate-700 hover:bg-slate-600 text-white'
                      : 'bg-red-600 hover:bg-red-500 text-white'
                  }`}
                  title={isMicOn ? 'Mute microphone' : 'Unmute microphone'}
                >
                  {isMicOn ? (
                    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11a7 7 0 01-7 7m0 0a7 7 0 01-7-7m7 7v4m0 0H8m4 0h4m-4-8a3 3 0 01-3-3V5a3 3 0 116 0v6a3 3 0 01-3 3z" />
                    </svg>
                  ) : (
                    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5.586 15H4a1 1 0 01-1-1v-4a1 1 0 011-1h1.586l4.707-4.707C10.923 3.663 12 4.109 12 5v14c0 .891-1.077 1.337-1.707.707L5.586 15z" />
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2" />
                    </svg>
                  )}
                </button>
                <button
                  onClick={toggleCam}
                  className={`p-3 rounded-full transition-colors ${
                    isCamOn
                      ? 'bg-slate-700 hover:bg-slate-600 text-white'
                      : 'bg-red-600 hover:bg-red-500 text-white'
                  }`}
                  title={isCamOn ? 'Turn off camera' : 'Turn on camera'}
                >
                  {isCamOn ? (
                    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z" />
                    </svg>
                  ) : (
                    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />
                    </svg>
                  )}
                </button>
              </div>
            </div>

            {/* Right - Name + Background + Join */}
            <div className="p-6 border-t md:border-t-0 md:border-l border-slate-700">
              {/* Name input */}
              <div className="mb-6">
                <label className="block text-sm font-medium text-slate-300 mb-1.5">
                  Your name <span className="text-[#D93D1A]">*</span>
                </label>
                <input
                  type="text"
                  required
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="Enter your name"
                  className="w-full px-4 py-2.5 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-[#2B88D9] focus:border-transparent"
                  onKeyDown={(e) => e.key === 'Enter' && name.trim() && joinRoom()}
                  autoFocus
                />
              </div>

              {/* Background selector */}
              <div className="mb-6">
                <label className="block text-sm font-medium text-slate-300 mb-2">
                  Virtual background
                </label>
                <BackgroundOptionGrid
                  selectedId={selectedBg.id}
                  onSelect={handleSelectBg}
                />
                <p className="text-slate-500 text-xs mt-2">
                  {selectedBg.type === 'none' ? selectedBg.label : `${selectedBg.label} - applied when you join`}
                </p>
              </div>

              {/* Join button */}
              <button
                onClick={joinRoom}
                disabled={!name.trim()}
                className="w-full py-3 px-4 bg-[#0396A6] hover:bg-[#027d8a] text-white font-medium rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed text-lg"
              >
                Join Meeting
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
