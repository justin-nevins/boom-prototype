import { useState } from 'react';

interface NotesModalProps {
  isOpen: boolean;
  onClose: () => void;
  markdown: string;
  isLoading?: boolean;
}

export default function NotesModal({ isOpen, onClose, markdown, isLoading }: NotesModalProps) {
  const [copied, setCopied] = useState(false);

  if (!isOpen) return null;

  const copyToClipboard = () => {
    navigator.clipboard.writeText(markdown);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const downloadMarkdown = () => {
    const blob = new Blob([markdown], { type: 'text/markdown' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `meeting-notes-${new Date().toISOString().split('T')[0]}.md`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-slate-800 rounded-lg w-full max-w-3xl max-h-[80vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-slate-700">
          <h2 className="text-lg font-semibold text-white">Meeting Notes</h2>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-white transition-colors"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-4">
          {isLoading ? (
            <div className="flex items-center justify-center py-12">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
              <span className="ml-3 text-slate-300">Generating notes with Claude...</span>
            </div>
          ) : (
            <div className="prose prose-invert prose-sm max-w-none">
              <MarkdownRenderer content={markdown} />
            </div>
          )}
        </div>

        {/* Footer */}
        {!isLoading && markdown && (
          <div className="flex items-center justify-end gap-2 px-4 py-3 border-t border-slate-700">
            <button
              onClick={copyToClipboard}
              className="px-3 py-1.5 bg-slate-700 hover:bg-slate-600 text-slate-300 text-sm rounded transition-colors"
            >
              {copied ? 'Copied!' : 'Copy to clipboard'}
            </button>
            <button
              onClick={downloadMarkdown}
              className="px-3 py-1.5 bg-blue-600 hover:bg-blue-500 text-white text-sm rounded transition-colors"
            >
              Download .md
            </button>
          </div>
        )}
      </div>
    </div>
  );
}

// Simple markdown renderer without external dependencies
function MarkdownRenderer({ content }: { content: string }) {
  // Convert markdown to HTML (basic implementation)
  const html = content
    // Headers
    .replace(/^### (.*$)/gim, '<h3 class="text-lg font-semibold text-white mt-4 mb-2">$1</h3>')
    .replace(/^## (.*$)/gim, '<h2 class="text-xl font-semibold text-white mt-6 mb-2">$1</h2>')
    .replace(/^# (.*$)/gim, '<h1 class="text-2xl font-bold text-white mt-6 mb-3">$1</h1>')
    // Bold
    .replace(/\*\*(.*?)\*\*/g, '<strong class="text-white">$1</strong>')
    // Italic
    .replace(/\*(.*?)\*/g, '<em>$1</em>')
    // Checkboxes
    .replace(/- \[ \] (.*$)/gim, '<div class="flex items-start gap-2 my-1"><input type="checkbox" class="mt-1" disabled /><span class="text-slate-300">$1</span></div>')
    .replace(/- \[x\] (.*$)/gim, '<div class="flex items-start gap-2 my-1"><input type="checkbox" class="mt-1" checked disabled /><span class="text-slate-300 line-through">$1</span></div>')
    // Bullet points
    .replace(/^- (.*$)/gim, '<li class="text-slate-300 ml-4">$1</li>')
    // Numbered lists
    .replace(/^\d+\. (.*$)/gim, '<li class="text-slate-300 ml-4 list-decimal">$1</li>')
    // Line breaks
    .replace(/\n\n/g, '</p><p class="text-slate-300 my-2">')
    .replace(/\n/g, '<br />');

  return (
    <div
      className="text-slate-300"
      dangerouslySetInnerHTML={{ __html: `<p class="text-slate-300">${html}</p>` }}
    />
  );
}
