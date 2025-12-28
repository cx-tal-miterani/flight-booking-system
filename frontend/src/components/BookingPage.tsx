import { useState, useEffect, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { api } from '../api';
import type { Flight, Seat, Order } from '../types';
import { SeatMap } from './SeatMap';
import { Timer } from './Timer';
import { PaymentForm } from './PaymentForm';

type BookingStep = 'customer' | 'seats' | 'payment' | 'confirmed' | 'failed';

export function BookingPage() {
  const { flightId } = useParams<{ flightId: string }>();
  const navigate = useNavigate();
  
  const [flight, setFlight] = useState<Flight | null>(null);
  const [seats, setSeats] = useState<Seat[]>([]);
  const [selectedSeats, setSelectedSeats] = useState<string[]>([]);
  const [order, setOrder] = useState<Order | null>(null);
  const [remainingSeconds, setRemainingSeconds] = useState(0);
  const [step, setStep] = useState<BookingStep>('customer');
  const [loading, setLoading] = useState(true);
  const [customerInfo, setCustomerInfo] = useState({ name: '', email: '' });

  useEffect(() => {
    if (!flightId) return;
    Promise.all([api.getFlight(flightId), api.getFlightSeats(flightId)])
      .then(([f, s]) => { setFlight(f); setSeats(s); })
      .finally(() => setLoading(false));
  }, [flightId]);

  const pollOrderStatus = useCallback(async () => {
    if (!order?.id) return;
    try {
      const status = await api.getOrderStatus(order.id);
      setOrder(status.order);
      setRemainingSeconds(status.remainingSeconds);
      if (status.order.status === 'confirmed') setStep('confirmed');
      else if (['failed', 'cancelled', 'expired'].includes(status.order.status)) setStep('failed');
    } catch (err) { console.error(err); }
  }, [order?.id]);

  useEffect(() => {
    if (!order?.id) return;
    const interval = setInterval(pollOrderStatus, 2000);
    return () => clearInterval(interval);
  }, [order?.id, pollOrderStatus]);

  const handleCustomerSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!flightId) return;
    setLoading(true);
    try {
      const newOrder = await api.createOrder({ flightId, customerName: customerInfo.name, customerEmail: customerInfo.email });
      setOrder(newOrder);
      setStep('seats');
    } finally { setLoading(false); }
  };

  const handleConfirmSeats = async () => {
    if (!order?.id || selectedSeats.length === 0) return;
    setLoading(true);
    try {
      const status = await api.selectSeats(order.id, selectedSeats);
      setOrder(status.order);
      setRemainingSeconds(status.remainingSeconds);
      setStep('payment');
    } finally { setLoading(false); }
  };

  const handlePayment = async (paymentCode: string) => {
    if (!order?.id) return;
    setLoading(true);
    try {
      await api.submitPayment(order.id, paymentCode);
      setTimeout(pollOrderStatus, 1000);
    } finally { setLoading(false); }
  };

  const totalAmount = selectedSeats.reduce((sum, seatId) => {
    const seat = seats.find((s) => s.id === seatId);
    return sum + (seat?.price || 0);
  }, 0);

  if (loading && !flight) {
    return <div className="flex items-center justify-center min-h-[400px]">
      <div className="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-cyan-500"></div>
    </div>;
  }

  return (
    <div>
      <button onClick={() => navigate('/')} className="text-slate-400 hover:text-white mb-4">← Back</button>
      <h1 className="text-3xl font-bold text-white mb-2">{flight?.flightNumber}: {flight?.origin} → {flight?.destination}</h1>
      
      {order && remainingSeconds > 0 && step === 'payment' && <Timer seconds={remainingSeconds} />}

      <div className="grid lg:grid-cols-3 gap-8 mt-8">
        <div className="lg:col-span-2">
          {step === 'customer' && (
            <form onSubmit={handleCustomerSubmit} className="bg-slate-800/50 border border-slate-700/50 rounded-2xl p-8">
              <h2 className="text-2xl font-semibold text-white mb-6">Customer Information</h2>
              <input type="text" placeholder="Full Name" value={customerInfo.name} onChange={(e) => setCustomerInfo(p => ({...p, name: e.target.value}))} className="w-full px-4 py-3 bg-slate-700 rounded-xl text-white mb-4" required />
              <input type="email" placeholder="Email" value={customerInfo.email} onChange={(e) => setCustomerInfo(p => ({...p, email: e.target.value}))} className="w-full px-4 py-3 bg-slate-700 rounded-xl text-white mb-4" required />
              <button type="submit" className="w-full py-4 bg-cyan-500 text-slate-900 font-semibold rounded-xl">Continue</button>
            </form>
          )}

          {step === 'seats' && (
            <div className="bg-slate-800/50 border border-slate-700/50 rounded-2xl p-8">
              <h2 className="text-2xl font-semibold text-white mb-6">Select Seats</h2>
              <SeatMap seats={seats} selectedSeats={selectedSeats} onSeatSelect={(id) => setSelectedSeats(p => p.includes(id) ? p.filter(s => s !== id) : [...p, id])} />
              <button onClick={handleConfirmSeats} disabled={selectedSeats.length === 0} className="mt-6 w-full py-4 bg-cyan-500 text-slate-900 font-semibold rounded-xl disabled:opacity-50">Confirm Seats</button>
            </div>
          )}

          {step === 'payment' && <PaymentForm onSubmit={handlePayment} loading={loading} attempts={order?.paymentAttempts || 0} />}

          {step === 'confirmed' && (
            <div className="bg-slate-800/50 border border-emerald-500/50 rounded-2xl p-8 text-center">
              <div className="w-20 h-20 mx-auto mb-6 rounded-full bg-emerald-500/20 flex items-center justify-center">
                <svg className="w-10 h-10 text-emerald-500" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" /></svg>
              </div>
              <h2 className="text-3xl font-bold text-white mb-2">Booking Confirmed!</h2>
              <p className="text-emerald-500 mb-8">Order ID: {order?.id}</p>
              <button onClick={() => navigate('/')} className="px-8 py-3 bg-emerald-500 text-slate-900 font-semibold rounded-xl">Book Another</button>
            </div>
          )}

          {step === 'failed' && (
            <div className="bg-slate-800/50 border border-red-500/50 rounded-2xl p-8 text-center">
              <h2 className="text-3xl font-bold text-white mb-2">Booking Failed</h2>
              <p className="text-red-400 mb-8">{order?.failureReason || 'An error occurred'}</p>
              <button onClick={() => navigate('/')} className="px-8 py-3 bg-slate-700 text-white rounded-xl">Try Again</button>
            </div>
          )}
        </div>

        <div className="bg-slate-800/50 border border-slate-700/50 rounded-2xl p-6 h-fit">
          <h3 className="text-xl font-semibold text-white mb-4">Order Summary</h3>
          <div className="space-y-2 text-sm">
            <div className="flex justify-between"><span className="text-slate-400">Flight</span><span className="text-white">{flight?.flightNumber}</span></div>
            {selectedSeats.length > 0 && <div className="flex justify-between"><span className="text-slate-400">Seats</span><span className="text-white">{selectedSeats.map(s => s.split('-')[1]).join(', ')}</span></div>}
            <div className="border-t border-slate-700 pt-3 mt-3 flex justify-between text-lg">
              <span className="text-white font-semibold">Total</span>
              <span className="text-emerald-500 font-bold">${order?.totalAmount || totalAmount}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

