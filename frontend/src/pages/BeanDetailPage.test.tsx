import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { beforeAll, afterEach, afterAll, test, expect, vi } from 'vitest';
import BeanDetailPage from './BeanDetailPage';
import { MantineProvider } from '@mantine/core';
import { Notifications } from '@mantine/notifications';
import { AuthProvider, AuthContext } from '../contexts/AuthContext';
import type { Session } from '@supabase/supabase-js';

// useNavigateのモック
const mockedNavigate = vi.fn();
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useNavigate: () => mockedNavigate,
  };
});

// APIのモック設定
const mockBeanDetail = {
  id: 1,
  name: 'モック・エチオピア',
  origin: 'モック産地',
  price: 1500,
  process: 'washed',
  roast_profile: 'medium',
};

// 正常なレスポンスを返すfetchモック
const mockFetchSuccess = vi.fn().mockResolvedValue({
  ok: true,
  json: () => Promise.resolve(mockBeanDetail),
});

// 認証済みセッションのモック
const mockSession = {
  access_token: 'test-token',
  user: { id: 'test-user-id' },
} as unknown as Session;

// テスト用のカスタムレンダー関数
const renderWithProviders = (
  ui: React.ReactElement,
  { session = null, initialEntries = ['/beans/1'] }: { session?: Session | null, initialEntries?: string[] } = {}
) => {
  return render(
    <MantineProvider>
      <MemoryRouter initialEntries={initialEntries}>
        <AuthProvider>
          <AuthContext.Provider value={{ session, isLoading: false }}>
            <Notifications />
            <Routes>
              <Route path="/beans/:beanId" element={ui} />
              <Route path="/login" element={<div>Login Page</div>} />
            </Routes>
          </AuthContext.Provider>
        </AuthProvider>
      </MemoryRouter>
    </MantineProvider>
  );
};

beforeAll(() => {
  globalThis.fetch = mockFetchSuccess;
});
afterEach(() => {
  vi.clearAllMocks();
  // 各テストの後にfetchをリセット
  globalThis.fetch = mockFetchSuccess;
});
afterAll(() => {
  vi.restoreAllMocks();
});

test('詳細データが正しく表示される', async () => {
  renderWithProviders(<BeanDetailPage />);

  expect(await screen.findByText(mockBeanDetail.name)).toBeInTheDocument();
  expect(screen.getByText(`産地: ${mockBeanDetail.origin}`)).toBeInTheDocument();
  expect(screen.getByText(`${mockBeanDetail.price}円`)).toBeInTheDocument();
  expect(mockFetchSuccess).toHaveBeenCalledWith('/api/beans/1');
});

test('未ログインで「カートに追加」をクリックするとログインページに遷移する', async () => {
  renderWithProviders(<BeanDetailPage />, { session: null });

  // カートに追加ボタンが表示されるのを待つ
  const addToCartButton = await screen.findByRole('button', { name: /カートに追加/i });
  fireEvent.click(addToCartButton);

  // navigateが/loginで呼ばれたことを確認
  expect(mockedNavigate).toHaveBeenCalledWith('/login');
});

test('ログイン済みで「カートに追加」をクリックするとAPIが呼ばれる', async () => {
  renderWithProviders(<BeanDetailPage />, { session: mockSession });

  const addToCartButton = await screen.findByRole('button', { name: /カートに追加/i });
  const quantityInput = screen.getByLabelText('数量');

  // 数量を2に変更
  fireEvent.change(quantityInput, { target: { value: 2 } });

  // APIモックをカート追加用に上書き
  globalThis.fetch = vi.fn().mockResolvedValue({
    ok: true,
    json: () => Promise.resolve({ message: '成功' }),
  });

  fireEvent.click(addToCartButton);

  // fetchが正しいエンドポイントとパラメータで呼ばれたか確認
  await waitFor(() => {
    expect(globalThis.fetch).toHaveBeenCalledWith('/api/cart/items', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${mockSession.access_token}`,
      },
      body: JSON.stringify({
        bean_id: mockBeanDetail.id,
        quantity: 2,
      }),
    });
  });

  // 成功通知が表示されることを確認
  expect(await screen.findByText(`${mockBeanDetail.name}をカートに追加しました。`)).toBeInTheDocument();
});

test('カート追加APIが失敗した場合、エラー通知が表示される', async () => {
  renderWithProviders(<BeanDetailPage />, { session: mockSession });

  const addToCartButton = await screen.findByRole('button', { name: /カートに追加/i });

  // APIモックをエラーを返すように上書き
  globalThis.fetch = vi.fn().mockResolvedValue({
    ok: false,
    json: () => Promise.resolve({ message: '在庫がありません' }),
  });

  fireEvent.click(addToCartButton);

  // エラー通知が表示されることを確認
  expect(await screen.findByText('在庫がありません')).toBeInTheDocument();
});
