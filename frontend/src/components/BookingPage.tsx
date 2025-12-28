import { useState, useEffect, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { ArrowLeft, Check, X, Loader2, Plane, RefreshCw } from 'lucide-react';
import { api } from '../api';
import type { Flight, Seat, Order, OrderStatus } from '../types';
import { SeatMap } from './SeatMap';
import { Timer } from './Timer';
import { PaymentForm } from './PaymentForm';
import { Button } from './ui/button';
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from './ui/card';
import { Input } from './ui/input';
import { Badge } from './ui/badge';
import { Alert, AlertDescription } from './ui/alert';
import { formatCurrency } from '../lib/utils';

type BookingStep = 'customer' | 'seats' | 'payment' | 'confirmed' | 'failed';

const STEP_ORDER: BookingStep[] = ['customer', 'seats', 'payment', 'confirmed'];

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
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [customerInfo, setCustomerInfo] = useState({ name: '', email: '' });

  // Fetch flight and seats
  useEffect(() => {
    if (!flightId) return;

    Promise.all([api.getFlight(flightId), api.getFlightSeats(flightId)])
      .then(([flightData, seatsData]) => {
        setFlight(flightData);
        setSeats(seatsData);
      })
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, [flightId]);

  // Poll order status for real-time updates
  const pollOrderStatus = useCallback(async () => {
    if (!order?.id) return;
    
    try {
      const status = await api.getOrderStatus(order.id);
      setOrder(status.order);
      setRemainingSeconds(status.remainingSeconds);

      // Update step based on status
      const statusStepMap: Partial<Record<OrderStatus, BookingStep>> = {
        pending: 'seats',
        seats_selected: step === 'payment' ? 'payment' : 'seats',
        awaiting_payment: 'payment',
        processing: 'payment',
        confirmed: 'confirmed',
        failed: 'failed',
        cancelled: 'failed',
        expired: 'failed',
      };
      
      const newStep = statusStepMap[status.order.status];
      if (newStep && newStep !== step && step !== 'payment') {
        setStep(newStep);
      }
      
      // Check for terminal states
      if (['confirmed', 'failed', 'cancelled', 'expired'].includes(status.order.status)) {
        setStep(status.order.status === 'confirmed' ? 'confirmed' : 'failed');
      }
    } catch (err) {
      console.error('Failed to poll status:', err);
    }
  }, [order?.id, step]);

  // Poll every 2 seconds when order exists
  useEffect(() => {
    if (!order?.id) return;
    
    const interval = setInterval(pollOrderStatus, 2000);
    return () => clearInterval(interval);
  }, [order?.id, pollOrderStatus]);

  const handleCustomerSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!flightId || !customerInfo.name || !customerInfo.email) return;

    setSubmitting(true);
    setError(null);
    try {
      const newOrder = await api.createOrder({
        flightId,
        customerName: customerInfo.name,
        customerEmail: customerInfo.email,
      });
      setOrder(newOrder);
      setStep('seats');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create order');
    } finally {
      setSubmitting(false);
    }
  };

  const handleSeatSelect = (seatId: string) => {
    setSelectedSeats((prev) =>
      prev.includes(seatId)
        ? prev.filter((id) => id !== seatId)
        : [...prev, seatId]
    );
  };

  const handleConfirmSeats = async () => {
    if (!order?.id || selectedSeats.length === 0) return;

    setSubmitting(true);
    setError(null);
    try {
      const status = await api.selectSeats(order.id, selectedSeats);
      setOrder(status.order);
      setRemainingSeconds(status.remainingSeconds);
      setStep('payment');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to select seats');
    } finally {
      setSubmitting(false);
    }
  };

  // Handle seat changes during payment step (refreshes timer)
  const handleModifySeats = async () => {
    if (!order?.id || selectedSeats.length === 0) return;

    setSubmitting(true);
    try {
      const status = await api.selectSeats(order.id, selectedSeats);
      setOrder(status.order);
      setRemainingSeconds(status.remainingSeconds); // Timer refreshes!
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update seats');
    } finally {
      setSubmitting(false);
    }
  };

  const handlePayment = async (paymentCode: string) => {
    if (!order?.id) return;

    setSubmitting(true);
    setError(null);
    try {
      const status = await api.submitPayment(order.id, paymentCode);
      setOrder(status.order);
      
      // Poll for final status after payment
      setTimeout(pollOrderStatus, 1000);
      setTimeout(pollOrderStatus, 2000);
      setTimeout(pollOrderStatus, 3000);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Payment failed');
    } finally {
      setSubmitting(false);
    }
  };

  const handleCancel = async () => {
    if (!order?.id) return;

    try {
      await api.cancelOrder(order.id);
      navigate('/');
    } catch (err) {
      console.error('Failed to cancel:', err);
    }
  };

  const handleRefreshTimer = async () => {
    if (!order?.id) return;
    try {
      // Re-selecting same seats refreshes the timer
      const status = await api.selectSeats(order.id, selectedSeats);
      setOrder(status.order);
      setRemainingSeconds(status.remainingSeconds);
    } catch (err) {
      console.error('Failed to refresh timer:', err);
    }
  };

  const totalAmount = selectedSeats.reduce((sum, seatId) => {
    const seat = seats.find((s) => s.id === seatId);
    return sum + (seat?.price || 0);
  }, 0);

  if (loading) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[400px] gap-4">
        <Loader2 className="w-12 h-12 text-cyan-500 animate-spin" />
        <p className="text-slate-400">Loading flight details...</p>
      </div>
    );
  }

  if (error && !order) {
    return (
      <div className="max-w-lg mx-auto">
        <Alert variant="danger">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
        <Button variant="outline" onClick={() => navigate('/')} className="mt-4">
          <ArrowLeft className="w-4 h-4 mr-2" />
          Back to Flights
        </Button>
      </div>
    );
  }

  const currentStepIndex = STEP_ORDER.indexOf(step);

  return (
    <div className="animate-fade-in">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mb-6">
        <div>
          <Button variant="ghost" onClick={() => navigate('/')} className="mb-2 -ml-2">
            <ArrowLeft className="w-4 h-4 mr-2" />
            Back to flights
          </Button>
          <h1 className="text-2xl sm:text-3xl font-bold text-white flex items-center gap-2">
            <Plane className="w-6 h-6 text-cyan-500" />
            {flight?.flightNumber}: {flight?.origin.split('(')[0]} → {flight?.destination.split('(')[0]}
          </h1>
        </div>

        {order && !['confirmed', 'failed'].includes(step) && (
          <Button variant="ghost" onClick={handleCancel} className="text-red-400 hover:text-red-300">
            <X className="w-4 h-4 mr-2" />
            Cancel Booking
          </Button>
        )}
      </div>

      {/* Progress Steps */}
      <div className="flex items-center gap-2 mb-8 overflow-x-auto pb-2">
        {['Customer Info', 'Select Seats', 'Payment', 'Confirmation'].map((label, index) => {
          const isActive = index === currentStepIndex;
          const isComplete = index < currentStepIndex;

          return (
            <div key={label} className="flex items-center gap-2 flex-shrink-0">
              <div
                className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-semibold transition-all ${
                  isComplete
                    ? 'bg-emerald-500 text-slate-900'
                    : isActive
                    ? 'bg-cyan-500 text-slate-900'
                    : 'bg-slate-700 text-slate-400'
                }`}
              >
                {isComplete ? <Check className="w-4 h-4" /> : index + 1}
              </div>
              <span className={`text-sm whitespace-nowrap ${isActive ? 'text-white font-medium' : 'text-slate-500'}`}>
                {label}
              </span>
              {index < 3 && (
                <div className={`w-8 sm:w-12 h-0.5 ${isComplete ? 'bg-emerald-500' : 'bg-slate-700'}`} />
              )}
            </div>
          );
        })}
      </div>

      {/* Timer - show during seats and payment steps */}
      {order && remainingSeconds > 0 && ['seats', 'payment'].includes(step) && (
        <div className="mb-6">
          <Timer 
            seconds={remainingSeconds} 
            totalSeconds={900}
            onRefresh={step === 'payment' ? handleRefreshTimer : undefined}
          />
        </div>
      )}

      {error && (
        <Alert variant="danger" className="mb-6">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {/* Main Content */}
      <div className="grid lg:grid-cols-3 gap-6 lg:gap-8">
        <div className="lg:col-span-2">
          {/* Customer Info Step */}
          {step === 'customer' && (
            <Card>
              <CardHeader>
                <CardTitle>Customer Information</CardTitle>
                <CardDescription>Please enter your details to begin booking</CardDescription>
              </CardHeader>
              <CardContent>
                <form onSubmit={handleCustomerSubmit} className="space-y-4">
                  <div>
                    <label className="block text-sm text-slate-400 mb-2">Full Name</label>
                    <Input
                      type="text"
                      value={customerInfo.name}
                      onChange={(e) => setCustomerInfo((prev) => ({ ...prev, name: e.target.value }))}
                      placeholder="John Doe"
                      required
                    />
                  </div>
                  <div>
                    <label className="block text-sm text-slate-400 mb-2">Email Address</label>
                    <Input
                      type="email"
                      value={customerInfo.email}
                      onChange={(e) => setCustomerInfo((prev) => ({ ...prev, email: e.target.value }))}
                      placeholder="john@example.com"
                      required
                    />
                  </div>
                  <Button type="submit" className="w-full" disabled={submitting}>
                    {submitting ? (
                      <>
                        <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                        Creating Order...
                      </>
                    ) : (
                      'Continue to Seat Selection'
                    )}
                  </Button>
                </form>
              </CardContent>
            </Card>
          )}

          {/* Seat Selection Step */}
          {step === 'seats' && (
            <Card>
              <CardHeader>
                <CardTitle>Select Your Seats</CardTitle>
                <CardDescription>
                  Choose your preferred seats. Selected: {selectedSeats.length}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <SeatMap
                  seats={seats}
                  selectedSeats={selectedSeats}
                  onSeatSelect={handleSeatSelect}
                />
              </CardContent>
              <CardFooter>
                <Button
                  onClick={handleConfirmSeats}
                  disabled={selectedSeats.length === 0 || submitting}
                  className="w-full"
                >
                  {submitting ? (
                    <>
                      <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                      Reserving Seats...
                    </>
                  ) : (
                    `Confirm ${selectedSeats.length} Seat${selectedSeats.length !== 1 ? 's' : ''} & Continue`
                  )}
                </Button>
              </CardFooter>
            </Card>
          )}

          {/* Payment Step */}
          {step === 'payment' && (
            <div className="space-y-6">
              <PaymentForm
                onSubmit={handlePayment}
                loading={submitting}
                attempts={order?.paymentAttempts || 0}
                maxAttempts={3}
              />
              
              {/* Option to modify seats */}
              <Card>
                <CardHeader>
                  <CardTitle className="text-lg">Need to change seats?</CardTitle>
                  <CardDescription>
                    You can modify your seat selection. The timer will refresh.
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <details className="group">
                    <summary className="cursor-pointer text-cyan-500 hover:text-cyan-400 flex items-center gap-2">
                      <RefreshCw className="w-4 h-4" />
                      Modify Seat Selection
                    </summary>
                    <div className="mt-4">
                      <SeatMap
                        seats={seats}
                        selectedSeats={selectedSeats}
                        onSeatSelect={handleSeatSelect}
                      />
                      <Button
                        variant="outline"
                        onClick={handleModifySeats}
                        disabled={submitting}
                        className="mt-4"
                      >
                        Update Seats & Refresh Timer
                      </Button>
                    </div>
                  </details>
                </CardContent>
              </Card>
            </div>
          )}

          {/* Confirmed Step */}
          {step === 'confirmed' && (
            <Card className="border-emerald-500/50">
              <CardContent className="pt-8 text-center">
                <div className="w-20 h-20 mx-auto mb-6 rounded-full bg-emerald-500/20 flex items-center justify-center">
                  <Check className="w-10 h-10 text-emerald-500" />
                </div>
                <h2 className="text-3xl font-bold text-white mb-2">Booking Confirmed!</h2>
                <p className="text-slate-400 mb-4">Your flight has been successfully booked.</p>
                <Badge variant="success" className="text-lg px-4 py-2 mb-8">
                  Order ID: {order?.id}
                </Badge>
                <Button variant="success" onClick={() => navigate('/')}>
                  Book Another Flight
                </Button>
              </CardContent>
            </Card>
          )}

          {/* Failed Step */}
          {step === 'failed' && (
            <Card className="border-red-500/50">
              <CardContent className="pt-8 text-center">
                <div className="w-20 h-20 mx-auto mb-6 rounded-full bg-red-500/20 flex items-center justify-center">
                  <X className="w-10 h-10 text-red-500" />
                </div>
                <h2 className="text-3xl font-bold text-white mb-2">Booking Failed</h2>
                <p className="text-red-400 mb-8">{order?.failureReason || 'An error occurred during booking'}</p>
                <Button variant="secondary" onClick={() => navigate('/')}>
                  Try Again
                </Button>
              </CardContent>
            </Card>
          )}
        </div>

        {/* Order Summary Sidebar */}
        <div className="lg:col-span-1">
          <Card className="sticky top-24">
            <CardHeader>
              <CardTitle className="text-lg">Order Summary</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex justify-between text-sm">
                <span className="text-slate-400">Flight</span>
                <span className="text-white font-medium">{flight?.flightNumber}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-slate-400">Route</span>
                <span className="text-white">{flight?.origin.split('(')[0]} → {flight?.destination.split('(')[0]}</span>
              </div>
              {selectedSeats.length > 0 && (
                <div className="flex justify-between text-sm">
                  <span className="text-slate-400">Seats ({selectedSeats.length})</span>
                  <span className="text-white">{selectedSeats.map(s => s.split('-')[1]).join(', ')}</span>
                </div>
              )}
              <div className="border-t border-slate-700 pt-4">
                <div className="flex justify-between text-lg">
                  <span className="text-white font-semibold">Total</span>
                  <span className="text-emerald-500 font-bold">
                    {formatCurrency(order?.totalAmount || totalAmount)}
                  </span>
                </div>
              </div>

              {order && (
                <div className="pt-4 border-t border-slate-700">
                  <div className="flex items-center gap-2">
                    <div className={`w-2 h-2 rounded-full ${
                      order.status === 'confirmed' ? 'bg-emerald-500' :
                      ['failed', 'cancelled', 'expired'].includes(order.status) ? 'bg-red-500' :
                      'bg-amber-500 animate-pulse'
                    }`} />
                    <span className="text-sm text-slate-400 capitalize">
                      Status: {order.status.replace('_', ' ')}
                    </span>
                  </div>
                  {order.paymentAttempts > 0 && (
                    <p className="text-xs text-slate-500 mt-1">
                      Payment attempts: {order.paymentAttempts}/3
                    </p>
                  )}
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
