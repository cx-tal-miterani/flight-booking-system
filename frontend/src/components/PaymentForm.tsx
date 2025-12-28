import { useState, useRef, useEffect } from 'react';
import { Loader2, Lock, RefreshCw } from 'lucide-react';
import { Button } from './ui/button';
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from './ui/card';
import { Alert, AlertDescription } from './ui/alert';

interface PaymentFormProps {
  onSubmit: (paymentCode: string) => void;
  loading: boolean;
  attempts: number;
  maxAttempts?: number;
}

export function PaymentForm({ onSubmit, loading, attempts, maxAttempts = 3 }: PaymentFormProps) {
  const [digits, setDigits] = useState(['', '', '', '', '']);
  const inputRefs = useRef<(HTMLInputElement | null)[]>([]);

  useEffect(() => {
    inputRefs.current[0]?.focus();
  }, []);

  // Clear digits on retry
  useEffect(() => {
    if (attempts > 0) {
      setDigits(['', '', '', '', '']);
      inputRefs.current[0]?.focus();
    }
  }, [attempts]);

  const handleChange = (index: number, value: string) => {
    if (!/^\d*$/.test(value)) return;
    
    const newDigits = [...digits];
    newDigits[index] = value.slice(-1);
    setDigits(newDigits);
    
    if (value && index < 4) {
      inputRefs.current[index + 1]?.focus();
    }
  };

  const handleKeyDown = (index: number, e: React.KeyboardEvent) => {
    if (e.key === 'Backspace' && !digits[index] && index > 0) {
      inputRefs.current[index - 1]?.focus();
    }
  };

  const handlePaste = (e: React.ClipboardEvent) => {
    e.preventDefault();
    const pastedData = e.clipboardData.getData('text').replace(/\D/g, '').slice(0, 5);
    const newDigits = ['', '', '', '', ''];
    for (let i = 0; i < pastedData.length; i++) {
      newDigits[i] = pastedData[i];
    }
    setDigits(newDigits);
    inputRefs.current[Math.min(pastedData.length, 4)]?.focus();
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const code = digits.join('');
    if (code.length === 5) {
      onSubmit(code);
    }
  };

  const isComplete = digits.every((d) => d !== '');
  const remainingAttempts = maxAttempts - attempts;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Lock className="w-5 h-5 text-cyan-500" />
          Payment
        </CardTitle>
        <CardDescription>
          Enter your 5-digit payment code to complete the booking
        </CardDescription>
      </CardHeader>
      <CardContent>
        {attempts > 0 && (
          <Alert variant="warning" className="mb-6">
            <AlertDescription>
              <div className="flex items-center gap-2">
                <RefreshCw className="w-4 h-4" />
                Payment attempt {attempts}/{maxAttempts} failed. 
                {remainingAttempts > 0 
                  ? ` ${remainingAttempts} attempt${remainingAttempts > 1 ? 's' : ''} remaining.`
                  : ' This is your last attempt!'
                }
              </div>
            </AlertDescription>
          </Alert>
        )}

        <form onSubmit={handleSubmit}>
          <div 
            className="flex justify-center gap-2 sm:gap-3 mb-8" 
            onPaste={handlePaste}
          >
            {digits.map((digit, index) => (
              <input
                key={index}
                ref={(el) => (inputRefs.current[index] = el)}
                type="text"
                inputMode="numeric"
                maxLength={1}
                value={digit}
                onChange={(e) => handleChange(index, e.target.value)}
                onKeyDown={(e) => handleKeyDown(index, e)}
                disabled={loading}
                className="w-12 h-14 sm:w-14 sm:h-16 text-center text-xl sm:text-2xl font-mono font-bold bg-slate-700/50 border-2 border-slate-600 rounded-xl text-white focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 disabled:opacity-50 transition-all"
                aria-label={`Digit ${index + 1}`}
              />
            ))}
          </div>

          <Button
            type="submit"
            variant="success"
            size="lg"
            disabled={!isComplete || loading}
            className="w-full"
          >
            {loading ? (
              <>
                <Loader2 className="w-5 h-5 mr-2 animate-spin" />
                Processing Payment...
              </>
            ) : (
              <>
                <Lock className="w-5 h-5 mr-2" />
                Pay Now
              </>
            )}
          </Button>
        </form>

        <div className="mt-6 p-4 bg-slate-800/50 rounded-lg">
          <p className="text-sm text-slate-400 text-center">
            <strong>Testing Info:</strong> Use any 5-digit code (e.g., 12345).
            <br />
            15% simulated failure rate with up to 3 retry attempts.
          </p>
        </div>
      </CardContent>
    </Card>
  );
}
