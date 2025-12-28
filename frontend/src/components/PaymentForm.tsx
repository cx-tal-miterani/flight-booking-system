import { useState, useRef, useEffect } from 'react';

interface PaymentFormProps {
  onSubmit: (paymentCode: string) => void;
  loading: boolean;
  attempts: number;
}

export function PaymentForm({ onSubmit, loading, attempts }: PaymentFormProps) {
  const [digits, setDigits] = useState(['', '', '', '', '']);
  const inputRefs = useRef<(HTMLInputElement | null)[]>([]);

  useEffect(() => { inputRefs.current[0]?.focus(); }, []);

  const handleChange = (index: number, value: string) => {
    if (!/^\d*$/.test(value)) return;
    const newDigits = [...digits];
    newDigits[index] = value.slice(-1);
    setDigits(newDigits);
    if (value && index < 4) inputRefs.current[index + 1]?.focus();
  };

  const handleKeyDown = (index: number, e: React.KeyboardEvent) => {
    if (e.key === 'Backspace' && !digits[index] && index > 0) inputRefs.current[index - 1]?.focus();
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const code = digits.join('');
    if (code.length === 5) onSubmit(code);
  };

  return (
    <div className="bg-slate-800/50 border border-slate-700/50 rounded-2xl p-8">
      <h2 className="text-2xl font-semibold text-white mb-2">Payment</h2>
      <p className="text-slate-400 mb-6">Enter your 5-digit payment code</p>

      {attempts > 0 && (
        <div className="mb-6 px-4 py-3 rounded-lg bg-amber-500/10 border border-amber-500/30">
          <p className="text-amber-400 text-sm">Payment attempt {attempts}/3 failed. Please try again.</p>
        </div>
      )}

      <form onSubmit={handleSubmit}>
        <div className="flex justify-center gap-3 mb-8">
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
              className="w-14 h-16 text-center text-2xl font-mono font-bold bg-slate-700 border-2 border-slate-600 rounded-xl text-white focus:border-cyan-500 focus:outline-none"
            />
          ))}
        </div>

        <button type="submit" disabled={digits.some(d => !d) || loading}
          className="w-full py-4 bg-emerald-500 text-slate-900 font-semibold rounded-xl disabled:opacity-50">
          {loading ? 'Processing...' : 'Pay Now'}
        </button>
      </form>

      <p className="mt-6 text-center text-slate-500 text-sm">
        For testing, use any 5-digit code. 15% simulated failure rate.
      </p>
    </div>
  );
}

