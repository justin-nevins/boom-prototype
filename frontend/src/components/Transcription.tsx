import { useEffect, useState, useRef, useCallback } from 'react';

const BACKEND_URL = import.meta.env.VITE_BACKEND_URL || 'http://localhost:8080';
const MAX_RETRIES = 5;

interface TranscriptEntry {
  id: string;
  speaker: string;
  text: string;
  timestamp: number;
}

export default function Transcription({ roomName }: { roomName: string }) {
  const [transcripts, setTranscripts] = useState<TranscriptEntry[]>([]);
  const [connected, setConnected] = useState(false);
  const [reconnecting, setReconnecting] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const retryCountRef = useRef(0);
  const reconnectTimeoutRef = useRef<number | null>(null);

  const connect = useCallback(() => {
    // Clean up existing connection
    if (wsRef.current) {
      wsRef.current.close();
    }

    const wsUrl = BACKEND_URL.replace('http', 'ws').replace('https', 'wss');
    const ws = new WebSocket(`${wsUrl}/ws/transcription/${roomName}`);
    wsRef.current = ws;

    ws.onopen = () => {
      setConnected(true);
      setReconnecting(false);
      retryCountRef.current = 0;
      console.log('Transcription WebSocket connected');
    };

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        const entry: TranscriptEntry = {
          id: crypto.randomUUID(),
          speaker: data.speaker || 'Unknown',
          text: data.text,
          timestamp: Date.now(),
        };
        setTranscripts((prev) => [...prev.slice(-50), entry]); // Keep last 50
      } catch (err) {
        console.error('Failed to parse transcript:', err);
      }
    };

    ws.onclose = () => {
      setConnected(false);
      console.log('Transcription WebSocket disconnected');

      // Attempt reconnection with exponential backoff
      if (retryCountRef.current < MAX_RETRIES) {
        setReconnecting(true);
        const delay = Math.min(1000 * Math.pow(2, retryCountRef.current), 30000);
        console.log(`Reconnecting in ${delay}ms (attempt ${retryCountRef.current + 1}/${MAX_RETRIES})`);

        reconnectTimeoutRef.current = window.setTimeout(() => {
          retryCountRef.current++;
          connect();
        }, delay);
      } else {
        setReconnecting(false);
        console.log('Max reconnection attempts reached');
      }
    };

    ws.onerror = (err) => {
      console.error('Transcription WebSocket error:', err);
    };
  }, [roomName]);

  useEffect(() => {
    connect();

    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, [connect]);

  // Auto-scroll to bottom
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [transcripts]);

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="px-4 py-3 border-b border-slate-700">
        <div className="flex items-center justify-between">
          <h3 className="text-white font-medium">Live Transcription</h3>
          <span
            className={`w-2 h-2 rounded-full ${
              connected ? 'bg-green-500' : reconnecting ? 'bg-yellow-500 animate-pulse' : 'bg-red-500'
            }`}
            title={connected ? 'Connected' : reconnecting ? 'Reconnecting...' : 'Disconnected'}
          />
        </div>
        {!connected && (
          <p className="text-slate-400 text-xs mt-1">
            {reconnecting ? 'Reconnecting to transcription service...' : 'Waiting for transcription service...'}
          </p>
        )}
      </div>

      {/* Transcript list */}
      <div ref={scrollRef} className="flex-1 overflow-y-auto p-4 space-y-3">
        {transcripts.length === 0 ? (
          <p className="text-slate-500 text-sm text-center mt-8">
            Transcription will appear here when someone speaks...
          </p>
        ) : (
          transcripts.map((entry) => (
            <div key={entry.id} className="text-sm">
              <div className="flex items-baseline gap-2 mb-1">
                <span className="font-medium text-blue-400">{entry.speaker}</span>
                <span className="text-slate-500 text-xs">
                  {new Date(entry.timestamp).toLocaleTimeString()}
                </span>
              </div>
              <p className="text-slate-300">{entry.text}</p>
            </div>
          ))
        )}
      </div>

      {/* Footer */}
      <div className="px-4 py-2 border-t border-slate-700">
        <p className="text-slate-500 text-xs">
          Powered by AI transcription
        </p>
      </div>
    </div>
  );
}
