import { useEffect, useRef, useState } from 'react';

interface GameLogProps {
  log: string[];
}

export default function GameLog({ log }: GameLogProps) {
  const [collapsed, setCollapsed] = useState(false);
  const bottomRef = useRef<HTMLDivElement | null>(null);

  // Auto-scroll to newest entry as the log grows (only while expanded).
  useEffect(() => {
    if (!collapsed) bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [log.length, collapsed]);

  if (collapsed) {
    return (
      <button
        onClick={() => setCollapsed(false)}
        aria-label="Show game log"
        className="flex h-full w-9 shrink-0 flex-col items-center gap-2 border-l border-purple-800 bg-[#0d0720]/95 pt-2 text-purple-400 hover:text-purple-200"
      >
        <span className="text-lg">📜</span>
        <span
          className="text-[9px] font-bold tracking-widest uppercase"
          style={{ writingMode: 'vertical-rl' }}
        >
          Log
        </span>
      </button>
    );
  }

  return (
    <div className="flex h-full w-[clamp(150px,28vw,240px)] shrink-0 flex-col border-l border-purple-800 bg-[#0d0720]/95">
      <div className="flex items-center justify-between border-b border-purple-900 px-3 py-2">
        <span className="text-[10px] font-bold tracking-widest text-purple-400 uppercase">
          Game Log
        </span>
        <button
          onClick={() => setCollapsed(true)}
          aria-label="Collapse game log"
          className="text-purple-400 hover:text-purple-200"
        >
          ✕
        </button>
      </div>
      <div className="flex-1 overflow-y-auto px-3 py-2">
        {log.length === 0 ? (
          <p className="text-[10px] text-purple-800">No events yet.</p>
        ) : (
          <ul className="flex flex-col gap-1.5">
            {log.map((entry, i) => (
              <li key={i} className="text-[10px] leading-snug text-purple-200">
                {entry}
              </li>
            ))}
          </ul>
        )}
        <div ref={bottomRef} />
      </div>
    </div>
  );
}
