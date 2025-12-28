import type { Seat } from '../types';

interface SeatMapProps {
  seats: Seat[];
  selectedSeats: string[];
  onSeatSelect: (seatId: string) => void;
}

export function SeatMap({ seats, selectedSeats, onSeatSelect }: SeatMapProps) {
  const seatsByRow = seats.reduce((acc, seat) => {
    if (!acc[seat.row]) acc[seat.row] = [];
    acc[seat.row].push(seat);
    return acc;
  }, {} as Record<number, Seat[]>);

  const rows = Object.keys(seatsByRow).map(Number).sort((a, b) => a - b);

  const getSeatClass = (seat: Seat) => {
    if (selectedSeats.includes(seat.id)) return 'seat-selected';
    if (seat.status === 'held') return 'seat-held';
    if (seat.status === 'booked') return 'seat-booked';
    return 'seat-available';
  };

  return (
    <div className="space-y-4">
      <div className="flex gap-6 text-sm mb-4">
        <div className="flex items-center gap-2"><div className="w-6 h-6 rounded seat-available" /><span className="text-slate-400">Available</span></div>
        <div className="flex items-center gap-2"><div className="w-6 h-6 rounded seat-selected" /><span className="text-slate-400">Selected</span></div>
        <div className="flex items-center gap-2"><div className="w-6 h-6 rounded seat-booked" /><span className="text-slate-400">Booked</span></div>
      </div>

      <div className="space-y-2 max-h-80 overflow-y-auto">
        {rows.slice(0, 10).map((rowNum) => {
          const rowSeats = seatsByRow[rowNum].sort((a, b) => a.column.localeCompare(b.column));
          const left = rowSeats.slice(0, 3);
          const right = rowSeats.slice(3);

          return (
            <div key={rowNum} className="flex items-center justify-center gap-4">
              <div className="flex gap-1">
                {left.map((seat) => (
                  <button key={seat.id} onClick={() => seat.status === 'available' && onSeatSelect(seat.id)}
                    className={`w-10 h-10 rounded text-xs font-medium ${getSeatClass(seat)}`}>
                    {seat.column}
                  </button>
                ))}
              </div>
              <div className="w-8 text-center text-slate-500">{rowNum}</div>
              <div className="flex gap-1">
                {right.map((seat) => (
                  <button key={seat.id} onClick={() => seat.status === 'available' && onSeatSelect(seat.id)}
                    className={`w-10 h-10 rounded text-xs font-medium ${getSeatClass(seat)}`}>
                    {seat.column}
                  </button>
                ))}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

