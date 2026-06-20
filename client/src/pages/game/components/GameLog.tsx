import { useEffect, useRef, useState } from 'react';

interface GameLogProps {
  log: string[];
}

export default function GameLog({ log }: GameLogProps) {
  const [open, setOpen] = useState(false);
  // Number of entries already seen by the player. Updated only in the toggle
  // handler (opening or closing marks everything currently logged as seen), so
  // there is no setState-in-effect. While the panel is open the badge is hidden
  // regardless; new entries only accrue as "unread" once the panel is closed.
  const [seenCount, setSeenCount] = useState(0);
  const bottomRef = useRef<HTMLDivElement | null>(null);

  // Auto-scroll to newest entry while the panel is open.
  useEffect(() => {
    if (open) bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [open, log.length]);

  const unread = open ? 0 : Math.max(0, log.length - seenCount);

  const toggle = () => {
    setSeenCount(log.length);
    setOpen((o) => !o);
  };

  return (
    <>
      {/* Floating toggle button */}
      <button
        onClick={toggle}
        className="fixed bottom-3 left-3 z-40 flex h-10 w-10 items-center justify-center rounded-full border border-purple-700 bg-purple-950/90 text-lg shadow-lg"
        aria-label="Toggle game log"
      >
        📜
        {unread > 0 && !open && (
          <span className="absolute -top-1 -right-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-red-600 px-1 text-[9px] font-bold text-white">
            {unread > 99 ? '99+' : unread}
          </span>
        )}
      </button>

      {/* Slide-in panel */}
      <div
        className={`fixed top-0 right-0 z-50 flex h-full w-64 max-w-[80vw] flex-col border-l border-purple-800 bg-[#0d0720]/95 shadow-2xl transition-transform duration-200 ${
          open ? 'translate-x-0' : 'translate-x-full'
        }`}
      >
        <div className="flex items-center justify-between border-b border-purple-900 px-3 py-2">
          <span className="text-[10px] font-bold tracking-widest text-purple-400 uppercase">
            Game Log
          </span>
          <button
            onClick={() => {
              setSeenCount(log.length);
              setOpen(false);
            }}
            className="text-purple-400 hover:text-purple-200"
            aria-label="Close game log"
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
                <li
                  key={i}
                  className="text-[10px] leading-snug text-purple-200"
                >
                  {entry}
                </li>
              ))}
            </ul>
          )}
          <div ref={bottomRef} />
        </div>
      </div>
    </>
  );
}
