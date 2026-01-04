import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { PaymentForm } from '../components/PaymentForm';

describe('PaymentForm', () => {
  it('should render 5 digit inputs', () => {
    render(<PaymentForm onSubmit={jest.fn()} loading={false} attempts={0} />);
    
    const inputs = screen.getAllByRole('textbox');
    expect(inputs).toHaveLength(5);
  });

  it('should focus first input on mount', () => {
    render(<PaymentForm onSubmit={jest.fn()} loading={false} attempts={0} />);
    
    const inputs = screen.getAllByRole('textbox');
    expect(inputs[0]).toHaveFocus();
  });

  it('should only accept numeric input', async () => {
    const user = userEvent.setup();
    render(<PaymentForm onSubmit={jest.fn()} loading={false} attempts={0} />);
    
    const inputs = screen.getAllByRole('textbox');
    
    await user.type(inputs[0], 'a');
    expect(inputs[0]).toHaveValue('');
    
    await user.type(inputs[0], '5');
    expect(inputs[0]).toHaveValue('5');
  });

  it('should auto-focus next input after entering digit', async () => {
    const user = userEvent.setup();
    render(<PaymentForm onSubmit={jest.fn()} loading={false} attempts={0} />);
    
    const inputs = screen.getAllByRole('textbox');
    
    await user.type(inputs[0], '1');
    expect(inputs[1]).toHaveFocus();
    
    await user.type(inputs[1], '2');
    expect(inputs[2]).toHaveFocus();
  });

  it('should move focus back on backspace when input is empty', async () => {
    const user = userEvent.setup();
    render(<PaymentForm onSubmit={jest.fn()} loading={false} attempts={0} />);
    
    const inputs = screen.getAllByRole('textbox');
    
    await user.type(inputs[0], '1');
    await user.type(inputs[1], '2');
    
    // Clear second input and press backspace
    await user.clear(inputs[1]);
    await user.type(inputs[1], '{backspace}');
    
    expect(inputs[0]).toHaveFocus();
  });

  it('should call onSubmit with complete code', async () => {
    const onSubmit = jest.fn();
    const user = userEvent.setup();
    render(<PaymentForm onSubmit={onSubmit} loading={false} attempts={0} />);
    
    const inputs = screen.getAllByRole('textbox');
    
    await user.type(inputs[0], '1');
    await user.type(inputs[1], '2');
    await user.type(inputs[2], '3');
    await user.type(inputs[3], '4');
    await user.type(inputs[4], '5');
    
    const submitButton = screen.getByRole('button', { name: /Pay Now/i });
    await user.click(submitButton);
    
    expect(onSubmit).toHaveBeenCalledWith('12345');
  });

  it('should disable submit button when code is incomplete', () => {
    render(<PaymentForm onSubmit={jest.fn()} loading={false} attempts={0} />);
    
    const submitButton = screen.getByRole('button', { name: /Pay Now/i });
    expect(submitButton).toBeDisabled();
  });

  it('should show loading state', () => {
    render(<PaymentForm onSubmit={jest.fn()} loading={true} attempts={0} />);
    
    expect(screen.getByText(/Processing Payment/i)).toBeInTheDocument();
  });

  it('should show failed attempts warning', () => {
    render(<PaymentForm onSubmit={jest.fn()} loading={false} attempts={1} />);
    
    expect(screen.getByText(/Payment attempt 1\/3 failed/i)).toBeInTheDocument();
    expect(screen.getByText(/2 attempts remaining/i)).toBeInTheDocument();
  });

  it('should show last attempt warning', () => {
    render(<PaymentForm onSubmit={jest.fn()} loading={false} attempts={2} />);
    
    expect(screen.getByText(/Payment attempt 2\/3 failed/i)).toBeInTheDocument();
    expect(screen.getByText(/1 attempt remaining/i)).toBeInTheDocument();
  });

  it('should handle paste event', async () => {
    const user = userEvent.setup();
    render(<PaymentForm onSubmit={jest.fn()} loading={false} attempts={0} />);
    
    const inputs = screen.getAllByRole('textbox');
    
    // Simulate paste by focusing and pasting
    inputs[0].focus();
    await user.paste('12345');
    
    expect(inputs[0]).toHaveValue('1');
    expect(inputs[1]).toHaveValue('2');
    expect(inputs[2]).toHaveValue('3');
    expect(inputs[3]).toHaveValue('4');
    expect(inputs[4]).toHaveValue('5');
  });

  it('should disable inputs when loading', () => {
    render(<PaymentForm onSubmit={jest.fn()} loading={true} attempts={0} />);
    
    const inputs = screen.getAllByRole('textbox');
    inputs.forEach(input => {
      expect(input).toBeDisabled();
    });
  });

  it('should clear inputs on new attempt', () => {
    const { rerender } = render(<PaymentForm onSubmit={jest.fn()} loading={false} attempts={0} />);
    
    const inputs = screen.getAllByRole('textbox');
    fireEvent.change(inputs[0], { target: { value: '1' } });
    fireEvent.change(inputs[1], { target: { value: '2' } });
    
    // Simulate retry with new attempt count
    rerender(<PaymentForm onSubmit={jest.fn()} loading={false} attempts={1} />);
    
    const newInputs = screen.getAllByRole('textbox');
    newInputs.forEach(input => {
      expect(input).toHaveValue('');
    });
  });
});

