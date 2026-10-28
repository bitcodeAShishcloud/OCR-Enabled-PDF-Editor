import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { uploadPdf } from '../api/client';
import type { UploadResponse } from '../api/types';
import UploadDropzone from '../components/UploadDropzone';

export default function UploadPage() {
  const navigate = useNavigate();
  const [file, setFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [doc, setDoc] = useState<UploadResponse | null>(null);

  const handleFile = async (f: File) => {
    setFile(f);
    setError(null);
    setUploading(true);
    try {
      const res = await uploadPdf(f);
      setDoc(res);
      navigate(`/documents/${res.docId}`, { state: { doc: res } });
    } catch (e) {
      setError(e instanceof Error ? e.message : 'upload failed');
    } finally {
      setUploading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center p-6">
      <div className="w-full max-w-xl bg-white shadow rounded-lg p-8 space-y-4">
        <h1 className="text-2xl font-bold">OCR PDF Editor</h1>
        <p className="text-gray-500">Upload a scanned PDF to begin.</p>
        <UploadDropzone onFile={handleFile} disabled={uploading} />
        {uploading && <p className="text-blue-600">Uploading and OCRing...</p>}
        {error && <p className="text-red-600">{error}</p>}
        {doc && <p className="text-green-600">Uploaded {doc.pages.length} page(s).</p>}
        {file && <p className="text-sm text-gray-400">{file.name}</p>}
      </div>
    </div>
  );
}