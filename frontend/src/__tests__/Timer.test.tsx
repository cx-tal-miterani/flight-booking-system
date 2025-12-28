import { render, screen, act } from '@testing-library/react';
import { Timer } from '../components/Timer';

describe('Timer', () => {
  beforeEach(() => {
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it('should display time correctly', () => {
    render(<Timer seconds={125} />);
    
    expect(screen.getByText('2:05')).toBeInTheDocument();
  });

  it('should countdown every second', () => {
    render(<Timer seconds={65} />);
    
    expect(screen.getByText('1:05')).toBeInTheDocument();
    
    act(() => {
      jest.advanceTimersByTime(1000);
    });
    
    expect(screen.getByText('1:04')).toBeInTheDocument();
    
    act(() => {
      jest.advanceTimersByTime(5000);
    });
    
    expect(screen.getByText('0:59')).toBeInTheDocument();
  });

  it('should not go below zero', () => {
    render(<Timer seconds={2} />);
    
    act(() => {
      jest.advanceTimersByTime(5000);
    });
    
    expect(screen.getByText('0:00')).toBeInTheDocument();
  });

  it('should show warning message when time is low', () => {
    render(<Timer seconds={90} />);
    
    expect(screen.getByText(/Less than 2 minutes/i)).toBeInTheDocument();
  });

  it('should show critical message when time is very low', () => {
    render(<Timer seconds={30} />);
    
    expect(screen.getByText(/Hurry/i)).toBeInTheDocument();
  });

  it('should update when seconds prop changes', () => {
    const { rerender } = render(<Timer seconds={100} />);
    
    expect(screen.getByText('1:40')).toBeInTheDocument();
    
    rerender(<Timer seconds={200} />);
    
    expect(screen.getByText('3:20')).toBeInTheDocument();
  });

  it('should show refresh button when onRefresh is provided', () => {
    const onRefresh = jest.fn();
    render(<Timer seconds={100} onRefresh={onRefresh} />);
    
    expect(screen.getByText('Refresh Timer')).toBeInTheDocument();
  });

  it('should call onRefresh when refresh button is clicked', () => {
    const onRefresh = jest.fn();
    render(<Timer seconds={100} onRefresh={onRefresh} />);
    
    screen.getByText('Refresh Timer').click();
    
    expect(onRefresh).toHaveBeenCalledTimes(1);
  });
});

