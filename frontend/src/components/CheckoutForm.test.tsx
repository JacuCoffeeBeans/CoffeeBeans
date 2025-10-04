import { render, screen, fireEvent, waitFor, within } from '@testing-library/react';
import { MantineProvider } from '@mantine/core';
import CheckoutForm from './CheckoutForm';
import { vi } from 'vitest';
import { MemoryRouter } from 'react-router-dom';

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
const mockConfirmPayment = vi.fn();
const mockGetElement = vi.fn(() => document.createElement('div'));
vi.mock('@stripe/react-stripe-js', () => ({
  PaymentElement: () => <div>Payment Element</div>,
  useStripe: () => ({
    confirmPayment: mockConfirmPayment,
  }),
  useElements: () => ({
    getElement: mockGetElement,
  }),
}));

const mockCartItems = [
  { id: '1', bean_id: 1, name: 'Test Bean 1', price: 1000, quantity: 2 },
  { id: '2', bean_id: 2, name: 'Test Bean 2', price: 500, quantity: 1 },
];
const mockTotalPrice = 2500;

describe('CheckoutForm', () => {
  beforeEach(() => {
    mockConfirmPayment.mockClear();
    mockNavigate.mockClear();
    globalThis.fetch = vi.fn();
  });

  const renderComponent = () => {
    return render(
      <MantineProvider>
        <MemoryRouter>
          <CheckoutForm cartItems={mockCartItems} totalPrice={mockTotalPrice} />
        </MemoryRouter>
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

  test('決済成功時に完了ページに遷移する', async () => {
    mockConfirmPayment.mockResolvedValue({
      paymentIntent: { status: 'succeeded', client_secret: 'test_secret' },
    });
    renderComponent();

    // フォームの必須項目を埋める
    fireEvent.change(screen.getByPlaceholderText('焙煎 太郎'), { target: { value: 'テストユーザー' } });
    fireEvent.change(screen.getByPlaceholderText('123-4567'), { target: { value: '123-4567' } });
    fireEvent.change(screen.getByPlaceholderText('〇〇県'), { target: { value: '東京都' } });
    fireEvent.change(screen.getByPlaceholderText('〇〇市'), { target: { value: '千代田区' } });
    fireEvent.change(screen.getByPlaceholderText('1-1'), { target: { value: '丸の内1-1' } });
    fireEvent.change(screen.getByPlaceholderText('000-1111-2222'), { target: { value: '09012345678' } });
    fireEvent.change(screen.getByPlaceholderText('TARO BAISEN'), { target: { value: 'TEST TARO' } });

    const payButtonInForm = screen.getByRole('button', { name: '支払う' });
    fireEvent.click(payButtonInForm);

    // モーダルが開くのを待つ
    const modal = await screen.findByRole('dialog');
    expect(modal).toBeInTheDocument();

    // モーダル内の支払うボタンをクリック
    const payButtonInModal = within(modal).getByRole('button', { name: '支払う' });
    fireEvent.click(payButtonInModal);

    await waitFor(() => {
      expect(mockConfirmPayment).toHaveBeenCalled();
    });

    expect(mockNavigate).toHaveBeenCalledWith(
      '/checkout/success?payment_intent_client_secret=test_secret'
    );
  });

  test('決済失敗時にエラーメッセージを表示する', async () => {
    const errorMessage = 'Your card was declined.';
    mockConfirmPayment.mockResolvedValue({ error: { type: 'card_error', message: errorMessage } });
    renderComponent();

    // フォームの必須項目を埋める
    fireEvent.change(screen.getByPlaceholderText('焙煎 太郎'), { target: { value: 'テストユーザー' } });
    fireEvent.change(screen.getByPlaceholderText('123-4567'), { target: { value: '123-4567' } });
    fireEvent.change(screen.getByPlaceholderText('〇〇県'), { target: { value: '東京都' } });
    fireEvent.change(screen.getByPlaceholderText('〇〇市'), { target: { value: '千代田区' } });
    fireEvent.change(screen.getByPlaceholderText('1-1'), { target: { value: '丸の内1-1' } });
    fireEvent.change(screen.getByPlaceholderText('000-1111-2222'), { target: { value: '09012345678' } });
    fireEvent.change(screen.getByPlaceholderText('TARO BAISEN'), { target: { value: 'TEST TARO' } });

    const payButtonInForm = screen.getByRole('button', { name: '支払う' });
    fireEvent.click(payButtonInForm);

    // モーダルが開くのを待つ
    const modal = await screen.findByRole('dialog');
    expect(modal).toBeInTheDocument();

    // モーダル内の支払うボタンをクリック
    const payButtonInModal = within(modal).getByRole('button', { name: '支払う' });
    fireEvent.click(payButtonInModal);

    await waitFor(() => {
      expect(screen.getByText(errorMessage)).toBeInTheDocument();
    });

    expect(mockNavigate).not.toHaveBeenCalled();
  });

  test('決済失敗時に予期せぬエラーが発生した場合、汎用エラーメッセージを表示する', async () => {
    const errorMessage = 'An unexpected error occurred.';
    mockConfirmPayment.mockResolvedValue({ error: { type: 'api_error', message: errorMessage } });
    renderComponent();

    // フォームの必須項目を埋める
    fireEvent.change(screen.getByPlaceholderText('焙煎 太郎'), { target: { value: 'テストユーザー' } });
    fireEvent.change(screen.getByPlaceholderText('123-4567'), { target: { value: '123-4567' } });
    fireEvent.change(screen.getByPlaceholderText('〇〇県'), { target: { value: '東京都' } });
    fireEvent.change(screen.getByPlaceholderText('〇〇市'), { target: { value: '千代田区' } });
    fireEvent.change(screen.getByPlaceholderText('1-1'), { target: { value: '丸の内1-1' } });
    fireEvent.change(screen.getByPlaceholderText('000-1111-2222'), { target: { value: '09012345678' } });
    fireEvent.change(screen.getByPlaceholderText('TARO BAISEN'), { target: { value: 'TEST TARO' } });

    const payButtonInForm = screen.getByRole('button', { name: '支払う' });
    fireEvent.click(payButtonInForm);

    // モーダルが開くのを待つ
    const modal = await screen.findByRole('dialog');
    expect(modal).toBeInTheDocument();

    // モーダル内の支払うボタンをクリック
    const payButtonInModal = within(modal).getByRole('button', { name: '支払う' });
    fireEvent.click(payButtonInModal);

    await waitFor(() => {
      expect(screen.getByText('予期せぬエラーが発生しました。別のカードをお試しください。')).toBeInTheDocument();
    });

    expect(mockNavigate).not.toHaveBeenCalled();
  });

  test('郵便番号入力時に自動でハイフンが挿入される', async () => {
    globalThis.fetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ status: 400, message: 'not found' }),
    });
    renderComponent();
    const postalCodeInput = screen.getByPlaceholderText('123-4567');

    fireEvent.change(postalCodeInput, { target: { value: '123' } });
    expect(postalCodeInput.value).toBe('123-');

    fireEvent.change(postalCodeInput, { target: { value: '123-4567' } });
    expect(postalCodeInput.value).toBe('123-4567');

    await waitFor(() => {
      expect(globalThis.fetch).toHaveBeenCalled();
    });
  });

  test('有効な郵便番号入力時に住所が自動入力される', async () => {
    globalThis.fetch.mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          status: 200,
          results: [
            {
              address1: '東京都',
              address2: '千代田区',
              address3: '丸の内',
            },
          ],
        }),
    });

    renderComponent();
    const postalCodeInput = screen.getByPlaceholderText('123-4567');
    const prefectureInput = screen.getByPlaceholderText('〇〇県');
    const cityInput = screen.getByPlaceholderText('〇〇市');

    fireEvent.change(postalCodeInput, { target: { value: '100-0005' } });

    await waitFor(() => {
      expect(globalThis.fetch).toHaveBeenCalledWith(
        'https://zipcloud.ibsnet.co.jp/api/search?zipcode=1000005'
      );
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