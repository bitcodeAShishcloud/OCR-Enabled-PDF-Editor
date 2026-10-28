import { useEffect, useState } from 'react';
import { Link, useParams, useLocation } from 'react-router-dom';
import { getPageImage, applyEdit, resultUrl, deleteDocument } from '../api/client';
import type { UploadResponse, Page, Word, EditRequest } from '../api/types';
import PagePreview from '../components/PagePreview';
import EditDialog from '../components/EditDialog';

interface LocationState {
  doc?: UploadResponse;
}

export default function DocumentPage() {
  const { id } = useParams<{ id: string }>();
  const location = useLocation();
  const state = (location.state || {}) as LocationState;
  const initialDoc = state.doc || null;

  const [doc, setDoc] = useState<UploadResponse | null>(initialDoc);
  const [currentPage, setCurrentPage] = useState(0);
  const [imageSrc, setImageSrc] = useState<string | null>(null);
  const [editing, setEditing] = useState<{ word: Word; index: number } | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    return () => {
      if (imageSrc) URL.revokeObjectURL(imageSrc);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (!doc) return;
    const page = doc.pages[currentPage];
    if (!page) return;
    let cancelled = false;
    setLoading(true);
    getPageImage(doc.docId, page.page)
      .then((url) => {
        if (cancelled) {
          URL.revokeObjectURL(url);
          return;
        }
        setImageSrc((prev) => {
          if (prev) URL.revokeObjectURL(prev);
          return url;
        });
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof Error ? e.message : 'failed to load page');
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [doc, currentPage]);

  useEffect(() => {
    return () => {
      if (doc) deleteDocument(doc.docId).catch(() => {});
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  if (!doc || !id) {
    return (
      <div className="p-8">
        <h1 className="text-2xl font-bold">Document not found</h1>
        <p className="text-gray-500">Go back and upload a PDF.</p>
        <Link to="/" className="text-blue-600">Back to upload</Link>
      </div>
    );
  }

  const page: Page = doc.pages[currentPage];

  const handleWordClick = (word: Word, index: number) => {
    setEditing({ word, index });
  };

  const handleSubmitEdit = async (newText: string, fontPath?: string, fontSize?: number, color?: string) => {
    const req: EditRequest = {
      page: page.page,
      wordIds: editing ? [editing.index] : [],
      newText,
      fontPath,
      fontSize,
      color,
    };
    try {
      const blob = await applyEdit(doc.docId, req);
      const url = URL.createObjectURL(blob);
      setImageSrc((prev) => {
        if (prev) URL.revokeObjectURL(prev);
        return url;
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : 'edit failed');
    }
  };

  return (
    <div className="min-h-screen p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Document {doc.docId}</h1>
        <a
          href={resultUrl(doc.docId)}
          download="result.pdf"
          className="px-4 py-2 rounded bg-green-600 text-white"
        >
          Download result.pdf
        </a>
      </div>
      {error && <p className="text-red-600">{error}</p>}
      {loading && <p className="text-blue-600">Loading...</p>}
      <div className="flex items-center gap-2">
        <button
          onClick={() => setCurrentPage((p) => Math.max(0, p - 1))}
          disabled={currentPage === 0}
          className="px-3 py-1 rounded border disabled:opacity-50"
        >
          Prev
        </button>
        <span>
          Page {currentPage + 1} / {doc.pages.length}
        </span>
        <button
          onClick={() => setCurrentPage((p) => Math.min(doc.pages.length - 1, p + 1))}
          disabled={currentPage >= doc.pages.length - 1}
          className="px-3 py-1 rounded border disabled:opacity-50"
        >
          Next
        </button>
      </div>
      {imageSrc && page && (
        <PagePreview src={imageSrc} page={page} onWordClick={handleWordClick} />
      )}
      <EditDialog
        open={editing !== null}
        oldText={editing?.word.text || ''}
        onClose={() => setEditing(null)}
        onSubmit={handleSubmitEdit}
      />
    </div>
  );
}