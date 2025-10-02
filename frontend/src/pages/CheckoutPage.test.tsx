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
  PaymentElement: () => <div>Payment Element</div>, // 追加
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

const renderWithProviders = (session: Session | null) => {
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
  beforeEach(() => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ client_secret: 'test_secret' }),
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  test('clientSecretを取得し、CheckoutFormを表示する', async () => {
    renderWithProviders(mockSession);

    // APIが呼ばれるのを待つ
    await waitFor(() => {
      expect(globalThis.fetch).toHaveBeenCalledWith('/api/checkout/payment-intent', expect.any(Object));
    });

    // CheckoutForm内の要素が表示されているかで判断
    expect(await screen.findByText('支払う')).toBeInTheDocument();
  });

  test('clientSecretが取得できない場合、ローダーが表示され続ける', () => {
    // fetchが解決しないPromiseを返すようにモック
    globalThis.fetch = vi.fn(() => new Promise(() => {}));
    renderWithProviders(mockSession);

    // MantineのLoaderは明示的なroleを持たないため、DOM構造から存在を確認
    const loader = document.querySelector('.mantine-Loader-root');
    expect(loader).toBeInTheDocument();
  });
});
