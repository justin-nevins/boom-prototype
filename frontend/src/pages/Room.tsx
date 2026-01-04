import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  LiveKitRoom,
  VideoConference,
  RoomAudioRenderer,
} from '@livekit/components-react';
import '@livekit/components-styles';
import Transcription from '../components/Transcription';
import ErrorBoundary from '../components/ErrorBoundary';
import BackgroundToggle from '../components/BackgroundToggle';

const BACKEND_URL = import.meta.env.VITE_BACKEND_URL || 'http://localhost:8080';
const LIVEKIT_URL = import.meta.env.VITE_LIVEKIT_URL;

export default function Room() {
  const { roomName } = useParams<{ roomName: string }>();
  const navigate = useNavigate();
  const [token, setToken] = useState<string>('');
  const [error, setError] = useState<string>('');

  useEffect(() => {
    const participantName = sessionStorage.getItem('participantName') || 'Guest';
    
    // Get token from backend
    fetch(`${BACKEND_URL}/api/token`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        roomName,
        participantName,
      }),
    })
      .then((res) => res.json())
      .then((data) => {
        if (data.token) {
          setToken(data.token);
        } else {
          setError('Failed to get access token');
        }
      })
      .catch((err) => {
        console.error('Token error:', err);
        setError('Failed to connect to server');
      });
  }, [roomName]);

  const handleDisconnect = () => {
    navigate('/');
  };

  if (error) {
    return (
      <div className="min-h-screen bg-slate-900 flex items-center justify-center">
        <div className="text-center">
          <p className="text-red-400 mb-4">{error}</p>
          <button
            onClick={() => navigate('/')}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg"
          >
            Go Back
          </button>
        </div>
      </div>
    );
  }

  if (!token) {
    return (
      <div className="min-h-screen bg-slate-900 flex items-center justify-center">
        <div className="text-white">Connecting to meeting...</div>
      </div>
    );
  }

  return (
    <ErrorBoundary>
      <div className="h-screen bg-slate-900">
        <LiveKitRoom
        token={token}
        serverUrl={LIVEKIT_URL}
        connectOptions={{ autoSubscribe: true }}
        onDisconnected={handleDisconnect}
        data-lk-theme="default"
        style={{ height: '100%' }}
      >
        <div className="h-full flex flex-col">
          {/* Header */}
          <div className="bg-slate-800 px-4 py-2 flex items-center justify-between">
            <div className="flex items-center gap-3">
              <span className="text-white font-semibold">Boom</span>
              <span className="text-slate-400 text-sm">|</span>
              <span className="text-slate-300 text-sm">{roomName}</span>
            </div>
            <div className="flex items-center gap-2">
              <BackgroundToggle />
              <CopyLinkButton roomName={roomName!} />
            </div>
          </div>

          {/* Main content */}
          <div className="flex-1 flex">
            {/* Video grid */}
            <div className="flex-1">
              <VideoConference />
            </div>

            {/* Transcription sidebar */}
            <div className="w-80 bg-slate-800 border-l border-slate-700">
              <Transcription roomName={roomName!} />
            </div>
          </div>

          <RoomAudioRenderer />
          </div>
        </LiveKitRoom>
      </div>
    </ErrorBoundary>
  );
}

function CopyLinkButton({ roomName }: { roomName: string }) {
  const [copied, setCopied] = useState(false);

  const copyLink = () => {
    const url = `${window.location.origin}/room/${roomName}`;
    navigator.clipboard.writeText(url);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <button
      onClick={copyLink}
      className="px-3 py-1 bg-slate-700 hover:bg-slate-600 text-slate-300 text-sm rounded transition-colors"
    >
      {copied ? 'Copied!' : 'Copy invite link'}
    </button>
  );
}
