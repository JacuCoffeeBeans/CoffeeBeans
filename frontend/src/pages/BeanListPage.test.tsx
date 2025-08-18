import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { beforeAll, afterEach, afterAll, test, expect, vi } from 'vitest';
import BeanListPage from './BeanListPage';
import BeanDetailPage from './BeanDetailPage';
import { MantineProvider } from '@mantine/core';
import { AuthProvider } from '../contexts/AuthContext';
import { Session } from '@supabase/supabase-js';
import { supabase } from '../lib/supabaseClient';

// --- APIのモック設定 ---
const mockBeanList = [{ id: 1, name: 'モック・ブルーマウンテン' }];
const mockBeanDetail = { id: 1, name: '詳細・ブルーマウンテン', origin: 'モック産地', price: 1200 };

beforeAll(() => {
  globalThis.fetch = vi.fn((url) => {
    let body;
    if (url === '/api/beans') {
      body = JSON.stringify(mockBeanList);
    } else if (url.toString().startsWith('/api/beans/')) {
      body = JSON.stringify(mockBeanDetail);
    }
    return Promise.resolve(new Response(body, {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    }));
  });
});



// テスト用のレンダリング関数
const renderWithProviders = (ui: React.ReactElement, initialEntries = ['/']) => {
  return render(
    <MantineProvider>
      <AuthProvider>
        <MemoryRouter initialEntries={initialEntries}>
          <Routes>
            <Route path="/" element={ui} />
            <Route path="/beans/:beanId" element={<BeanDetailPage />} />
          </Routes>
        </MemoryRouter>
      </AuthProvider>
    </MantineProvider>
  );
};

test('未ログイン時、ログインボタンが表示され、登録ボタンが非表示であること', async () => {
  renderWithProviders(<BeanListPage />);
  // ローディングが終わるのを待つ
  await screen.findByText('モック・ブルーマウンテン');
  
  expect(screen.getByRole('button', { name: /ログイン/i })).toBeInTheDocument();
  expect(screen.queryByRole('button', { name: /新しい豆を登録/i })).not.toBeInTheDocument();
});

test('ログイン時、ユーザー情報とログアウト・登録ボタンが表示されること', async () => {
  // ログイン状態をモック
  const mockSession: Session = {
    access_token: 'mock-access-token',
    refresh_token: 'mock-refresh-token',
    expires_in: 3600,
    token_type: 'bearer',
    user: {
      id: '1',
      app_metadata: {},
      user_metadata: {},
      aud: 'authenticated',
      email: 'test@example.com',
      created_at: new Date().toISOString(),
    },
  };
  vi.spyOn(supabase.auth, 'getSession').mockResolvedValue({ data: { session: mockSession }, error: null });
  vi.spyOn(supabase.auth, 'onAuthStateChange').mockImplementation((callback) => {
    callback('SIGNED_IN', mockSession);
    return {
      data: { subscription: { unsubscribe: vi.fn() } },
    };
  });
  
  renderWithProviders(<BeanListPage />);

  expect(await screen.findByText('test@example.com')).toBeInTheDocument();
  expect(screen.getByRole('button', { name: /ログアウト/i })).toBeInTheDocument();
  expect(screen.getByText(/新しい豆を登録/i)).toBeInTheDocument();
});

test('リストの項目をクリックすると、対応する詳細ページに遷移する', async () => {
  renderWithProviders(<BeanListPage />);
  const user = userEvent.setup();

  const linkElement = await screen.findByText('モック・ブルーマウンテン');
  await user.click(linkElement);

  expect(await screen.findByText(mockBeanDetail.name)).toBeInTheDocument();
  expect(screen.getByText(`産地: ${mockBeanDetail.origin}`)).toBeInTheDocument();
});
