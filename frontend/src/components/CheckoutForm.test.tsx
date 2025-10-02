import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { MantineProvider } from '@mantine/core';
import CheckoutForm from './CheckoutForm';
import { vi } from 'vitest';

// Stripeのモック
vi.mock('@stripe/react-stripe-js', () => ({
  PaymentElement: () => <div>Payment Element</div>,
  useStripe: () => ({}),
  useElements: () => ({}),
}));

describe('CheckoutForm', () => {
  test('決済フォームの要素が正しく表示される', () => {
    render(
      <MantineProvider>
        <MemoryRouter>
          <CheckoutForm />
        </MemoryRouter>
      </MantineProvider>
    );

    // 各要素が表示されているか確認
    expect(screen.getByText('Payment Element')).toBeInTheDocument();
    expect(screen.getByText('カード名義')).toBeInTheDocument(); // ラベルのテキストで存在確認
    expect(screen.getByPlaceholderText('TARO BAISEN')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '支払う' })).toBeInTheDocument();
  });
});
