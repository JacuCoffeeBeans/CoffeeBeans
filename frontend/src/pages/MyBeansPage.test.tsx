import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import { MantineProvider } from '@mantine/core';
import { ModalsProvider } from '@mantine/modals';
import { Notifications } from '@mantine/notifications';
import { vi } from 'vitest';
import type { Session } from '@supabase/supabase-js';
import { AuthProvider } from '../contexts/AuthContext';
import MyBeansPage from './MyBeansPage';
import { supabase } from '../lib/supabaseClient';

// supabaseClientのモック
vi.mock('../lib/supabaseClient');

// fetchのモック
window.fetch = vi.fn();

const renderWithProviders = (ui: React.ReactElement) => {
  return render(
    <BrowserRouter>
      <MantineProvider>
        <AuthProvider>
          <ModalsProvider>
            <Notifications />
            {ui}
          </ModalsProvider>
        </AuthProvider>
      </MantineProvider>
    </BrowserRouter>
  );
};

const mockSession: Session = {
  access_token: 'test-token',
  refresh_token: 'test-refresh-token',
  expires_in: 3600,
  token_type: 'bearer',
  user: {
    id: 'test-user-id',
    app_metadata: {},
    user_metadata: {},
    aud: 'authenticated',
    created_at: new Date().toISOString(),
  },
};

describe('MyBeansPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // onAuthStateChangeのモックを追加
    vi.mocked(supabase.auth.onAuthStateChange).mockReturnValue({
      data: {
        subscription: {
          id: 'test-subscription',
          callback: vi.fn(),
          unsubscribe: vi.fn(),
        },
      },
    });
  });

  test('ログイン状態で、登録した豆がない場合にメッセージが表示される', async () => {
    // ログイン状態をモック
    vi.mocked(supabase.auth.getSession).mockResolvedValueOnce({
      data: { session: mockSession },
      error: null,
    });
    // APIレスポンスをモック（空の配列）
    vi.mocked(fetch).mockResolvedValueOnce(
      new Response(JSON.stringify([]), { status: 200 })
    );

    renderWithProviders(<MyBeansPage />);

    // APIが呼ばれるのを待つ
    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith('/api/my/beans', {
        headers: {
          Authorization: `Bearer ${mockSession.access_token}`,
        },
      });
    });

    // ローディングが消え、メッセージが表示されることを確認
    expect(
      screen.getByText('マイページ（出品した豆一覧）')
    ).toBeInTheDocument();
    expect(
      screen.getByText('登録されている豆はありません。')
    ).toBeInTheDocument();
    expect(
      screen.getByRole('link', { name: '一覧に戻る' })
    ).toBeInTheDocument();
  });

  test('ログイン状態で、登録した豆がある場合に一覧表示される', async () => {
    const mockBeans = [
      { id: 1, name: 'My Coffee 1' },
      { id: 2, name: 'My Coffee 2' },
    ];
    // ログイン状態をモック
    vi.mocked(supabase.auth.getSession).mockResolvedValueOnce({
      data: { session: mockSession },
      error: null,
    });
    // APIレスポンスをモック
    vi.mocked(fetch).mockResolvedValueOnce(
      new Response(JSON.stringify(mockBeans), { status: 200 })
    );

    renderWithProviders(<MyBeansPage />);

    // データが表示されるのを待つ
    await waitFor(() => {
      expect(screen.getByText('My Coffee 1')).toBeInTheDocument();
      expect(screen.getByText('My Coffee 2')).toBeInTheDocument();
    });

    // 編集・削除ボタンが表示されていることを確認
    expect(screen.getAllByRole('link', { name: '編集' })).toHaveLength(2);
    expect(screen.getAllByRole('button', { name: '削除' })).toHaveLength(2);
    expect(
      screen.getByRole('link', { name: '一覧に戻る' })
    ).toBeInTheDocument();
  });

  test('豆の削除処理が成功し、リストから削除される', async () => {
    const user = userEvent.setup();
    const mockBeans = [
      { id: 1, name: 'My Coffee 1' },
      { id: 2, name: 'My Coffee 2' },
    ];
    // ログイン状態をモック
    vi.mocked(supabase.auth.getSession).mockResolvedValueOnce({
      data: { session: mockSession },
      error: null,
    });
    // GET APIレスポンスをモック
    vi.mocked(fetch).mockResolvedValueOnce(
      new Response(JSON.stringify(mockBeans), { status: 200 })
    );
    // DELETE APIレスポンスをモック
    vi.mocked(fetch).mockResolvedValueOnce(
      new Response(null, { status: 204 })
    );

    renderWithProviders(<MyBeansPage />);

    // 最初に豆が2つ表示されていることを確認
    await waitFor(() => {
      expect(screen.getByText('My Coffee 1')).toBeInTheDocument();
    });
    expect(screen.getByText('My Coffee 2')).toBeInTheDocument();

    // 1つ目の豆の削除ボタンをクリック
    const deleteButtons = screen.getAllByRole('button', { name: '削除' });
    await user.click(deleteButtons[0]);

    // モーダルが表示されるのを待つ
    const modal = await screen.findByRole('dialog');

    // モーダル内にテキストが表示されることを確認
    expect(
      within(modal).getByText(/本当に「My Coffee 1」を削除しますか？/)
    ).toBeInTheDocument();

    // モーダルの削除ボタンをクリック
    const confirmButton = within(modal).getByRole('button', { name: 'はい、削除します' });
    await user.click(confirmButton);

    // DELETEリクエストが呼ばれたことを確認
    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith('/api/beans/1', {
        method: 'DELETE',
        headers: {
          Authorization: `Bearer ${mockSession.access_token}`,
        },
      });
    });

    // リストから豆が削除されていることを確認
    await waitFor(() => {
      expect(screen.queryByText('My Coffee 1')).not.toBeInTheDocument();
    });
    expect(screen.getByText('My Coffee 2')).toBeInTheDocument();

    // 成功通知が表示されることを確認
    expect(
      await screen.findByText('コーヒー豆の情報を削除しました。')
    ).toBeInTheDocument();
  });

  test('豆の削除処理が失敗し、エラー通知が表示される', async () => {
    const user = userEvent.setup();
    const mockBeans = [{ id: 1, name: 'My Coffee 1' }];
    // ログイン状態をモック
    vi.mocked(supabase.auth.getSession).mockResolvedValueOnce({
      data: { session: mockSession },
      error: null,
    });
    // GET APIレスポンスをモック
    vi.mocked(fetch).mockResolvedValueOnce(
      new Response(JSON.stringify(mockBeans), { status: 200 })
    );
    // DELETE APIレスポンスをモック（失敗）
    vi.mocked(fetch).mockResolvedValueOnce(
      new Response(JSON.stringify({ message: 'Internal Server Error' }), {
        status: 500,
      })
    );

    renderWithProviders(<MyBeansPage />);

    // 豆が表示されていることを確認
    await waitFor(() => {
      expect(screen.getByText('My Coffee 1')).toBeInTheDocument();
    });

    // 削除ボタンをクリッ��
    await user.click(screen.getByRole('button', { name: '削除' }));

    // モーダルが表示されるのを待つ
    const modal = await screen.findByRole('dialog');
    await user.click(within(modal).getByRole('button', { name: 'はい、削除します' }));

    // エラー通知が表示されることを確認
    expect(await screen.findByText('エラー')).toBeInTheDocument();
    expect(await screen.findByText('削除に失敗しました。')).toBeInTheDocument();

    // リストから豆が削除されていないことを確認
    expect(screen.getByText('My Coffee 1')).toBeInTheDocument();
  });
});

