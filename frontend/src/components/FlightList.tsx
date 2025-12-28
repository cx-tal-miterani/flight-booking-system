import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { Plane, Clock, Users, ArrowRight, Loader2 } from 'lucide-react';
import { api } from '../api';
import type { Flight } from '../types';
import { Card, CardContent } from './ui/card';
import { Button } from './ui/button';
import { Badge } from './ui/badge';
import { Alert, AlertDescription } from './ui/alert';
import { formatTime, formatDate, formatCurrency } from '../lib/utils';

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

  if (loading) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[400px] gap-4">
        <Loader2 className="w-12 h-12 text-cyan-500 animate-spin" />
        <p className="text-slate-400">Loading available flights...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="max-w-lg mx-auto">
        <Alert variant="danger">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      </div>
    );
  }

  return (
    <div className="animate-fade-in">
      <div className="mb-8">
        <h1 className="text-3xl sm:text-4xl font-bold text-white mb-2">Available Flights</h1>
        <p className="text-slate-400">Select a flight to begin your booking</p>
      </div>

      <div className="grid gap-4">
        {flights.map((flight, index) => (
          <Card 
            key={flight.id}
            className="group hover:border-cyan-500/50 transition-all duration-300 hover:shadow-lg hover:shadow-cyan-500/10"
            style={{ animationDelay: `${index * 100}ms` }}
          >
            <CardContent className="p-4 sm:p-6">
              <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-4 lg:gap-6">
                {/* Flight Info */}
                <div className="flex-1">
                  <div className="flex flex-wrap items-center gap-2 mb-4">
                    <Badge variant="default">{flight.flightNumber}</Badge>
                    <Badge variant="secondary">
                      <Clock className="w-3 h-3 mr-1" />
                      {formatDate(flight.departureTime)}
                    </Badge>
                    <Badge variant="secondary">
                      <Users className="w-3 h-3 mr-1" />
                      {flight.availableSeats} seats left
                    </Badge>
                  </div>

                  {/* Route */}
                  <div className="flex items-center gap-3 sm:gap-6">
                    <div className="text-center sm:text-left">
                      <p className="text-xl sm:text-2xl font-bold text-white">
                        {formatTime(flight.departureTime)}
                      </p>
                      <p className="text-sm text-slate-400 truncate max-w-[120px] sm:max-w-none">
                        {flight.origin}
                      </p>
                    </div>

                    <div className="flex-1 flex items-center gap-2 px-2 sm:px-4">
                      <div className="h-[2px] flex-1 bg-gradient-to-r from-cyan-500 to-transparent" />
                      <Plane className="w-5 h-5 text-cyan-500 transform rotate-90" />
                      <div className="h-[2px] flex-1 bg-gradient-to-l from-emerald-500 to-transparent" />
                    </div>

                    <div className="text-center sm:text-right">
                      <p className="text-xl sm:text-2xl font-bold text-white">
                        {formatTime(flight.arrivalTime)}
                      </p>
                      <p className="text-sm text-slate-400 truncate max-w-[120px] sm:max-w-none">
                        {flight.destination}
                      </p>
                    </div>
                  </div>
                </div>

                {/* Price & CTA */}
                <div className="flex flex-row lg:flex-col items-center lg:items-end justify-between lg:justify-start gap-4 pt-4 lg:pt-0 border-t lg:border-t-0 lg:border-l border-slate-700/50 lg:pl-6">
                  <div className="text-left lg:text-right">
                    <p className="text-2xl sm:text-3xl font-bold text-emerald-500">
                      {formatCurrency(flight.pricePerSeat)}
                    </p>
                    <p className="text-sm text-slate-500">per seat</p>
                  </div>
                  
                  <Button
                    onClick={() => navigate(`/book/${flight.id}`)}
                    className="group-hover:shadow-lg group-hover:shadow-cyan-500/25"
                  >
                    Book Now
                    <ArrowRight className="w-4 h-4 ml-2 group-hover:translate-x-1 transition-transform" />
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
