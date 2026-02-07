import { useEffect, useState, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  LiveKitRoom,
  VideoConference,
  RoomAudioRenderer,
  useRoomContext,
} from '@livekit/components-react';
import '@livekit/components-styles';
import { VideoPresets, RoomOptions, DisconnectReason } from 'livekit-client';
import ErrorBoundary from '../components/ErrorBoundary';
import BackgroundToggle from '../components/BackgroundToggle';
import NotesModal from '../components/NotesModal';
import EmailSubscription from '../components/EmailSubscription';

const BACKEND_URL = import.meta.env.VITE_BACKEND_URL || 'http://localhost:8080';
const LIVEKIT_URL = import.meta.env.VITE_LIVEKIT_URL;

// Video quality settings
const roomOptions: RoomOptions = {
  videoCaptureDefaults: {
    resolution: VideoPresets.h720,
    facingMode: 'user',
  },
  publishDefaults: {
    simulcast: true,
    videoSimulcastLayers: [
      VideoPresets.h180,
      VideoPresets.h360,
      VideoPresets.h720,
    ],
  },
  adaptiveStream: true,
  disconnectOnPageLeave: false,
};

const connectOptions = {
  autoSubscribe: true,
  peerConnectionTimeout: 45_000,
  rtcConfig: {
    iceTransportPolicy: 'all' as RTCIceTransportPolicy,
  },
};

type TranscriptionStatus = 'idle' | 'transcribing' | 'processing' | 'completed' | 'failed';

export default function Room() {
  const { roomName } = useParams<{ roomName: string }>();
  const navigate = useNavigate();
  const [token, setToken] = useState<string>('');
  const [error, setError] = useState<string>('');
  const [wasConnected, setWasConnected] = useState(false);

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

  const handleDisconnect = (reason?: DisconnectReason) => {
    console.log('LiveKit disconnected, reason:', reason, 'wasConnected:', wasConnected);
    if (wasConnected) {
      // Normal disconnect after being in a meeting
      navigate('/');
    } else {
      // Never fully connected — show error instead of silent redirect
      setError('Failed to connect to meeting. The video connection could not be established.');
    }
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
          options={roomOptions}
          connectOptions={connectOptions}
          onConnected={() => setWasConnected(true)}
          onDisconnected={handleDisconnect}
          data-lk-theme="default"
          style={{ height: '100%' }}
        >
          <RoomContent roomName={roomName!} onLeave={handleDisconnect} />
        </LiveKitRoom>
      </div>
    </ErrorBoundary>
  );
}

function RoomContent({ roomName, onLeave }: { roomName: string; onLeave: () => void }) {
  const room = useRoomContext();
  const [transcriptionStatus, setTranscriptionStatus] = useState<TranscriptionStatus>('idle');
  const [processingStage, setProcessingStage] = useState<string>('');
  const [notes, setNotes] = useState<string>('');
  const [showNotes, setShowNotes] = useState(false);
  const [endingMeeting, setEndingMeeting] = useState(false);

  // Start transcription when room connects
  useEffect(() => {
    if (room.state === 'connected') {
      startTranscription();
    }
  }, [room.state]);

  const startTranscription = async () => {
    try {
      const res = await fetch(`${BACKEND_URL}/api/meetings/${roomName}/start-transcription`, {
        method: 'POST',
      });
      const data = await res.json();

      if (data.status === 'transcribing' || data.status === 'already_joined') {
        setTranscriptionStatus('transcribing');
        console.log('Transcription started for room:', roomName);
      } else {
        console.error('Failed to start transcription:', data);
      }
    } catch (err) {
      console.error('Error starting transcription:', err);
    }
  };

  const endMeeting = async () => {
    setEndingMeeting(true);
    setTranscriptionStatus('processing');
    setProcessingStage('Generating notes...');

    try {
      // End transcription and generate notes
      const res = await fetch(`${BACKEND_URL}/api/meetings/${roomName}/end-transcription`, {
        method: 'POST',
      });
      const data = await res.json();

      if (data.status === 'processing') {
        setProcessingStage('Processing transcript...');

        // Poll for notes completion
        pollForNotes();
      } else {
        setTranscriptionStatus('failed');
        setProcessingStage('Failed to end transcription');
      }
    } catch (err) {
      console.error('Error ending meeting:', err);
      setTranscriptionStatus('failed');
      setProcessingStage('Error ending meeting');
    }
  };

  const pollForNotes = useCallback(async () => {
    const maxAttempts = 60; // 5 minutes max (5 sec intervals)
    let attempts = 0;

    const poll = async () => {
      try {
        const res = await fetch(`${BACKEND_URL}/api/meetings/${roomName}/notes`);

        if (res.ok) {
          const data = await res.json();
          if (data.markdown) {
            setNotes(data.markdown);
            setTranscriptionStatus('completed');
            setProcessingStage('');
            setShowNotes(true);
            return;
          }
        }

        attempts++;
        if (attempts < maxAttempts) {
          // Update processing stage message
          if (attempts < 10) {
            setProcessingStage('Processing transcript...');
          } else if (attempts < 30) {
            setProcessingStage('Generating meeting notes...');
          } else {
            setProcessingStage('Almost done...');
          }

          setTimeout(poll, 5000);
        } else {
          setTranscriptionStatus('failed');
          setProcessingStage('Timed out waiting for notes');
        }
      } catch (err) {
        attempts++;
        if (attempts < maxAttempts) {
          setTimeout(poll, 5000);
        }
      }
    };

    poll();
  }, [roomName]);

  const handleLeave = () => {
    room.disconnect();
    onLeave();
  };

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="bg-slate-800 px-4 py-2 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <span className="text-white font-semibold">Boom</span>
          <span className="text-slate-400 text-sm">|</span>
          <span className="text-slate-300 text-sm">{roomName}</span>

          {/* Transcription indicator */}
          {transcriptionStatus === 'transcribing' && (
            <div className="flex items-center gap-1.5 ml-2">
              <span className="w-2 h-2 bg-green-500 rounded-full animate-pulse" />
              <span className="text-green-400 text-xs">Live Transcribing</span>
            </div>
          )}
        </div>

        <div className="flex items-center gap-2">
          <BackgroundToggle />
          <EmailSubscription 
            roomName={roomName} 
            participantName={sessionStorage.getItem('participantName') || 'Guest'} 
          />
          <CopyLinkButton roomName={roomName} />

          {/* End Meeting Button */}
          {!endingMeeting ? (
            <button
              onClick={endMeeting}
              className="px-3 py-1 bg-red-600 hover:bg-red-500 text-white text-sm rounded transition-colors flex items-center gap-1.5"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
              </svg>
              End Meeting
            </button>
          ) : (
            <button
              disabled
              className="px-3 py-1 bg-slate-600 text-slate-300 text-sm rounded flex items-center gap-1.5"
            >
              <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
              </svg>
              Processing...
            </button>
          )}
        </div>
      </div>

      {/* Processing overlay */}
      {endingMeeting && transcriptionStatus === 'processing' && (
        <div className="absolute inset-0 bg-slate-900/90 z-50 flex items-center justify-center">
          <div className="text-center">
            <div className="w-16 h-16 border-4 border-blue-500 border-t-transparent rounded-full animate-spin mx-auto mb-4" />
            <h3 className="text-white text-xl font-semibold mb-2">Processing Meeting</h3>
            <p className="text-slate-400">{processingStage}</p>
            <p className="text-slate-500 text-sm mt-4">This may take a few minutes...</p>
          </div>
        </div>
      )}

      {/* Main content - full width now, no sidebar */}
      <div className="flex-1">
        <VideoConference />
      </div>

      <RoomAudioRenderer />

      {/* Notes Modal */}
      <NotesModal
        isOpen={showNotes}
        onClose={() => {
          setShowNotes(false);
          handleLeave();
        }}
        markdown={notes}
        isLoading={false}
      />
    </div>
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
