import { useEffect, useState } from 'react';

interface Props {
  open: boolean;
  oldText: string;
  onClose: () => void;
  onSubmit: (newText: string, fontPath?: string, fontSize?: number, color?: string) => Promise<void> | void;
}

export default function EditDialog({ open, oldText, onClose, onSubmit }: Props) {
  const [newText, setNewText] = useState(oldText);
  const [fontPath, setFontPath] = useState('');
  const [fontSize, setFontSize] = useState('');
  const [color, setColor] = useState('');
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (open) {
      setNewText(oldText);
      setFontPath('');
      setFontSize('');
      setColor('');
    }
  }, [open, oldText]);

  if (!open) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newText.trim()) return;
    setSubmitting(true);
    try {
      await onSubmit(
        newText,
        fontPath || undefined,
        fontSize ? Number(fontSize) : undefined,
        color || undefined,
      );
      onClose();
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-lg p-6 w-full max-w-md">
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-lg font-bold">Edit word</h2>
          <button onClick={onClose} className="text-gray-500 hover:text-gray-800">✕</button>
        </div>
        <p className="text-sm text-gray-500 mb-3">Old text: {oldText}</p>
        <form onSubmit={handleSubmit} className="space-y-3">
          <div>
            <label className="block text-sm font-medium">New text</label>
            <input
              className="w-full border rounded px-3 py-2"
              value={newText}
              onChange={(e) => setNewText(e.target.value)}
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium">Font path (optional)</label>
            <input
              className="w-full border rounded px-3 py-2"
              value={fontPath}
              onChange={(e) => setFontPath(e.target.value)}
              placeholder="C:\\Windows\\Fonts\\arial.ttf"
            />
          </div>
          <div>
            <label className="block text-sm font-medium">Font size (optional)</label>
            <input
              type="number"
              className="w-full border rounded px-3 py-2"
              value={fontSize}
              onChange={(e) => setFontSize(e.target.value)}
            />
          </div>
          <div>
            <label className="block text-sm font-medium">Color (optional, #RRGGBB)</label>
            <input
              className="w-full border rounded px-3 py-2"
              value={color}
              onChange={(e) => setColor(e.target.value)}
              placeholder="#000000"
            />
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <button type="button" onClick={onClose} className="px-4 py-2 rounded border">Cancel</button>
            <button
              type="submit"
              disabled={!newText.trim() || submitting}
              className="px-4 py-2 rounded bg-blue-600 text-white disabled:opacity-50"
            >
              {submitting ? 'Applying...' : 'Apply'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}