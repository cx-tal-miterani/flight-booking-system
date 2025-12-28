import { useState, useEffect } from 'react';
import { Clock, AlertTriangle } from 'lucide-react';
import { Progress } from './ui/progress';
import { cn } from '../lib/utils';

interface TimerProps {
  seconds: number;
  totalSeconds?: number;
  onRefresh?: () => void;
}

export function Timer({ seconds: initialSeconds, totalSeconds = 900, onRefresh }: TimerProps) {
  const [seconds, setSeconds] = useState(initialSeconds);

  useEffect(() => {
    setSeconds(initialSeconds);
  }, [initialSeconds]);

  useEffect(() => {
    const interval = setInterval(() => {
      setSeconds((prev) => Math.max(0, prev - 1));
    }, 1000);

    return () => clearInterval(interval);
  }, []);

  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  const progress = (seconds / totalSeconds) * 100;
  
  const isLow = seconds < 120; // Less than 2 minutes
  const isCritical = seconds < 60; // Less than 1 minute

  return (
    <div
      className={cn(
        "rounded-xl p-4 border transition-all duration-300",
        isCritical 
          ? "bg-red-500/10 border-red-500/30" 
          : isLow 
          ? "bg-amber-500/10 border-amber-500/30" 
          : "bg-cyan-500/10 border-cyan-500/30"
      )}
    >
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          {isCritical ? (
            <AlertTriangle className={cn("w-5 h-5 text-red-500", isCritical && "animate-pulse")} />
          ) : (
            <Clock className={cn("w-5 h-5", isLow ? "text-amber-500" : "text-cyan-500")} />
          )}
          <span className={cn(
            "text-sm font-medium",
            isCritical ? "text-red-400" : isLow ? "text-amber-400" : "text-cyan-400"
          )}>
            Seat Hold Timer
          </span>
        </div>
        
        {onRefresh && (
          <button
            onClick={onRefresh}
            className="text-xs text-slate-400 hover:text-white transition-colors underline"
          >
            Refresh Timer
          </button>
        )}
      </div>

      <div className="flex items-center gap-4">
        <div className={cn(
          "text-3xl font-mono font-bold",
          isCritical ? "text-red-500" : isLow ? "text-amber-500" : "text-white"
        )}>
          {minutes}:{remainingSeconds.toString().padStart(2, '0')}
        </div>
        
        <div className="flex-1">
          <Progress 
            value={progress} 
            className="h-2"
            indicatorClassName={cn(
              isCritical ? "bg-red-500" : isLow ? "bg-amber-500" : "bg-cyan-500"
            )}
          />
          <p className="text-xs text-slate-500 mt-1">
            {isCritical 
              ? "Hurry! Seats will be released soon" 
              : isLow 
              ? "Less than 2 minutes remaining" 
              : "Time remaining to complete payment"}
          </p>
        </div>
      </div>
    </div>
  );
}
