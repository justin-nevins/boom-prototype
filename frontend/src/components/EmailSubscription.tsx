import { useState } from 'react';

const BACKEND_URL = import.meta.env.VITE_BACKEND_URL || 'http://localhost:8080';

interface EmailSubscriptionProps {
  roomName: string;
  participantName: string;
}

export default function EmailSubscription({ roomName, participantName }: EmailSubscriptionProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [email, setEmail] = useState('');
  const [isSubscribed, setIsSubscribed] = useState(false);
  const [subscribedEmail, setSubscribedEmail] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleSubscribe = async () => {
    if (!email || !email.includes('@')) {
      setError('Please enter a valid email');
      return;
    }

    setLoading(true);
    setError('');

    try {
      const res = await fetch(`${BACKEND_URL}/api/meetings/${roomName}/subscribe-email`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, participantName }),
      });

      if (!res.ok) {
        throw new Error('Failed to subscribe');
      }

      setIsSubscribed(true);
      setSubscribedEmail(email);
      setEmail('');
    } catch (err) {
      setError('Failed to subscribe. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const handleUnsubscribe = async () => {
    setLoading(true);
    try {
      await fetch(`${BACKEND_URL}/api/meetings/${roomName}/unsubscribe-email`, {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: subscribedEmail }),
      });
      setIsSubscribed(false);
      setSubscribedEmail('');
    } catch (err) {
      // Ignore errors on unsubscribe
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="relative">
      <button
        onClick={() => setIsOpen(!isOpen)}
        className={`flex items-center gap-2 px-3 py-2 rounded-lg transition-colors ${
          isSubscribed
            ? 'bg-green-600 hover:bg-green-700 text-white'
            : 'bg-slate-700 hover:bg-slate-600 text-slate-200'
        }`}
        title={isSubscribed ? `Subscribed: ${subscribedEmail}` : 'Get meeting summary via email'}
      >
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
        </svg>
        {isSubscribed && (
          <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
            <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
          </svg>
        )}
      </button>

      {isOpen && (
        <div className="absolute top-12 right-0 w-72 bg-slate-800 rounded-lg shadow-xl border border-slate-700 z-50">
          <div className="flex items-center justify-between px-4 py-3 border-b border-slate-700">
            <h3 className="text-white font-medium text-sm">Get Meeting Summary</h3>
            <button
              onClick={() => setIsOpen(false)}
              className="text-slate-400 hover:text-white transition-colors"
              aria-label="Close"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          <div className="p-4">
            {isSubscribed ? (
              <div className="space-y-3">
                <div className="flex items-center gap-2 text-green-400">
                  <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
                    <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                  </svg>
                  <span className="text-sm">Subscribed</span>
                </div>
                <p className="text-slate-400 text-xs">
                  Summary will be sent to <strong className="text-white">{subscribedEmail}</strong> when the meeting ends.
                </p>
                <button
                  onClick={handleUnsubscribe}
                  disabled={loading}
                  className="w-full px-3 py-2 text-sm text-slate-400 hover:text-white border border-slate-600 rounded-lg hover:border-slate-500 transition-colors disabled:opacity-50"
                >
                  {loading ? 'Unsubscribing...' : 'Unsubscribe'}
                </button>
              </div>
            ) : (
              <div className="space-y-3">
                <p className="text-slate-400 text-xs">
                  Receive the meeting notes and transcript summary via email when the meeting ends.
                </p>

                {error && (
                  <div className="text-red-400 text-xs">{error}</div>
                )}

                <input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="Enter your email"
                  className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white text-sm placeholder-slate-400 focus:outline-none focus:border-blue-500"
                  onKeyDown={(e) => e.key === 'Enter' && handleSubscribe()}
                />

                <button
                  onClick={handleSubscribe}
                  disabled={loading || !email}
                  className="w-full px-3 py-2 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {loading ? 'Subscribing...' : 'Subscribe'}
                </button>
              </div>
            )}
          </div>

          <div className="px-4 py-2 border-t border-slate-700">
            <p className="text-slate-500 text-xs">
              Your email is only used to send meeting summaries.
            </p>
          </div>
        </div>
      )}
    </div>
  );
}
