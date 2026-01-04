import { useState } from 'react';
import NotesModal from './NotesModal';

const AI_SERVICE_URL = import.meta.env.VITE_AI_SERVICE_URL || 'http://localhost:8081';

interface MeetingNotesButtonProps {
  roomName: string;
}

export default function MeetingNotesButton({ roomName }: MeetingNotesButtonProps) {
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [notes, setNotes] = useState<string>('');
  const [error, setError] = useState<string>('');

  const generateNotes = async () => {
    setIsLoading(true);
    setError('');
    setIsModalOpen(true);

    try {
      const res = await fetch(`${AI_SERVICE_URL}/generate-notes`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ room_name: roomName }),
      });

      const data = await res.json();

      if (res.ok && data.markdown) {
        setNotes(data.markdown);
      } else {
        setError(data.error || 'Failed to generate notes');
        setNotes('');
      }
    } catch (err) {
      console.error('Error generating notes:', err);
      setError('Failed to connect to AI service');
      setNotes('');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <>
      <button
        onClick={generateNotes}
        disabled={isLoading}
        className="px-3 py-1 bg-emerald-600 hover:bg-emerald-500 disabled:bg-slate-600 text-white text-sm rounded transition-colors flex items-center gap-1.5"
      >
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
        </svg>
        {isLoading ? 'Generating...' : 'Generate Notes'}
      </button>

      <NotesModal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        markdown={error || notes}
        isLoading={isLoading}
      />
    </>
  );
}
