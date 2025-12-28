import { useMemo } from 'react';
import type { Seat } from '../types';
import { cn } from '../lib/utils';

interface SeatMapProps {
  seats: Seat[];
  selectedSeats: string[];
  onSeatSelect: (seatId: string) => void;
}

export function SeatMap({ seats, selectedSeats, onSeatSelect }: SeatMapProps) {
  // Group seats by row
  const seatsByRow = useMemo(() => {
    return seats.reduce((acc, seat) => {
      if (!acc[seat.row]) acc[seat.row] = [];
      acc[seat.row].push(seat);
      return acc;
    }, {} as Record<number, Seat[]>);
  }, [seats]);

  const rows = Object.keys(seatsByRow).map(Number).sort((a, b) => a - b);

  const getSeatClass = (seat: Seat) => {
    if (selectedSeats.includes(seat.id)) return 'seat-selected';
    if (seat.status === 'held') return 'seat-held';
    if (seat.status === 'booked') return 'seat-booked';
    return 'seat-available';
  };

  const handleSeatClick = (seat: Seat) => {
    if (seat.status === 'available' || selectedSeats.includes(seat.id)) {
      onSeatSelect(seat.id);
    }
  };

  return (
    <div className="space-y-6">
      {/* Legend */}
      <div className="flex flex-wrap items-center gap-4 sm:gap-6 text-sm p-4 bg-slate-800/50 rounded-xl">
        <div className="flex items-center gap-2">
          <div className="seat seat-available w-6 h-6 sm:w-8 sm:h-8" />
          <span className="text-slate-400">Available</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="seat seat-selected w-6 h-6 sm:w-8 sm:h-8" />
          <span className="text-slate-400">Selected</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="seat seat-held w-6 h-6 sm:w-8 sm:h-8" />
          <span className="text-slate-400">Held</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="seat seat-booked w-6 h-6 sm:w-8 sm:h-8" />
          <span className="text-slate-400">Booked</span>
        </div>
      </div>

      {/* Airplane nose indicator */}
      <div className="flex justify-center">
        <div className="w-24 sm:w-32 h-6 sm:h-8 bg-slate-800 rounded-t-full border-t border-x border-slate-600 flex items-center justify-center">
          <span className="text-xs text-slate-500">FRONT</span>
        </div>
      </div>

      {/* Seat grid */}
      <div className="space-y-2 max-h-[400px] overflow-y-auto px-2 py-4">
        {rows.slice(0, 15).map((rowNum) => {
          const rowSeats = seatsByRow[rowNum].sort((a, b) => a.column.localeCompare(b.column));
          const leftSeats = rowSeats.slice(0, 3);
          const rightSeats = rowSeats.slice(3);

          return (
            <div key={rowNum} className="flex items-center justify-center gap-2 sm:gap-4">
              {/* Left side (A, B, C) */}
              <div className="flex gap-1 sm:gap-2">
                {leftSeats.map((seat) => (
                  <button
                    key={seat.id}
                    onClick={() => handleSeatClick(seat)}
                    disabled={seat.status === 'booked' || seat.status === 'held'}
                    className={cn("seat", getSeatClass(seat))}
                    title={`Seat ${seat.row}${seat.column} - $${seat.price}`}
                    aria-label={`Seat ${seat.row}${seat.column}, ${seat.status}`}
                  >
                    {seat.column}
                  </button>
                ))}
              </div>

              {/* Aisle / Row number */}
              <div className="w-8 sm:w-10 text-center text-slate-500 text-sm font-medium">
                {rowNum}
              </div>

              {/* Right side (D, E, F) */}
              <div className="flex gap-1 sm:gap-2">
                {rightSeats.map((seat) => (
                  <button
                    key={seat.id}
                    onClick={() => handleSeatClick(seat)}
                    disabled={seat.status === 'booked' || seat.status === 'held'}
                    className={cn("seat", getSeatClass(seat))}
                    title={`Seat ${seat.row}${seat.column} - $${seat.price}`}
                    aria-label={`Seat ${seat.row}${seat.column}, ${seat.status}`}
                  >
                    {seat.column}
                  </button>
                ))}
              </div>
            </div>
          );
        })}
      </div>

      {/* Column labels */}
      <div className="flex items-center justify-center gap-2 sm:gap-4 pt-4 border-t border-slate-700">
        <div className="flex gap-1 sm:gap-2">
          {['A', 'B', 'C'].map((col) => (
            <div key={col} className="w-10 sm:w-12 text-center text-slate-500 text-xs font-medium">
              {col}
            </div>
          ))}
        </div>
        <div className="w-8 sm:w-10" />
        <div className="flex gap-1 sm:gap-2">
          {['D', 'E', 'F'].map((col) => (
            <div key={col} className="w-10 sm:w-12 text-center text-slate-500 text-xs font-medium">
              {col}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
