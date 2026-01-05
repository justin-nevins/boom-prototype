import { useState } from 'react';
import { useNavigate } from 'react-router-dom';

const BACKEND_URL = import.meta.env.VITE_BACKEND_URL || 'http://localhost:8080';
const APPOINTMENT_WEBHOOK = import.meta.env.VITE_APPOINTMENT_WEBHOOK || '';

export default function Home() {
  const navigate = useNavigate();
  const [activeTab, setActiveTab] = useState<'join' | 'appointment'>('join');

  // Join meeting state
  const [joinCode, setJoinCode] = useState('');
  const [name, setName] = useState('');
  const [loading, setLoading] = useState(false);

  // Appointment form state
  const [appointmentForm, setAppointmentForm] = useState({
    name: '',
    email: '',
    date: '',
    time: '',
    message: '',
  });
  const [submitting, setSubmitting] = useState(false);
  const [submitted, setSubmitted] = useState(false);

  const createMeeting = async () => {
    setLoading(true);
    try {
      const res = await fetch(`${BACKEND_URL}/api/rooms`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: '' }),
      });
      const data = await res.json();
      sessionStorage.setItem('participantName', name || 'Guest');
      navigate(`/room/${data.roomName}`);
    } catch (err) {
      console.error('Failed to create room:', err);
    } finally {
      setLoading(false);
    }
  };

  const joinMeeting = () => {
    if (!joinCode.trim()) return;
    sessionStorage.setItem('participantName', name || 'Guest');
    navigate(`/room/${joinCode.trim()}`);
  };

  const submitAppointment = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!APPOINTMENT_WEBHOOK) {
      console.error('Appointment webhook not configured');
      return;
    }
    setSubmitting(true);
    try {
      await fetch(APPOINTMENT_WEBHOOK, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(appointmentForm),
      });
      setSubmitted(true);
      setAppointmentForm({ name: '', email: '', date: '', time: '', message: '' });
    } catch (err) {
      console.error('Failed to submit appointment:', err);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="min-h-screen bg-slate-50">
      {/* Header */}
      <header className="flex items-center justify-between px-6 py-4 bg-white border-b border-slate-200">
        <div className="flex items-center gap-2">
          <svg className="w-8 h-8 text-[#2B88D9]" fill="currentColor" viewBox="0 0 24 24">
            <path d="M17 10.5V7c0-.55-.45-1-1-1H4c-.55 0-1 .45-1 1v10c0 .55.45 1 1 1h12c.55 0 1-.45 1-1v-3.5l4 4v-11l-4 4z"/>
          </svg>
          <span className="text-xl font-semibold text-slate-800">Meet</span>
        </div>
        <button className="px-4 py-2 text-sm font-medium text-[#2B88D9] hover:bg-slate-100 rounded-lg transition-colors">
          Sign In
        </button>
      </header>

      {/* Main Content */}
      <main className="flex flex-col items-center px-4 py-12">
        {/* Hero */}
        <div className="text-center mb-10 max-w-lg">
          <h1 className="text-3xl font-bold text-slate-800 mb-3">
            Professional Video Meetings
          </h1>
          <p className="text-slate-600">
            Connect with anyone, anywhere. Secure video calls with automatic transcription and AI-powered notes.
          </p>
        </div>

        {/* Tabbed Card */}
        <div className="w-full max-w-md bg-white rounded-xl shadow-lg overflow-hidden">
          {/* Tabs */}
          <div className="flex border-b border-slate-200">
            <button
              onClick={() => setActiveTab('join')}
              className={`flex-1 py-3 text-sm font-medium transition-colors ${
                activeTab === 'join'
                  ? 'text-[#2B88D9] border-b-2 border-[#2B88D9] bg-white'
                  : 'text-slate-500 hover:text-slate-700 bg-slate-50'
              }`}
            >
              Join Meeting
            </button>
            <button
              onClick={() => { setActiveTab('appointment'); setSubmitted(false); }}
              className={`flex-1 py-3 text-sm font-medium transition-colors ${
                activeTab === 'appointment'
                  ? 'text-[#2B88D9] border-b-2 border-[#2B88D9] bg-white'
                  : 'text-slate-500 hover:text-slate-700 bg-slate-50'
              }`}
            >
              Request Appointment
            </button>
          </div>

          {/* Tab Content */}
          <div className="p-6">
            {activeTab === 'join' ? (
              <div className="space-y-4">
                {/* Name Input */}
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1.5">
                    Your name
                  </label>
                  <input
                    type="text"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder="Enter your name"
                    className="w-full px-4 py-2.5 bg-white border border-slate-300 rounded-lg text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-[#2B88D9] focus:border-transparent"
                  />
                </div>

                {/* Meeting Code + Join */}
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1.5">
                    Meeting code
                  </label>
                  <div className="flex gap-2">
                    <input
                      type="text"
                      value={joinCode}
                      onChange={(e) => setJoinCode(e.target.value)}
                      placeholder="Enter meeting code"
                      className="flex-1 px-4 py-2.5 bg-white border border-slate-300 rounded-lg text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-[#2B88D9] focus:border-transparent"
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

                {/* Divider */}
                <div className="relative my-2">
                  <div className="absolute inset-0 flex items-center">
                    <div className="w-full border-t border-slate-200"></div>
                  </div>
                  <div className="relative flex justify-center text-sm">
                    <span className="px-3 bg-white text-slate-400">or start new</span>
                  </div>
                </div>

                {/* Create Meeting Button */}
                <button
                  onClick={createMeeting}
                  disabled={loading}
                  className="w-full py-2.5 px-4 bg-[#2B88D9] hover:bg-[#2477c2] text-white font-medium rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {loading ? 'Creating...' : 'Start New Meeting'}
                </button>
              </div>
            ) : (
              <form onSubmit={submitAppointment} className="space-y-4">
                {submitted ? (
                  <div className="text-center py-8">
                    <svg className="w-12 h-12 text-[#0396A6] mx-auto mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                    </svg>
                    <h3 className="text-lg font-medium text-slate-800 mb-1">Request Sent</h3>
                    <p className="text-slate-600 text-sm">We'll be in touch soon.</p>
                  </div>
                ) : (
                  <>
                    {/* Name + Email Row */}
                    <div className="grid grid-cols-2 gap-3">
                      <div>
                        <label className="block text-sm font-medium text-slate-700 mb-1.5">
                          Name <span className="text-[#D93D1A]">*</span>
                        </label>
                        <input
                          type="text"
                          required
                          value={appointmentForm.name}
                          onChange={(e) => setAppointmentForm({ ...appointmentForm, name: e.target.value })}
                          placeholder="Your name"
                          className="w-full px-3 py-2.5 bg-white border border-slate-300 rounded-lg text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-[#2B88D9] focus:border-transparent text-sm"
                        />
                      </div>
                      <div>
                        <label className="block text-sm font-medium text-slate-700 mb-1.5">
                          Email <span className="text-[#D93D1A]">*</span>
                        </label>
                        <input
                          type="email"
                          required
                          value={appointmentForm.email}
                          onChange={(e) => setAppointmentForm({ ...appointmentForm, email: e.target.value })}
                          placeholder="you@email.com"
                          className="w-full px-3 py-2.5 bg-white border border-slate-300 rounded-lg text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-[#2B88D9] focus:border-transparent text-sm"
                        />
                      </div>
                    </div>

                    {/* Date + Time Row */}
                    <div className="grid grid-cols-2 gap-3">
                      <div>
                        <label className="block text-sm font-medium text-slate-700 mb-1.5">
                          Preferred Date <span className="text-[#D93D1A]">*</span>
                        </label>
                        <input
                          type="date"
                          required
                          value={appointmentForm.date}
                          onChange={(e) => setAppointmentForm({ ...appointmentForm, date: e.target.value })}
                          className="w-full px-3 py-2.5 bg-white border border-slate-300 rounded-lg text-slate-800 focus:outline-none focus:ring-2 focus:ring-[#2B88D9] focus:border-transparent text-sm"
                        />
                      </div>
                      <div>
                        <label className="block text-sm font-medium text-slate-700 mb-1.5">
                          Preferred Time <span className="text-[#D93D1A]">*</span>
                        </label>
                        <input
                          type="time"
                          required
                          value={appointmentForm.time}
                          onChange={(e) => setAppointmentForm({ ...appointmentForm, time: e.target.value })}
                          className="w-full px-3 py-2.5 bg-white border border-slate-300 rounded-lg text-slate-800 focus:outline-none focus:ring-2 focus:ring-[#2B88D9] focus:border-transparent text-sm"
                        />
                      </div>
                    </div>

                    {/* Message */}
                    <div>
                      <label className="block text-sm font-medium text-slate-700 mb-1.5">
                        Message
                      </label>
                      <textarea
                        value={appointmentForm.message}
                        onChange={(e) => setAppointmentForm({ ...appointmentForm, message: e.target.value })}
                        placeholder="What would you like to discuss?"
                        rows={3}
                        className="w-full px-3 py-2.5 bg-white border border-slate-300 rounded-lg text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-[#2B88D9] focus:border-transparent text-sm resize-none"
                      />
                    </div>

                    {/* Submit */}
                    <button
                      type="submit"
                      disabled={submitting}
                      className="w-full py-2.5 px-4 bg-[#D93D1A] hover:bg-[#c23516] text-white font-medium rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      {submitting ? 'Sending...' : 'Request Appointment'}
                    </button>
                  </>
                )}
              </form>
            )}
          </div>
        </div>

        {/* Features */}
        <div className="flex flex-wrap justify-center gap-8 mt-12 max-w-lg">
          <div className="flex flex-col items-center text-center">
            <div className="w-12 h-12 bg-slate-100 rounded-full flex items-center justify-center mb-2">
              <svg className="w-6 h-6 text-[#6394BF]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
              </svg>
            </div>
            <span className="text-sm font-medium text-slate-700">Secure</span>
            <span className="text-xs text-slate-500">End-to-end encrypted</span>
          </div>
          <div className="flex flex-col items-center text-center">
            <div className="w-12 h-12 bg-slate-100 rounded-full flex items-center justify-center mb-2">
              <svg className="w-6 h-6 text-[#6394BF]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
            </div>
            <span className="text-sm font-medium text-slate-700">Transcribed</span>
            <span className="text-xs text-slate-500">AI-powered notes</span>
          </div>
          <div className="flex flex-col items-center text-center">
            <div className="w-12 h-12 bg-slate-100 rounded-full flex items-center justify-center mb-2">
              <svg className="w-6 h-6 text-[#6394BF]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
              </svg>
            </div>
            <span className="text-sm font-medium text-slate-700">Group Ready</span>
            <span className="text-xs text-slate-500">Multiple participants</span>
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer className="text-center py-6 text-slate-400 text-sm">
        Professional video meetings, simplified.
      </footer>
    </div>
  );
}
