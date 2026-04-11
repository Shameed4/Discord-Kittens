interface LastActionBannerProps {
  lastAction?: string;
}

export default function LastActionBanner({ lastAction }: LastActionBannerProps) {
  if (!lastAction) return <div className="h-8" />;
  return (
    <div className="bg-amber-50 border border-amber-200 text-amber-900 rounded-full px-5 py-1.5 text-sm font-medium text-center max-w-md">
      {lastAction}
    </div>
  );
}
