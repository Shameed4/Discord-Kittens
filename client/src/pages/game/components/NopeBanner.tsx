import { formatNopeSeconds } from '../use-nope-countdown';

interface NopeBannerProps {
  isNoped: boolean;
  remaining: number;
  fraction: number;
}

// Center-of-table banner shown to everyone during the nope window: the current
// verdict plus a countdown bar. Amber when the action will currently succeed,
// red when it is currently noped (and will be denied).
export default function NopeBanner({
  isNoped,
  remaining,
  fraction,
}: NopeBannerProps) {
  const accent = isNoped ? '#ef4444' : '#f59e0b';
  const verdict = isNoped ? 'NOPED' : 'Will succeed';

  return (
    <div className="flex flex-col items-center gap-1">
      <div
        className="flex items-center gap-1.5 text-[10px] font-black tracking-widest uppercase"
        style={{ color: accent }}
      >
        <span>{isNoped ? '🚫' : '✅'}</span>
        <span>{verdict}</span>
        <span className="tabular-nums opacity-90">
          · {formatNopeSeconds(remaining)}s
        </span>
      </div>
      <div className="h-1 w-28 overflow-hidden rounded-full bg-black/40">
        <div
          className="h-full rounded-full transition-[width] duration-100 ease-linear"
          style={{ width: `${fraction * 100}%`, background: accent }}
        />
      </div>
    </div>
  );
}
