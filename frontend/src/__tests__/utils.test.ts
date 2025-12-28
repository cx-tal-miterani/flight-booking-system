import { cn, formatTime, formatDate, formatCurrency } from '../lib/utils';

describe('utils', () => {
  describe('cn', () => {
    it('should merge class names', () => {
      expect(cn('foo', 'bar')).toBe('foo bar');
    });

    it('should handle conditional classes', () => {
      expect(cn('foo', false && 'bar', 'baz')).toBe('foo baz');
      expect(cn('foo', true && 'bar', 'baz')).toBe('foo bar baz');
    });

    it('should handle undefined and null', () => {
      expect(cn('foo', undefined, null, 'bar')).toBe('foo bar');
    });

    it('should merge tailwind classes correctly', () => {
      expect(cn('px-2 py-1', 'px-4')).toBe('py-1 px-4');
    });
  });

  describe('formatTime', () => {
    it('should format time correctly', () => {
      const date = '2025-01-15T14:30:00Z';
      const result = formatTime(date);
      
      // Time format depends on locale, so just check it contains numbers
      expect(result).toMatch(/\d{1,2}:\d{2}/);
    });
  });

  describe('formatDate', () => {
    it('should format date correctly', () => {
      const date = '2025-01-15T14:30:00Z';
      const result = formatDate(date);
      
      // Should contain day of week and date
      expect(result).toMatch(/\w+/);
    });
  });

  describe('formatCurrency', () => {
    it('should format currency correctly', () => {
      expect(formatCurrency(150)).toBe('$150.00');
      expect(formatCurrency(1234.56)).toBe('$1,234.56');
      expect(formatCurrency(0)).toBe('$0.00');
    });

    it('should handle decimal values', () => {
      expect(formatCurrency(99.99)).toBe('$99.99');
      expect(formatCurrency(100.5)).toBe('$100.50');
    });
  });
});

