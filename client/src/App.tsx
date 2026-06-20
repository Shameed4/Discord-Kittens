import { MemoryRouter, Routes, Route } from 'react-router-dom';
import HomePage from './pages/home/page';
import GamePage from './pages/game/page';
import { getInstanceId } from './discord/sdk';

export default function App() {
  // if in discord, skip the lobby join option
  const inDiscord = Boolean(getInstanceId());
  return (
    <MemoryRouter initialEntries={inDiscord ? ['/game'] : ['/']}>
      <Routes>
        <Route path="/" element={<HomePage />} />
        <Route path="/game" element={<GamePage />} />
      </Routes>
    </MemoryRouter>
  );
}
