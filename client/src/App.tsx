import { MemoryRouter, Routes, Route } from 'react-router-dom';
import HomePage from './pages/Home';
import GamePage from './pages/Game';

export default function App() {
  return (
    <MemoryRouter>
      <Routes>
        <Route path="/" element={<HomePage />} />
        <Route path="/game" element={<GamePage />} />
      </Routes>
    </MemoryRouter>
  );
}
