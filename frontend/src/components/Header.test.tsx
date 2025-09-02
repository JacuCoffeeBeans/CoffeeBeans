import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { test, expect, vi, afterEach } from 'vitest';
import { MantineProvider } from '@mantine/core';
import { AuthProvider } from '../contexts/AuthContext';
import Header from './Header';
import { Session } from '@supabase/supabase-js';
import { supabase } from '../lib/supabaseClient';

afterEach(() => {
  vi.restoreAllMocks();
});

const renderWithProviders = (ui: React.ReactElement) => {
  return render(
    <MemoryRouter>
      <MantineProvider>
        <AuthProvider>{ui}</AuthProvider>
      </MantineProvider>
    </MemoryRouter>
  );
};

test('未ログイン時、ログインボタンが表示され、登録ボタンが非表示であること', async () => {
  vi.spyOn(supabase.auth, 'getSession').mockResolvedValue({ data: { session: null }, error: null });
  vi.spyOn(supabase.auth, 'onAuthStateChange').mockImplementation((callback) => {
    callback('INITIAL_SESSION', null);
    return {
      data: { subscription: { unsubscribe: vi.fn() } },
    };
  });

  renderWithProviders(<Header />);
  
  expect(screen.getByRole('button', { name: /ログイン/i })).toBeInTheDocument();
  expect(screen.queryByRole('button', { name: /新しい豆を登録/i })).not.toBeInTheDocument();
});

test('ログイン時、ユーザー情報とログアウト・登録ボタンが表示されること', async () => {
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
  
  renderWithProviders(<Header />);

  expect(await screen.findByText('test@example.com')).toBeInTheDocument();
  expect(screen.getByRole('button', { name: /ログアウト/i })).toBeInTheDocument();
  expect(screen.getByText(/マイページ/i)).toBeInTheDocument();
  expect(screen.getByText(/新しい豆を登録/i)).toBeInTheDocument();
});
