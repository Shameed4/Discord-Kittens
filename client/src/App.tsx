import { MemoryRouter, Routes, Route } from 'react-router-dom';
import HomePage from './pages/home/page';
import GamePage from './pages/game/page';

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
