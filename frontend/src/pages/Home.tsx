import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';

const BACKEND_URL = import.meta.env.VITE_BACKEND_URL || 'http://localhost:8080';

interface ScheduledMeeting {
  id: number;
  roomName: string;
  clientName: string;
  clientEmail: string;
  scheduledAt: string;
  status: string;
  inviteLink: string;
}

export default function Home() {
  const navigate = useNavigate();
  const { user, isAuthenticated, token, logout, loading: authLoading } = useAuth();
  const [activeTab, setActiveTab] = useState<'join' | 'schedule'>('join');

  // Join meeting state
  const [joinCode, setJoinCode] = useState('');
  const [name, setName] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  // Schedule form state
  const [scheduleForm, setScheduleForm] = useState({
    clientName: '',
    clientEmail: '',
    date: '',
    time: '',
  });
  const [scheduling, setScheduling] = useState(false);
  const [scheduledLink, setScheduledLink] = useState('');

  // My meetings state
  const [myMeetings, setMyMeetings] = useState<ScheduledMeeting[]>([]);
  const [copiedLink, setCopiedLink] = useState<number | null>(null);

  useEffect(() => {
    if (isAuthenticated && token) {
      fetchMyMeetings();
    }
  }, [isAuthenticated, token]);

  const fetchMyMeetings = async () => {
    try {
      const res = await fetch(`${BACKEND_URL}/api/scheduled-meetings`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) {
        const data = await res.json();
        setMyMeetings(data);
      }
    } catch (err) {
      console.error('Failed to fetch meetings:', err);
    }
  };

  const createMeeting = async () => {
    setLoading(true);
    setError('');
    try {
      const res = await fetch(`${BACKEND_URL}/api/rooms`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify({ name: '' }),
      });
      if (!res.ok) throw new Error('Failed to create room');
      const data = await res.json();
      sessionStorage.setItem('participantName', name || user?.name || 'Guest');
      navigate(`/room/${data.roomName}`);
    } catch (err) {
      console.error('Failed to create room:', err);
      setError('Failed to create meeting. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const joinMeeting = () => {
    if (!joinCode.trim()) return;
    sessionStorage.setItem('participantName', name || 'Guest');
    navigate(`/room/${joinCode.trim()}`);
  };

  const scheduleMeeting = async (e: React.FormEvent) => {
    e.preventDefault();
    setScheduling(true);
    setScheduledLink('');
    try {
      const scheduledAt = new Date(`${scheduleForm.date}T${scheduleForm.time}`).toISOString();
      const res = await fetch(`${BACKEND_URL}/api/scheduled-meetings`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          clientName: scheduleForm.clientName,
          clientEmail: scheduleForm.clientEmail,
          scheduledAt,
        }),
      });
      if (!res.ok) throw new Error('Failed to schedule meeting');
      const data = await res.json();
      setScheduledLink(data.inviteLink);
      setScheduleForm({ clientName: '', clientEmail: '', date: '', time: '' });
      fetchMyMeetings();
    } catch (err) {
      console.error('Failed to schedule:', err);
    } finally {
      setScheduling(false);
    }
  };

  const cancelMeeting = async (id: number) => {
    try {
      await fetch(`${BACKEND_URL}/api/scheduled-meetings/${id}`, {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${token}` },
      });
      fetchMyMeetings();
    } catch (err) {
      console.error('Failed to cancel:', err);
    }
  };

  const startScheduledMeeting = async (meeting: ScheduledMeeting) => {
    try {
      const res = await fetch(`${BACKEND_URL}/api/scheduled-meetings/${meeting.id}/start`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) throw new Error('Failed to start meeting');
      sessionStorage.setItem('participantName', user?.name || 'Host');
      navigate(`/room/${meeting.roomName}`);
    } catch (err) {
      console.error('Failed to start meeting:', err);
    }
  };

  const copyInviteLink = (link: string, id: number) => {
    navigator.clipboard.writeText(link);
    setCopiedLink(id);
    setTimeout(() => setCopiedLink(null), 2000);
  };

  return (
    <div className="min-h-screen bg-slate-900 relative">
      {/* Background Image */}
      <div
        className="absolute inset-0 bg-cover bg-center bg-no-repeat opacity-20"
        style={{ backgroundImage: 'url(https://images.unsplash.com/photo-1519681393784-d120267933ba?w=1920&q=80)' }}
      />
      <div className="absolute inset-0 bg-gradient-to-b from-slate-900/50 via-slate-900/70 to-slate-900" />

      {/* Header */}
      <header className="relative z-10 flex items-center justify-between px-6 py-4 bg-slate-800/80 backdrop-blur-sm border-b border-slate-700">
        <div className="flex items-center gap-2">
          <svg className="w-8 h-8 text-[#2B88D9]" fill="currentColor" viewBox="0 0 24 24">
            <path d="M17 10.5V7c0-.55-.45-1-1-1H4c-.55 0-1 .45-1 1v10c0 .55.45 1 1 1h12c.55 0 1-.45 1-1v-3.5l4 4v-11l-4 4z"/>
          </svg>
          <span className="text-xl font-semibold text-white">Meet</span>
        </div>
        {!authLoading && (
          isAuthenticated ? (
            <div className="flex items-center gap-3">
              <span className="text-sm text-slate-300">{user?.name}</span>
              <button
                onClick={logout}
                className="px-4 py-2 text-sm font-medium text-slate-400 hover:text-white hover:bg-slate-700 rounded-lg transition-colors"
              >
                Sign Out
              </button>
            </div>
          ) : (
            <button
              onClick={() => navigate('/login')}
              className="px-4 py-2 text-sm font-medium text-[#2B88D9] hover:bg-slate-700 rounded-lg transition-colors"
            >
              Sign In
            </button>
          )
        )}
      </header>

      {/* Main Content */}
      <main className="relative z-10 flex flex-col items-center px-4 py-12">
        {/* Hero */}
        <div className="text-center mb-10 max-w-lg">
          <h1 className="text-3xl font-bold text-white mb-3">
            Professional Video Meetings
          </h1>
          <p className="text-slate-400">
            Connect with anyone, anywhere. Secure video calls with automatic transcription and AI-powered notes.
          </p>
        </div>

        {/* Tabbed Card */}
        <div className="w-full max-w-md bg-slate-800 rounded-xl shadow-lg overflow-hidden border border-slate-700">
          {/* Tabs */}
          <div className="flex border-b border-slate-700">
            <button
              onClick={() => setActiveTab('join')}
              className={`flex-1 py-3 text-sm font-medium transition-colors ${
                activeTab === 'join'
                  ? 'text-[#2B88D9] border-b-2 border-[#2B88D9] bg-slate-800'
                  : 'text-slate-400 hover:text-slate-300 bg-slate-850'
              }`}
            >
              {isAuthenticated ? 'Meetings' : 'Join Meeting'}
            </button>
            {isAuthenticated && (
              <button
                onClick={() => { setActiveTab('schedule'); setScheduledLink(''); }}
                className={`flex-1 py-3 text-sm font-medium transition-colors ${
                  activeTab === 'schedule'
                    ? 'text-[#2B88D9] border-b-2 border-[#2B88D9] bg-slate-800'
                    : 'text-slate-400 hover:text-slate-300 bg-slate-850'
                }`}
              >
                Schedule Meeting
              </button>
            )}
          </div>

          {/* Tab Content */}
          <div className="p-6">
            {activeTab === 'join' ? (
              <div className="space-y-4">
                {error && (
                  <div className="p-3 bg-red-900/50 border border-red-700 rounded-lg text-red-300 text-sm">
                    {error}
                  </div>
                )}

                {/* Name Input */}
                <div>
                  <label className="block text-sm font-medium text-slate-300 mb-1.5">
                    Your name
                  </label>
                  <input
                    type="text"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder={isAuthenticated ? user?.name || 'Enter your name' : 'Enter your name'}
                    className="w-full px-4 py-2.5 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-[#2B88D9] focus:border-transparent"
                  />
                </div>

                {/* Meeting Code + Join */}
                <div>
                  <label className="block text-sm font-medium text-slate-300 mb-1.5">
                    Meeting code
                  </label>
                  <div className="flex gap-2">
                    <input
                      type="text"
                      value={joinCode}
                      onChange={(e) => setJoinCode(e.target.value)}
                      placeholder="Enter meeting code"
                      className="flex-1 px-4 py-2.5 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-[#2B88D9] focus:border-transparent"
                      onKeyDown={(e) => e.key === 'Enter' && joinMeeting()}
                    />
                    <button
                      onClick={joinMeeting}
                      className="px-5 py-2.5 bg-[#0396A6] hover:bg-[#027d8a] text-white font-medium rounded-lg transition-colors"
                    >
                      Join
                    </button>
                  </div>
                </div>

                {/* Start New Meeting - only for authenticated users */}
                {isAuthenticated && (
                  <>
                    <div className="relative my-2">
                      <div className="absolute inset-0 flex items-center">
                        <div className="w-full border-t border-slate-600"></div>
                      </div>
                      <div className="relative flex justify-center text-sm">
                        <span className="px-3 bg-slate-800 text-slate-400">or start new</span>
                      </div>
                    </div>

                    <button
                      onClick={createMeeting}
                      disabled={loading}
                      className="w-full py-2.5 px-4 bg-[#2B88D9] hover:bg-[#2477c2] text-white font-medium rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      {loading ? 'Creating...' : 'Start New Meeting'}
                    </button>
                  </>
                )}
              </div>
            ) : (
              /* Schedule Meeting Tab */
              <div className="space-y-4">
                {scheduledLink ? (
                  <div className="text-center py-4">
                    <svg className="w-12 h-12 text-[#0396A6] mx-auto mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                    </svg>
                    <h3 className="text-lg font-medium text-white mb-2">Meeting Scheduled</h3>
                    <p className="text-slate-400 text-sm mb-4">Send this link to your client:</p>
                    <div className="flex items-center gap-2 bg-slate-700 rounded-lg p-3">
                      <input
                        type="text"
                        readOnly
                        value={scheduledLink}
                        className="flex-1 bg-transparent text-white text-sm focus:outline-none"
                      />
                      <button
                        onClick={() => { navigator.clipboard.writeText(scheduledLink); }}
                        className="px-3 py-1 bg-[#2B88D9] hover:bg-[#2477c2] text-white text-sm rounded transition-colors"
                      >
                        Copy
                      </button>
                    </div>
                    <button
                      onClick={() => setScheduledLink('')}
                      className="mt-4 text-slate-400 text-sm hover:text-slate-300 transition-colors"
                    >
                      Schedule another
                    </button>
                  </div>
                ) : (
                  <form onSubmit={scheduleMeeting} className="space-y-4">
                    <div className="grid grid-cols-2 gap-3">
                      <div>
                        <label className="block text-sm font-medium text-slate-300 mb-1.5">
                          Client Name <span className="text-[#D93D1A]">*</span>
                        </label>
                        <input
                          type="text"
                          required
                          value={scheduleForm.clientName}
                          onChange={(e) => setScheduleForm({ ...scheduleForm, clientName: e.target.value })}
                          placeholder="Client name"
                          className="w-full px-3 py-2.5 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-[#2B88D9] focus:border-transparent text-sm"
                        />
                      </div>
                      <div>
                        <label className="block text-sm font-medium text-slate-300 mb-1.5">
                          Client Email
                        </label>
                        <input
                          type="email"
                          value={scheduleForm.clientEmail}
                          onChange={(e) => setScheduleForm({ ...scheduleForm, clientEmail: e.target.value })}
                          placeholder="client@email.com"
                          className="w-full px-3 py-2.5 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-[#2B88D9] focus:border-transparent text-sm"
                        />
                      </div>
                    </div>

                    <div className="grid grid-cols-2 gap-3">
                      <div>
                        <label className="block text-sm font-medium text-slate-300 mb-1.5">
                          Date <span className="text-[#D93D1A]">*</span>
                        </label>
                        <input
                          type="date"
                          required
                          value={scheduleForm.date}
                          onChange={(e) => setScheduleForm({ ...scheduleForm, date: e.target.value })}
                          className="w-full px-3 py-2.5 bg-slate-700 border border-slate-600 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-[#2B88D9] focus:border-transparent text-sm"
                        />
                      </div>
                      <div>
                        <label className="block text-sm font-medium text-slate-300 mb-1.5">
                          Time <span className="text-[#D93D1A]">*</span>
                        </label>
                        <input
                          type="time"
                          required
                          value={scheduleForm.time}
                          onChange={(e) => setScheduleForm({ ...scheduleForm, time: e.target.value })}
                          className="w-full px-3 py-2.5 bg-slate-700 border border-slate-600 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-[#2B88D9] focus:border-transparent text-sm"
                        />
                      </div>
                    </div>

                    <button
                      type="submit"
                      disabled={scheduling}
                      className="w-full py-2.5 px-4 bg-[#2B88D9] hover:bg-[#2477c2] text-white font-medium rounded-lg transition-colors disabled:opacity-50"
                    >
                      {scheduling ? 'Scheduling...' : 'Schedule Meeting'}
                    </button>
                  </form>
                )}
              </div>
            )}
          </div>
        </div>

        {/* My Meetings - only for authenticated users */}
        {isAuthenticated && myMeetings.length > 0 && (
          <div className="w-full max-w-md mt-6">
            <h3 className="text-white font-semibold mb-3">Upcoming Meetings</h3>
            <div className="space-y-2">
              {myMeetings.map((m) => (
                <div key={m.id} className="bg-slate-800 border border-slate-700 rounded-lg p-4">
                  <div className="flex items-start justify-between">
                    <div>
                      <p className="text-white font-medium">{m.clientName || 'No client name'}</p>
                      <p className="text-slate-400 text-sm">
                        {new Date(m.scheduledAt).toLocaleDateString()} at{' '}
                        {new Date(m.scheduledAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                      </p>
                      {m.clientEmail && (
                        <p className="text-slate-500 text-xs mt-0.5">{m.clientEmail}</p>
                      )}
                    </div>
                    <span className={`text-xs px-2 py-0.5 rounded ${
                      m.status === 'active'
                        ? 'bg-green-900/50 text-green-400'
                        : 'bg-slate-700 text-slate-400'
                    }`}>
                      {m.status}
                    </span>
                  </div>
                  <div className="flex gap-2 mt-3">
                    <button
                      onClick={() => startScheduledMeeting(m)}
                      className="flex-1 py-1.5 px-3 bg-[#0396A6] hover:bg-[#027d8a] text-white text-sm rounded transition-colors"
                    >
                      Start
                    </button>
                    <button
                      onClick={() => copyInviteLink(m.inviteLink, m.id)}
                      className="py-1.5 px-3 bg-slate-700 hover:bg-slate-600 text-slate-300 text-sm rounded transition-colors"
                    >
                      {copiedLink === m.id ? 'Copied!' : 'Copy Link'}
                    </button>
                    <button
                      onClick={() => cancelMeeting(m.id)}
                      className="py-1.5 px-3 bg-slate-700 hover:bg-red-900/50 text-slate-400 hover:text-red-400 text-sm rounded transition-colors"
                    >
                      Cancel
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Features */}
        <div className="flex flex-wrap justify-center gap-8 mt-12 max-w-lg">
          <div className="flex flex-col items-center text-center">
            <div className="w-12 h-12 bg-slate-800 rounded-full flex items-center justify-center mb-2 border border-slate-700">
              <svg className="w-6 h-6 text-[#6394BF]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
              </svg>
            </div>
            <span className="text-sm font-medium text-slate-300">Secure</span>
            <span className="text-xs text-slate-500">End-to-end encrypted</span>
          </div>
          <div className="flex flex-col items-center text-center">
            <div className="w-12 h-12 bg-slate-800 rounded-full flex items-center justify-center mb-2 border border-slate-700">
              <svg className="w-6 h-6 text-[#6394BF]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
            </div>
            <span className="text-sm font-medium text-slate-300">Transcribed</span>
            <span className="text-xs text-slate-500">AI-powered notes</span>
          </div>
          <div className="flex flex-col items-center text-center">
            <div className="w-12 h-12 bg-slate-800 rounded-full flex items-center justify-center mb-2 border border-slate-700">
              <svg className="w-6 h-6 text-[#6394BF]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
              </svg>
            </div>
            <span className="text-sm font-medium text-slate-300">Group Ready</span>
            <span className="text-xs text-slate-500">Multiple participants</span>
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer className="relative z-10 text-center py-6 text-slate-500 text-sm">
        Professional video meetings, simplified.
      </footer>
    </div>
  );
}
