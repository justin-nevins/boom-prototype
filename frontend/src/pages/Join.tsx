import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';

const BACKEND_URL = import.meta.env.VITE_BACKEND_URL || 'http://localhost:8080';

interface MeetingInfo {
  roomName: string;
  hostName: string;
  clientName: string;
  scheduledAt: string;
  status: string;
}

export default function Join() {
  const { roomName } = useParams<{ roomName: string }>();
  const navigate = useNavigate();
  const [meeting, setMeeting] = useState<MeetingInfo | null>(null);
  const [name, setName] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    fetchMeetingInfo();
  }, [roomName]);

  // Poll for status changes when meeting is scheduled (not yet active)
  useEffect(() => {
    if (!meeting || meeting.status === 'active') return;
    if (meeting.status !== 'scheduled') return;

    const interval = setInterval(async () => {
      try {
        const res = await fetch(`${BACKEND_URL}/api/join/${roomName}`);
        if (res.ok) {
          const data = await res.json();
          setMeeting(data);
        }
      } catch {
        // ignore polling errors
      }
    }, 5000);

    return () => clearInterval(interval);
  }, [meeting?.status, roomName]);

  const fetchMeetingInfo = async () => {
    try {
      const res = await fetch(`${BACKEND_URL}/api/join/${roomName}`);
      if (!res.ok) {
        setError('Meeting not found');
        return;
      }
      const data = await res.json();
      setMeeting(data);
      if (data.clientName) {
        setName(data.clientName);
      }
    } catch {
      setError('Failed to load meeting info');
    } finally {
      setLoading(false);
    }
  };

  const joinRoom = () => {
    if (!name.trim()) return;
    sessionStorage.setItem('participantName', name.trim());
    navigate(`/room/${roomName}`);
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-slate-900 flex items-center justify-center">
        <div className="text-slate-400">Loading meeting info...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="min-h-screen bg-slate-900 flex items-center justify-center px-4">
        <div className="text-center">
          <p className="text-red-400 mb-4">{error}</p>
          <button
            onClick={() => navigate('/')}
            className="px-4 py-2 bg-slate-700 text-white rounded-lg hover:bg-slate-600 transition-colors"
          >
            Go to Home
          </button>
        </div>
      </div>
    );
  }

  const scheduledDate = meeting ? new Date(meeting.scheduledAt) : null;
  const isActive = meeting?.status === 'active';
  const isPast = scheduledDate ? scheduledDate < new Date() : false;

  return (
    <div className="min-h-screen bg-slate-900 flex items-center justify-center px-4">
      <div className="w-full max-w-sm">
        {/* Logo */}
        <div className="flex items-center justify-center gap-2 mb-8">
          <svg className="w-8 h-8 text-[#2B88D9]" fill="currentColor" viewBox="0 0 24 24">
            <path d="M17 10.5V7c0-.55-.45-1-1-1H4c-.55 0-1 .45-1 1v10c0 .55.45 1 1 1h12c.55 0 1-.45 1-1v-3.5l4 4v-11l-4 4z"/>
          </svg>
          <span className="text-xl font-semibold text-white">Meet</span>
        </div>

        <div className="bg-slate-800 rounded-xl shadow-lg border border-slate-700 p-6">
          {/* Meeting info */}
          <div className="text-center mb-6">
            <h2 className="text-lg font-semibold text-white mb-1">
              {isActive ? 'Meeting is Live' : 'Upcoming Meeting'}
            </h2>
            <p className="text-slate-400 text-sm">
              Hosted by <span className="text-slate-300">{meeting?.hostName}</span>
            </p>
            {scheduledDate && (
              <p className="text-slate-500 text-sm mt-1">
                {scheduledDate.toLocaleDateString()} at{' '}
                {scheduledDate.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
              </p>
            )}
          </div>

          {/* Status indicator */}
          {isActive ? (
            <div className="flex items-center justify-center gap-2 mb-6 py-2 bg-green-900/30 border border-green-800 rounded-lg">
              <span className="w-2 h-2 bg-green-500 rounded-full animate-pulse" />
              <span className="text-green-400 text-sm font-medium">Meeting in progress</span>
            </div>
          ) : (
            <div className="flex items-center justify-center gap-2 mb-6 py-2 bg-slate-700/50 border border-slate-600 rounded-lg">
              <svg className="w-4 h-4 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <span className="text-slate-400 text-sm">
                {isPast ? 'Waiting for host to start...' : 'Not yet started'}
              </span>
            </div>
          )}

          {/* Name input */}
          <div className="mb-4">
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
              onKeyDown={(e) => e.key === 'Enter' && isActive && joinRoom()}
            />
          </div>

          {/* Join button */}
          {isActive ? (
            <button
              onClick={joinRoom}
              disabled={!name.trim()}
              className="w-full py-2.5 px-4 bg-[#0396A6] hover:bg-[#027d8a] text-white font-medium rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Join Meeting
            </button>
          ) : (
            <button
              disabled
              className="w-full py-2.5 px-4 bg-slate-600 text-slate-400 font-medium rounded-lg cursor-not-allowed"
            >
              Waiting for host...
            </button>
          )}

          {!isActive && (
            <p className="text-slate-500 text-xs text-center mt-3">
              This page will update automatically when the host starts the meeting.
            </p>
          )}
        </div>
      </div>
    </div>
  );
}
