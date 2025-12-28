import { useState, useEffect } from 'react';

export function Timer({ seconds: initialSeconds }: { seconds: number }) {
  const [seconds, setSeconds] = useState(initialSeconds);

  useEffect(() => { setSeconds(initialSeconds); }, [initialSeconds]);
  useEffect(() => {
    const interval = setInterval(() => setSeconds((p) => Math.max(0, p - 1)), 1000);
    return () => clearInterval(interval);
  }, []);

  const minutes = Math.floor(seconds / 60);
  const secs = seconds % 60;
  const isLow = seconds < 60;

  return (
    <div className={`inline-flex items-center gap-2 px-4 py-2 rounded-lg ${isLow ? 'bg-red-500/20 text-red-400' : 'bg-amber-500/20 text-amber-400'}`}>
      <span className="font-mono text-lg font-bold">{minutes}:{secs.toString().padStart(2, '0')}</span>
      <span className="text-sm">seat hold remaining</span>
    </div>
  );
}

