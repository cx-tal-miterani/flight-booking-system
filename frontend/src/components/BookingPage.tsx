import { useState, useEffect, useCallback, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { ArrowLeft, Check, X, Loader2, Plane, RefreshCw, Wifi, WifiOff } from 'lucide-react';
import { api } from '../api';
import type { Flight, Seat, Order } from '../types';
import { SeatMap } from './SeatMap';
import { Timer } from './Timer';
import { PaymentForm } from './PaymentForm';
import { Button } from './ui/button';
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from './ui/card';
import { Input } from './ui/input';
import { Badge } from './ui/badge';
import { Alert, AlertDescription } from './ui/alert';
import { formatCurrency } from '../lib/utils';
import { useFlightWebSocket, applySeatUpdates, SeatUpdate } from '../hooks/useFlightWebSocket';
import { useSeatConflictToast, useSeatsReleasedToast } from './ui/toast';
import { saveOrderSession, getOrderSession, clearOrderSession } from '../lib/orderSession';

type BookingStep = 'customer' | 'seats' | 'payment' | 'confirmed' | 'failed';

const STEP_ORDER: BookingStep[] = ['customer', 'seats', 'payment', 'confirmed'];

export function BookingPage() {
  const { flightId } = useParams<{ flightId: string }>();
  const navigate = useNavigate();
  const showSeatConflict = useSeatConflictToast();
  const showSeatsReleased = useSeatsReleasedToast();
  
  const [flight, setFlight] = useState<Flight | null>(null);
  const [seats, setSeats] = useState<Seat[]>([]);
  const [selectedSeats, setSelectedSeats] = useState<string[]>([]);
  const [order, setOrder] = useState<Order | null>(null);
  const [remainingSeconds, setRemainingSeconds] = useState(0);
  const [step, setStep] = useState<BookingStep>('customer');
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showModifySeats, setShowModifySeats] = useState(false);
  const [customerInfo, setCustomerInfo] = useState({ name: '', email: '' });
  const [restoringSession, setRestoringSession] = useState(true);
  const sessionRestored = useRef(false);

  // WebSocket handlers for real-time updates
  const handleSeatsUpdated = useCallback((updates: SeatUpdate[]) => {
    setSeats((prevSeats) => applySeatUpdates(prevSeats, updates));
    
    // Check if any of our selected seats were taken by someone else
    const takenSeats = updates.filter(
      (u) => u.status === 'held' && selectedSeats.includes(u.seatId) && u.heldBy !== order?.id
    );
    
    if (takenSeats.length > 0 && step === 'seats') {
      // Remove taken seats from selection
      setSelectedSeats((prev) => prev.filter((id) => !takenSeats.some((t) => t.seatId === id)));
      showSeatConflict(takenSeats.map((t) => t.seatId));
    }
  }, [selectedSeats, order?.id, step, showSeatConflict]);

  const handleSeatConflict = useCallback((conflictSeats: SeatUpdate[], message: string) => {
    setSeats((prevSeats) => applySeatUpdates(prevSeats, conflictSeats));
    showSeatConflict(conflictSeats.map((s) => s.seatId));
    setError(message);
  }, [showSeatConflict]);

  const handleOrderCompleted = useCallback((completedOrderId: string, seatUpdates: SeatUpdate[]) => {
    // Always refresh seats from server for accuracy
    if (flightId) {
      api.getFlightSeats(flightId).then(setSeats).catch(console.error);
    }
    
    if (completedOrderId === order?.id) {
      // OUR order completed - show confirmation!
      setOrder((prev) => prev ? { ...prev, status: 'confirmed' } : prev);
      setStep('confirmed');
      if (flightId) {
        clearOrderSession(flightId);
      }
    } else {
      // Another order completed - check if it affects our selection
      const bookedSeats = seatUpdates.filter((s) => selectedSeats.includes(s.seatId));
      if (bookedSeats.length > 0 && step === 'seats') {
        setSelectedSeats((prev) => prev.filter((id) => !bookedSeats.some((b) => b.seatId === id)));
        showSeatConflict(bookedSeats.map((s) => s.seatId));
      }
    }
  }, [order?.id, flightId, selectedSeats, step, showSeatConflict]);

  const handleOrderExpired = useCallback((expiredOrderId: string, seatUpdates: SeatUpdate[]) => {
    console.log('handleOrderExpired called:', { expiredOrderId, currentOrderId: order?.id, seatUpdates });
    
    // Always refresh seats from server for accuracy
    if (flightId) {
      api.getFlightSeats(flightId).then(setSeats).catch(console.error);
    }
    
    if (expiredOrderId === order?.id) {
      // OUR order expired - show failure!
      console.log('OUR order expired - setting step to failed');
      setOrder((prev) => prev ? { ...prev, status: 'expired', failureReason: 'Reservation expired' } : prev);
      setStep('failed');
      if (flightId) {
        clearOrderSession(flightId);
      }
    } else {
      // Another order expired - seats are now available
      showSeatsReleased(seatUpdates.map((s) => s.seatId));
    }
  }, [order?.id, flightId, showSeatsReleased]);

  // Handle seats voluntarily released (NOT order expiry - just seat modification)
  const handleSeatsReleased = useCallback((releasedOrderId: string, seatUpdates: SeatUpdate[]) => {
    // Update seat status
    setSeats((prevSeats) => applySeatUpdates(prevSeats, seatUpdates));
    
    // Only show notification to OTHER users
    if (releasedOrderId !== order?.id) {
      showSeatsReleased(seatUpdates.map((s) => s.seatId));
    }
    // For the owner: nothing special - they know they released the seats
  }, [order?.id, showSeatsReleased]);

  // Connect to WebSocket for real-time updates
  const { isConnected } = useFlightWebSocket({
    flightId,
    orderId: order?.id,
    onSeatsUpdated: handleSeatsUpdated,
    onSeatConflict: handleSeatConflict,
    onOrderCompleted: handleOrderCompleted,
    onOrderExpired: handleOrderExpired,
    onSeatsReleased: handleSeatsReleased,
  });

  // Restore session from localStorage on mount
  useEffect(() => {
    if (!flightId || sessionRestored.current) return;

    const session = getOrderSession(flightId);
    if (!session) {
      // No session to restore - let the other useEffect handle loading
      sessionRestored.current = false; // Explicitly false so other effect runs
      setRestoringSession(false);
      return;
    }
    
    // Mark that we're handling session restoration
    sessionRestored.current = true;

    // Fetch order status, flight, and seats in parallel for session restoration
    Promise.all([
      api.getOrderStatus(session.orderId),
      api.getFlight(flightId),
      api.getFlightSeats(flightId),
    ])
      .then(([status, flightData, seatsData]) => {
        const orderStatus = status.order.status;
        
        // Check if order is in a terminal state
        if (['confirmed', 'failed', 'cancelled', 'expired'].includes(orderStatus)) {
          // Clear session and start fresh, but keep the flight/seat data
          clearOrderSession(flightId);
          setFlight(flightData);
          setSeats(seatsData);
          setLoading(false);
          setRestoringSession(false);
          return;
        }

        // Check if timer has expired
        if (status.remainingSeconds <= 0) {
          clearOrderSession(flightId);
          setFlight(flightData);
          setSeats(seatsData);
          setLoading(false);
          setRestoringSession(false);
          return;
        }

        // Restore the session
        setOrder(status.order);
        setRemainingSeconds(status.remainingSeconds);
        setCustomerInfo({
          name: session.customerName,
          email: session.customerEmail,
        });
        setFlight(flightData);
        setSeats(seatsData);

        // Restore selected seats from order - convert seat numbers to IDs
        if (status.order.seats && status.order.seats.length > 0) {
          const seatIds = seatsData
            .filter((s) => status.order.seats.includes(`${s.row}${s.column}`))
            .map((s) => s.id);
          setSelectedSeats(seatIds);
          console.log('Restored selected seats:', seatIds);
        }

        // Determine which step to restore to
        if (orderStatus === 'pending') {
          setStep('seats');
        } else if (['seats_selected', 'awaiting_payment', 'processing'].includes(orderStatus)) {
          setStep('payment');
        }

        console.log('Session restored for order:', session.orderId);
        
        // IMPORTANT: Mark restoration as complete
        setLoading(false);
        setRestoringSession(false);
      })
      .catch((err) => {
        console.error('Failed to restore session:', err);
        // Order not found or error - clear session and load fresh data
        clearOrderSession(flightId);
        // Load flight and seats fresh
        Promise.all([api.getFlight(flightId), api.getFlightSeats(flightId)])
          .then(([flightData, seatsData]) => {
            setFlight(flightData);
            setSeats(seatsData);
          })
          .catch((e) => setError(e.message))
          .finally(() => {
            setLoading(false);
            setRestoringSession(false);
          });
      });
  }, [flightId]);

  // Fetch flight and seats (only when no session to restore)
  useEffect(() => {
    if (!flightId) return;

    // Skip if session restoration already handled data loading
    if (sessionRestored.current) return;
    
    // Check if there's a session to restore - if so, the other effect will handle it
    const session = getOrderSession(flightId);
    if (session) return;

    // No session - load data fresh
    setRestoringSession(false);
    Promise.all([api.getFlight(flightId), api.getFlightSeats(flightId)])
      .then(([flightData, seatsData]) => {
        setFlight(flightData);
        setSeats(seatsData);
      })
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, [flightId]);

  // Check order status (used after payment submission as fallback)
  const checkOrderStatus = useCallback(async () => {
    if (!order?.id) return;
    
    console.log('checkOrderStatus called for order:', order.id);
    
    try {
      const status = await api.getOrderStatus(order.id);
      console.log('checkOrderStatus response:', status.order.status, 'remaining:', status.remainingSeconds);
      setOrder(status.order);
      setRemainingSeconds(status.remainingSeconds);
      
      // Check for terminal states
      if (['confirmed', 'failed', 'cancelled', 'expired'].includes(status.order.status)) {
        console.log('checkOrderStatus: Terminal state detected, setting step to', status.order.status === 'confirmed' ? 'confirmed' : 'failed');
        setStep(status.order.status === 'confirmed' ? 'confirmed' : 'failed');
        // Clear session on terminal state
        if (flightId) {
          clearOrderSession(flightId);
        }
      }
    } catch (err) {
      console.error('Failed to check order status:', err);
    }
  }, [order?.id, flightId]);

  // Handle local timer expiry - check with server if order expired
  useEffect(() => {
    if (remainingSeconds === 0 && order?.id && ['seats', 'payment'].includes(step)) {
      console.log('Timer reached 0 - checking order status');
      // Timer reached 0, check with server if order expired
      checkOrderStatus();
    }
  }, [remainingSeconds, order?.id, step, checkOrderStatus]);

  // Decrement timer locally every second
  useEffect(() => {
    if (remainingSeconds <= 0 || !['seats', 'payment'].includes(step)) return;
    
    const interval = setInterval(() => {
      setRemainingSeconds((prev) => Math.max(0, prev - 1));
    }, 1000);
    
    return () => clearInterval(interval);
  }, [remainingSeconds, step]);

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
      
      // Save session for page refresh recovery
      saveOrderSession(flightId, newOrder.id, customerInfo.name, customerInfo.email);
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
      const errorMsg = err instanceof Error ? err.message : 'Failed to select seats';
      // Check for seat conflict (409 status)
      if (errorMsg.includes('not available') || errorMsg.includes('conflict')) {
        showSeatConflict(selectedSeats);
        // Refresh seats to get latest status
        if (flightId) {
          api.getFlightSeats(flightId).then(setSeats).catch(console.error);
        }
        setSelectedSeats([]);
      }
      setError(errorMsg);
    } finally {
      setSubmitting(false);
    }
  };

  // Handle seat changes during payment step (refreshes timer)
  const handleModifySeats = async () => {
    if (!order?.id || selectedSeats.length === 0) return;

    setSubmitting(true);
    setError(null);
    try {
      const status = await api.selectSeats(order.id, selectedSeats);
      setOrder(status.order);
      setRemainingSeconds(status.remainingSeconds); // Timer refreshes!
      
      // Clear selection and collapse accordion - seats now show as "held"
      setSelectedSeats([]);
      setShowModifySeats(false);
      
      // Refresh seats to show updated status
      if (flightId) {
        api.getFlightSeats(flightId).then(setSeats).catch(console.error);
      }
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Failed to update seats';
      // Check for seat conflict
      if (errorMsg.includes('not available') || errorMsg.includes('conflict')) {
        showSeatConflict(selectedSeats);
        // Refresh seats to get latest status
        if (flightId) {
          api.getFlightSeats(flightId).then(setSeats).catch(console.error);
        }
      }
      setError(errorMsg);
    } finally {
      setSubmitting(false);
    }
  };

  const handlePayment = async (paymentCode: string) => {
    // Prevent double submission
    if (!order?.id || submitting) return;

    setSubmitting(true);
    setError(null);
    try {
      const status = await api.submitPayment(order.id, paymentCode);
      setOrder(status.order);
      
      // WebSocket will notify us of completion, but check as fallback
      // in case WebSocket message is missed
      setTimeout(checkOrderStatus, 2000);
      setTimeout(checkOrderStatus, 5000);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Payment failed');
    } finally {
      setSubmitting(false);
    }
  };

  const handleCancel = async () => {
    if (!order?.id || !flightId) return;

    try {
      await api.cancelOrder(order.id);
      // Clear session so refresh doesn't restore cancelled order
      clearOrderSession(flightId);
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

  if (loading || restoringSession) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[400px] gap-4">
        <Loader2 className="w-12 h-12 text-cyan-500 animate-spin" />
        <p className="text-slate-400">
          {restoringSession ? 'Restoring your session...' : 'Loading flight details...'}
        </p>
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
                  ownHeldSeats={
                    // Convert seat numbers ("1A") to seat IDs (UUIDs)
                    seats
                      .filter((s) => order?.seats?.includes(`${s.row}${s.column}`))
                      .map((s) => s.id)
                  }
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
              
              {/* Modify Seat Selection Accordion */}
              <Card>
                <CardHeader className="cursor-pointer" onClick={() => {
                  const opening = !showModifySeats;
                  setShowModifySeats(opening);
                  
                  // When opening, pre-select the currently held seats
                  if (opening && order?.seats && order.seats.length > 0) {
                    const heldSeatIds = seats
                      .filter((s) => order.seats.includes(`${s.row}${s.column}`))
                      .map((s) => s.id);
                    setSelectedSeats(heldSeatIds);
                  }
                }}>
                  <div className="flex items-center justify-between">
                    <CardTitle className="text-lg">Modify Seat Selection</CardTitle>
                    <span className={`transform transition-transform ${showModifySeats ? 'rotate-180' : ''}`}>
                      ▼
                    </span>
                  </div>
                  <CardDescription>
                    Change your selected seats (timer will refresh)
                  </CardDescription>
                </CardHeader>
                {showModifySeats && (
                  <CardContent className="space-y-4">
                      <SeatMap
                        seats={seats}
                        selectedSeats={selectedSeats}
                        onSeatSelect={handleSeatSelect}
                        ownHeldSeats={
                          // Convert seat numbers ("1A") to seat IDs (UUIDs)
                          seats
                            .filter((s) => order?.seats?.includes(`${s.row}${s.column}`))
                            .map((s) => s.id)
                        }
                      />
                      <Button
                        onClick={handleModifySeats}
                      disabled={selectedSeats.length === 0 || submitting}
                      className="w-full"
                    >
                      {submitting ? (
                        <>
                          <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                          Updating Seats...
                        </>
                      ) : (
                        <>
                          <RefreshCw className="w-4 h-4 mr-2" />
                        Update Seats & Refresh Timer
                        </>
                      )}
                      </Button>
                </CardContent>
                )}
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

              {/* Real-time connection status */}
              <div className="pt-4 border-t border-slate-700">
                <div className="flex items-center gap-2 text-xs">
                  {isConnected ? (
                    <>
                      <Wifi className="w-3 h-3 text-emerald-400" />
                      <span className="text-emerald-400">Live updates active</span>
                    </>
                  ) : (
                    <>
                      <WifiOff className="w-3 h-3 text-slate-500" />
                      <span className="text-slate-500">Connecting...</span>
                    </>
                  )}
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
