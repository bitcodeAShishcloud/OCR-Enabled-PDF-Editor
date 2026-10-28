import { useEffect, useRef, useState } from 'react';
import type { Page, Word } from '../api/types';

interface Props {
  src: string;
  page: Page;
  onWordClick: (word: Word, index: number) => void;
}

export default function PagePreview({ src, page, onWordClick }: Props) {
  const imgRef = useRef<HTMLImageElement>(null);
  const [scale, setScale] = useState(1);

  useEffect(() => {
    const img = imgRef.current;
    if (!img) return;
    const update = () => {
      if (img.naturalWidth > 0) {
        setScale(img.clientWidth / img.naturalWidth);
      }
    };
    const ro = new ResizeObserver(update);
    ro.observe(img);
    img.addEventListener('load', update);
    update();
    return () => {
      ro.disconnect();
      img.removeEventListener('load', update);
    };
  }, [src]);

  return (
    <div className="relative inline-block">
      <img ref={imgRef} src={src} alt={`page ${page.page}`} className="max-w-full block" />
      <div className="absolute top-0 left-0" style={{ width: imgRef.current?.clientWidth }}>
        {page.words.map((w, i) => (
          <button
            key={i}
            type="button"
            aria-label={w.text}
            className="absolute border border-blue-500/60 hover:bg-blue-500/20 cursor-pointer"
            style={{
              left: w.bbox.x0 * scale,
              top: w.bbox.y0 * scale,
              width: (w.bbox.x1 - w.bbox.x0) * scale,
              height: (w.bbox.y1 - w.bbox.y0) * scale,
            }}
            onClick={() => onWordClick(w, i)}
          />
        ))}
      </div>
    </div>
  );
}