import { useState } from 'react';
import { useNavigate } from 'react-router-dom';

export default function HomePage() {
  const navigate = useNavigate();
  const [createName, setCreateName] = useState('');
  const [joinName, setJoinName] = useState('');
  const [status, setStatus] = useState({ message: '', isError: false });

  const showStatus = (message: string, isError = true) => {
    setStatus({ message, isError });
    setTimeout(() => setStatus({ message: '', isError: false }), 4000);
  };

  const handleCreate = async () => {
    if (!createName.trim()) return;

    try {
      const response = await fetch('/api/lobby', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: createName.trim() }),
      });

      if (response.status === 201) {
        // Automatically join the room we just created
        navigate('/game', { state: { lobbyName: createName.trim() } });
      } else if (response.status === 409) {
        showStatus('Lobby already exists. Try joining it.');
      } else {
        showStatus('Failed to create lobby.');
      }
    } catch {
      showStatus('Network error while creating lobby.');
    }
  };

  const handleJoin = () => {
    if (!joinName.trim()) return;
    // Route to the game screen and pass the lobby name in memory
    navigate('/game', { state: { lobbyName: joinName.trim() } });
  };

  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-gray-100 font-sans text-gray-800">
      <div className="mb-8 flex w-87.5 flex-col gap-4 rounded-lg bg-white p-6 shadow-sm">
        {/* Create Input Group */}
        <div className="flex gap-2">
          <input
            type="text"
            className="grow rounded-md border border-gray-300 p-2 text-base"
            placeholder="New lobby name..."
            value={createName}
            onChange={(e) => setCreateName(e.target.value)}
          />
          <button
            onClick={handleCreate}
            className="rounded-md bg-blue-600 px-4 py-2 text-white transition-colors hover:bg-blue-700"
          >
            Create
          </button>
        </div>

        {/* Join Input Group */}
        <div className="flex gap-2">
          <input
            type="text"
            className="grow rounded-md border border-gray-300 p-2 text-base"
            placeholder="Existing lobby name..."
            value={joinName}
            onChange={(e) => setJoinName(e.target.value)}
          />
          <button
            onClick={handleJoin}
            className="rounded-md bg-gray-500 px-4 py-2 text-white transition-colors hover:bg-gray-600"
          >
            Join
          </button>
        </div>

        {/* Status Message */}
        <div
          className={`mt-1 min-h-5 text-sm ${status.isError ? 'text-red-500' : 'text-green-500'}`}
        >
          {status.message}
        </div>
      </div>
    </div>
  );
}
