import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { MantineProvider } from '@mantine/core';
import { Notifications } from '@mantine/notifications';
import { ModalsProvider, modals } from '@mantine/modals';
import { AuthContext } from '../contexts/AuthContext';
import CartPage from './CartPage';
import { vi } from 'vitest';
import type { Session } from '@supabase/supabase-js';

// Mantineのmodalsをモックする
vi.mock('@mantine/modals', async () => {
    const actual = await vi.importActual('@mantine/modals');
    return {
        ...actual,
        modals: {
            ...actual.modals,
            openConfirmModal: vi.fn(),
        },
    };
});

// モックデータ
const mockCartItems = [
    { id: 'uuid-1', bean_id: 1, name: 'モカ', price: 1200, quantity: 2 },
    { id: 'uuid-2', bean_id: 2, name: 'キリマンジャロ', price: 1500, quantity: 1 },
];

const mockSession = {
  access_token: 'test-token',
  user: { id: 'test-user-id' },
} as unknown as Session;

// カスタムレンダー関数
const renderWithProviders = (session: Session | null) => {
  return render(
    <MantineProvider>
      <MemoryRouter initialEntries={['/cart']}>
        <AuthContext.Provider value={{ session, isLoading: false }}>
          <ModalsProvider>
            <Notifications />
            <Routes>
              <Route path="/cart" element={<CartPage />} />
              <Route path="/login" element={<div>Login Page</div>} />
            </Routes>
          </ModalsProvider>
        </AuthContext.Provider>
      </MemoryRouter>
    </MantineProvider>
  );
};

describe('CartPage', () => {
  beforeEach(() => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve([...mockCartItems]), // データをコピーして渡す
    });
    // 各テストの前にモックをクリア
    vi.clearAllMocks();
  });

  test('ログイン状態でカートアイテムが正しく表示される', async () => {
    renderWithProviders(mockSession);
    await waitFor(() => {
      expect(screen.getByText('モカ')).toBeInTheDocument();
      expect(screen.getByText('キリマンジャロ')).toBeInTheDocument();
    });
    const total = 1200 * 2 + 1500 * 1;
    expect(screen.getByText(`合計: ${total.toLocaleString()}円`)).toBeInTheDocument();
  });

  test('数量変更後、ページ離脱時に更新APIが呼ばれる', async () => {
    const { unmount } = renderWithProviders(mockSession);
    await waitFor(() => expect(globalThis.fetch).toHaveBeenCalledTimes(1)); // 初回読み込み

    const quantityInput = (await screen.findAllByRole('textbox'))[0];
    fireEvent.change(quantityInput, { target: { value: '5' } });

    // アンマウント（ページ離脱）
    unmount();

    // 更新APIが呼ばれたか確認
    await waitFor(() => {
        expect(globalThis.fetch).toHaveBeenCalledWith(`/api/cart/items/${mockCartItems[0].id}`,
        {
            method: 'PUT',
            headers: expect.any(Object),
            body: JSON.stringify({ quantity: 5 }),
        });
    });
    // 2回目のアイテムは変更ないので呼ばれない
    expect(globalThis.fetch).not.toHaveBeenCalledWith(`/api/cart/items/${mockCartItems[1].id}`, expect.any(Object));
  });

  test('削除ボタンクリックで確認モーダルが表示される', async () => {
    renderWithProviders(mockSession);
    await waitFor(() => expect(screen.getByText('モカ')).toBeInTheDocument());

    const deleteButtons = screen.getAllByRole('button', { name: /delete/i });
    fireEvent.click(deleteButtons[0]);

    expect(modals.openConfirmModal).toHaveBeenCalledWith(expect.objectContaining({
        title: '削除の確認',
    }));
  });
});
