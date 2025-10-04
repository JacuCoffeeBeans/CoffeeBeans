import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { MantineProvider } from '@mantine/core';
import { vi } from 'vitest';
import CheckoutSuccessPage from './CheckoutSuccessPage';

// useNavigateのモック
const mockNavigate = vi.fn();
vi.mock('react-router-dom', async () => {
  const original = await vi.importActual('react-router-dom');
  return {
    ...original,
    useNavigate: () => mockNavigate,
  };
});

// Stripeのモック
const mockRetrievePaymentIntent = vi.fn();
vi.mock('@stripe/react-stripe-js', () => ({
  useStripe: () => ({
    retrievePaymentIntent: mockRetrievePaymentIntent,
  }),
}));

const renderComponent = (initialEntry: string) => {
  return render(
    <MantineProvider>
      <MemoryRouter initialEntries={[initialEntry]}>
        <Routes>
          <Route path="/checkout/success" element={<CheckoutSuccessPage />} />
        </Routes>
      </MemoryRouter>
    </MantineProvider>
  );
};

describe('CheckoutSuccessPage', () => {
  beforeEach(() => {
    mockRetrievePaymentIntent.mockClear();
    mockNavigate.mockClear();
  });

  test('client_secretがない場合、トップページにリダイレクトする', () => {
    renderComponent('/checkout/success');
    expect(mockNavigate).toHaveBeenCalledWith('/', { replace: true });
  });

  test('決済が成功した場合(succeeded)、成功メッセージを表示する', async () => {
    mockRetrievePaymentIntent.mockResolvedValue({
      paymentIntent: { status: 'succeeded' },
    });
    renderComponent('/checkout/success?payment_intent_client_secret=test_secret');

    await waitFor(() => {
      expect(screen.getByText('ご購入ありがとうございます。お支払いが正常に完了しました。')).toBeInTheDocument();
    });
    expect(screen.getByRole('link', { name: 'トップページに戻る' })).toBeInTheDocument();
  });

  test('決済が処理中の場合(processing)、処理中メッセージを表示する', async () => {
    mockRetrievePaymentIntent.mockResolvedValue({
      paymentIntent: { status: 'processing' },
    });
    renderComponent('/checkout/success?payment_intent_client_secret=test_secret');

    await waitFor(() => {
      expect(screen.getByText('決済処理中です。完了までしばらくお待ちください。')).toBeInTheDocument();
    });
    // ローダーが表示されていることを確認
    expect(document.querySelector('.mantine-Loader-root')).toBeInTheDocument();
  });

  test('決済が失敗した場合(requires_payment_method)、エラーメッセージを表示する', async () => {
    mockRetrievePaymentIntent.mockResolvedValue({
      paymentIntent: { status: 'requires_payment_method' },
    });
    renderComponent('/checkout/success?payment_intent_client_secret=test_secret');

    await waitFor(() => {
      expect(screen.getByText('お支払いが失敗しました。お支払い方法をご確認の上、再度お試しください。')).toBeInTheDocument();
    });
    expect(screen.getByRole('link', { name: 'カートに戻る' })).toBeInTheDocument();
  });

  test('予期せぬステータスの場合、汎用エラーメッセージを表示する', async () => {
    mockRetrievePaymentIntent.mockResolvedValue({ paymentIntent: { status: 'canceled' } });
    renderComponent('/checkout/success?payment_intent_client_secret=test_secret');

    await waitFor(() => {
      expect(screen.getByText('何らかの問題が発生しました。サポートにお問い合わせください。')).toBeInTheDocument();
    });
  });
});
