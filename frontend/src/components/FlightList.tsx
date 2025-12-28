import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '../api';
import type { Flight } from '../types';

export function FlightList() {
  const [flights, setFlights] = useState<Flight[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const navigate = useNavigate();

  useEffect(() => {
    api.getFlights()
      .then(setFlights)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, []);

  const formatTime = (dateString: string) => {
    return new Date(dateString).toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' });
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-cyan-500"></div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="text-center py-12">
        <p className="text-red-400">{error}</p>
      </div>
    );
  }

  return (
    <div>
      <h1 className="text-4xl font-bold text-white mb-8">Available Flights</h1>
      <div className="grid gap-4">
        {flights.map((flight) => (
          <div
            key={flight.id}
            className="bg-slate-800/50 border border-slate-700/50 rounded-2xl p-6 hover:border-cyan-500/50 transition-all"
          >
            <div className="flex items-center justify-between">
              <div>
                <span className="px-3 py-1 rounded-full bg-cyan-500/10 text-cyan-500 text-sm font-medium">
                  {flight.flightNumber}
                </span>
                <div className="flex items-center gap-4 mt-4">
                  <div>
                    <p className="text-2xl font-semibold text-white">{formatTime(flight.departureTime)}</p>
                    <p className="text-slate-400">{flight.origin}</p>
                  </div>
                  <div className="text-cyan-500">→</div>
                  <div>
                    <p className="text-2xl font-semibold text-white">{formatTime(flight.arrivalTime)}</p>
                    <p className="text-slate-400">{flight.destination}</p>
                  </div>
                </div>
              </div>
              <div className="text-right">
                <p className="text-3xl font-bold text-emerald-500">${flight.pricePerSeat}</p>
                <p className="text-slate-500">per seat</p>
                <button
                  onClick={() => navigate(`/book/${flight.id}`)}
                  className="mt-4 px-6 py-3 bg-cyan-500 text-slate-900 font-semibold rounded-xl hover:bg-cyan-400 transition-colors"
                >
                  Book Now
                </button>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

