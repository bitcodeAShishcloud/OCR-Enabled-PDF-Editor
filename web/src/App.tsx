import { Routes, Route } from 'react-router-dom';
import UploadPage from './pages/UploadPage';
import DocumentPage from './pages/DocumentPage';

export default function App() {
  return (
    <div className="min-h-screen bg-gray-50 text-gray-900">
      <Routes>
        <Route path="/" element={<UploadPage />} />
        <Route path="/documents/:id" element={<DocumentPage />} />
      </Routes>
    </div>
  );
}