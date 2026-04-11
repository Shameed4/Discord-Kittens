// client/src/pages/game/components/LastActionBanner.tsx
interface LastActionBannerProps {
  lastAction?: string;
}

export default function LastActionBanner({ lastAction }: LastActionBannerProps) {
  if (!lastAction) return null;
  return (
    <div className="rounded-full border border-purple-800/60 bg-purple-950/60 px-4 py-1 text-[10px] font-semibold text-purple-200 text-center max-w-[200px] leading-snug">
      {lastAction}
    </div>
  );
}
