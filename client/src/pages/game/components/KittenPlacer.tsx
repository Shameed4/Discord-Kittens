import { useState } from 'react';

interface KittenPlacerProps {
  deckSize: number;
  onPlace: (index: number) => void;
}

export default function KittenPlacer({ deckSize, onPlace }: KittenPlacerProps) {
  const [position, setPosition] = useState(0);

  const clamp = (v: number) => Math.max(0, Math.min(deckSize, v));

  return (
    <div className="flex flex-col items-center gap-3 bg-gray-800 border border-red-700 rounded-xl p-5">
      <span className="text-red-300 font-bold text-sm uppercase tracking-widest">Place the Exploding Kitten</span>
      <span className="text-gray-400 text-xs">0 = top of deck &nbsp;·&nbsp; {deckSize} = bottom</span>
      <div className="flex items-center gap-3">
        <button
          onClick={() => setPosition(clamp(position - 1))}
          className="w-8 h-8 bg-gray-700 hover:bg-gray-600 rounded-full font-bold text-white"
        >
          −
        </button>
        <span className="text-white text-xl font-bold w-10 text-center">{position}</span>
        <button
          onClick={() => setPosition(clamp(position + 1))}
          className="w-8 h-8 bg-gray-700 hover:bg-gray-600 rounded-full font-bold text-white"
        >
          +
        </button>
      </div>
      <button
        onClick={() => onPlace(position)}
        className="px-5 py-2 bg-red-700 hover:bg-red-600 text-white font-bold rounded-lg transition-colors"
      >
        Place Kitten
      </button>
    </div>
  );
}
