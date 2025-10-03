import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { MantineProvider } from '@mantine/core';
import { AuthContext } from '../contexts/AuthContext';
import CheckoutPage from './CheckoutPage';
import { vi } from 'vitest';
import type { Session } from '@supabase/supabase-js';

// Stripeのモック
vi.mock('@stripe/react-stripe-js', () => ({
  Elements: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  PaymentElement: () => <div>Payment Element</div>,
  useStripe: () => ({}),
  useElements: () => ({}),
}));
vi.mock('@stripe/stripe-js', () => ({
  loadStripe: async () => ({}),
}));

const mockSession = {
  access_token: 'test-token',
  user: { id: 'test-user-id' },
} as unknown as Session;

const mockCartItems = [
  { id: '1', bean_id: 1, name: 'Test Bean', price: 1000, quantity: 2 },
];

// 成功時のデフォルトfetchモック
const successfulFetchMock = vi.fn().mockImplementation((url) => {
  if (url === '/api/cart') {
    return Promise.resolve({ ok: true, json: () => Promise.resolve(mockCartItems) });
  }
  if (url === '/api/checkout/payment-intent') {
    return Promise.resolve({ ok: true, json: () => Promise.resolve({ client_secret: 'test_secret' }) });
  }
  return Promise.reject(new Error(`Unhandled fetch url: ${url}`));
});

const renderWithProviders = (session: Session | null, cartItems: any[] = mockCartItems, fetchMock: any = successfulFetchMock) => {
  globalThis.fetch = fetchMock;

  return render(
    <MantineProvider>
      <MemoryRouter initialEntries={['/checkout']}>
        <AuthContext.Provider value={{ session, isLoading: false }}>
          <Routes>
            <Route path="/checkout" element={<CheckoutPage />} />
          </Routes>
        </AuthContext.Provider>
      </MemoryRouter>
    </MantineProvider>
  );
};

describe('CheckoutPage', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  test('カート情報と決済情報を取得し、CheckoutFormを表示する', async () => {
    renderWithProviders(mockSession);

    await waitFor(() => {
      expect(globalThis.fetch).toHaveBeenCalledWith('/api/cart', expect.any(Object));
      expect(globalThis.fetch).toHaveBeenCalledWith('/api/checkout/payment-intent', expect.any(Object));
    });

    expect(await screen.findByText('ご注文内容')).toBeInTheDocument();
    expect(await screen.findByText('配送先情報')).toBeInTheDocument();
  });

  test('カートが空の場合、エラーメッセージを表示する', async () => {
    const emptyCartFetchMock = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve([]) });
    renderWithProviders(mockSession, [], emptyCartFetchMock);

    expect(await screen.findByText('カートが空のため、決済に進めません。')).toBeInTheDocument();
  });

  test('clientSecretが取得できない場合、エラーメッセージを表示する', async () => {
    const failingFetchMock = vi.fn().mockImplementation((url) => {
      if (url === '/api/cart') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(mockCartItems) });
      }
      if (url === '/api/checkout/payment-intent') {
        return Promise.resolve({ ok: false, json: () => Promise.resolve({ message: 'Intent Error' }) });
      }
      return Promise.reject(new Error(`Unhandled fetch url: ${url}`));
    });

    renderWithProviders(mockSession, mockCartItems, failingFetchMock);
    
    const alert = await screen.findByRole('alert');
    expect(alert).toHaveTextContent('Intent Error');
  });
});