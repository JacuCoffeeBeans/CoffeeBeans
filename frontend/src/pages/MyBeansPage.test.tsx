import { render, screen, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { MantineProvider } from '@mantine/core';
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
        <AuthProvider>{ui}</AuthProvider>
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
});
