export interface BBox {
  x0: number;
  y0: number;
  x1: number;
  y1: number;
}

export interface Word {
  text: string;
  conf: number;
  bbox: BBox;
}

export interface Page {
  page: number;
  words: Word[];
}

export interface UploadResponse {
  docId: string;
  pages: Page[];
}

export interface EditRequest {
  page: number;
  oldText?: string;
  wordIds?: number[];
  newText: string;
  fontPath?: string;
  fontSize?: number;
  color?: string;
}