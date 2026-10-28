import type { UploadResponse, EditRequest, Page, Word, BBox } from './types';

const BASE = '/api';

export async function uploadPdf(file: File): Promise<UploadResponse> {
  const form = new FormData();
  form.append('file', file);
  const res = await fetch(`${BASE}/documents`, { method: 'POST', body: form });
  if (!res.ok) throw new Error(`upload failed: ${res.status}`);
  return res.json();
}

export async function getPageImage(docId: string, page: number): Promise<string> {
  const res = await fetch(`${BASE}/documents/${docId}/pages/${page}`);
  if (!res.ok) throw new Error(`page fetch failed: ${res.status}`);
  const blob = await res.blob();
  return URL.createObjectURL(blob);
}

export async function applyEdit(docId: string, req: EditRequest): Promise<Blob> {
  const res = await fetch(`${BASE}/documents/${docId}/edits`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  });
  if (!res.ok) throw new Error(`edit failed: ${res.status}`);
  return res.blob();
}

export function resultUrl(docId: string): string {
  return `${BASE}/documents/${docId}/result.pdf`;
}

export async function deleteDocument(docId: string): Promise<void> {
  await fetch(`${BASE}/documents/${docId}`, { method: 'DELETE' });
}

export type { UploadResponse, Page, Word, BBox, EditRequest };