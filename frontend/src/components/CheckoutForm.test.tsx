import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MantineProvider } from '@mantine/core';
import CheckoutForm from './CheckoutForm';
import { vi } from 'vitest';

// Stripeのモック
const mockConfirmPayment = vi.fn();
vi.mock('@stripe/react-stripe-js', () => ({
  PaymentElement: () => <div>Payment Element</div>,
  useStripe: () => ({
    confirmPayment: mockConfirmPayment,
  }),
  useElements: () => ({}),
}));

const mockCartItems = [
  { id: '1', bean_id: 1, name: 'Test Bean 1', price: 1000, quantity: 2 },
  { id: '2', bean_id: 2, name: 'Test Bean 2', price: 500, quantity: 1 },
];
const mockTotalPrice = 2500;

describe('CheckoutForm', () => {
  beforeEach(() => {
    mockConfirmPayment.mockClear();
    globalThis.fetch = vi.fn();
  });

  const renderComponent = () => {
    return render(
      <MantineProvider>
        <CheckoutForm cartItems={mockCartItems} totalPrice={mockTotalPrice} />
      </MantineProvider>
    );
  };

  test('注文内容とすべてのフォーム要素が正しく表示される', () => {
    renderComponent();

    // 注文内容
    expect(screen.getByText('ご注文内容')).toBeInTheDocument();
    expect(screen.getByText(/Test Bean 1/)).toBeInTheDocument();
    expect(screen.getByText(/Test Bean 2/)).toBeInTheDocument();
    expect(screen.getByText(/合計金額 : 2,500円/)).toBeInTheDocument();

    // 配送先情報（placeholderで取得）
    expect(screen.getByPlaceholderText('焙煎 太郎')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('123-4567')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('〇〇県')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('〇〇市')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('1-1')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('000-1111-2222')).toBeInTheDocument();

    // 支払い情報
    expect(screen.getByText('お支払い情報')).toBeInTheDocument();
    expect(screen.getByText('Payment Element')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('TARO BAISEN')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '支払う' })).toBeInTheDocument();
  });

  test('郵便番号入力時に自動でハイフンが挿入される', () => {
    renderComponent();
    const postalCodeInput = screen.getByPlaceholderText('123-4567');

    fireEvent.change(postalCodeInput, { target: { value: '123' } });
    expect(postalCodeInput.value).toBe('123-');

    fireEvent.change(postalCodeInput, { target: { value: '123-4567' } });
    expect(postalCodeInput.value).toBe('123-4567');
  });

  test('有効な郵便番号入力時に住所が自動入力される', async () => {
    globalThis.fetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({
        status: 200,
        results: [{
          address1: '東京都',
          address2: '千代田区',
          address3: '丸の内',
        }],
      }),
    });

    renderComponent();
    const postalCodeInput = screen.getByPlaceholderText('123-4567');
    const prefectureInput = screen.getByPlaceholderText('〇〇県');
    const cityInput = screen.getByPlaceholderText('〇〇市');

    fireEvent.change(postalCodeInput, { target: { value: '100-0005' } });

    await waitFor(() => {
      expect(globalThis.fetch).toHaveBeenCalledWith('https://zipcloud.ibsnet.co.jp/api/search?zipcode=1000005');
      expect(prefectureInput.value).toBe('東京都');
      expect(cityInput.value).toBe('千代田区丸の内');
    });
  });

  test('無効な郵便番号入力時に住所欄をクリアする', async () => {
    globalThis.fetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ status: 400, message: 'not found' }),
    });

    renderComponent();
    const postalCodeInput = screen.getByPlaceholderText('123-4567');
    const prefectureInput = screen.getByPlaceholderText('〇〇県');
    const cityInput = screen.getByPlaceholderText('〇〇市');

    // 事前に値を入れておく
    fireEvent.change(prefectureInput, { target: { value: 'ダミー県' } });
    fireEvent.change(cityInput, { target: { value: 'ダミー市' } });

    fireEvent.change(postalCodeInput, { target: { value: '000-0000' } });

    await waitFor(() => {
      expect(prefectureInput.value).toBe('');
      expect(cityInput.value).toBe('');
    });
  });

});
