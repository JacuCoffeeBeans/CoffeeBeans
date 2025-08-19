import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { test, expect, vi, beforeEach, afterEach } from 'vitest';
import Login from './Login';
import { MantineProvider } from '@mantine/core';
import { supabase } from '../lib/supabaseClient';
import { AuthProvider } from '../contexts/AuthContext';
import { Session } from '@supabase/supabase-js';

const mockNavigate = vi.fn();
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

beforeEach(() => {
  vi.mocked(supabase.auth.getSession).mockResolvedValue({ data: { session: null }, error: null });
  vi.mocked(supabase.auth.onAuthStateChange).mockImplementation((callback) => {
    callback('INITIAL_SESSION', null);
    return {
      data: { subscription: { unsubscribe: vi.fn() } },
    };
  });
});

afterEach(() => {
    vi.clearAllMocks();
});


// テスト用のレンダリング関数
const renderWithProviders = (ui: React.ReactElement) => {
  return render(
    <MantineProvider>
      <AuthProvider>
        <MemoryRouter>
          {ui}
        </MemoryRouter>
      </AuthProvider>
    </MantineProvider>
  );
};

test('未ログインのユーザーにはログインフォームが表示される', async () => {
  renderWithProviders(<Login />);
  expect(screen.getByLabelText('Email')).toBeInTheDocument();
  expect(screen.getByRole('button', { name: 'ログインリンクを送る' })).toBeInTheDocument();
  expect(screen.getByRole('button', { name: 'Googleでログイン' })).toBeInTheDocument();
});

test('ログイン済みのユーザーはホームページにリダイレクトされる', async () => {
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
  vi.mocked(supabase.auth.getSession).mockResolvedValue({ data: { session: mockSession }, error: null });
  vi.mocked(supabase.auth.onAuthStateChange).mockImplementation((callback) => {
    callback('SIGNED_IN', mockSession);
    return {
      data: { subscription: { unsubscribe: vi.fn() } },
    };
  });

  renderWithProviders(<Login />);

  await waitFor(() => {
    expect(mockNavigate).toHaveBeenCalledWith('/', { replace: true });
  });
});

test('無効なメールアドレスを入力して送信するとエラーメッセージが表示される', async () => {
  renderWithProviders(<Login />);
  const user = userEvent.setup();

  const emailInput = screen.getByLabelText('Email');
  await user.type(emailInput, 'invalid-email');

  const submitButton = screen.getByRole('button', { name: 'ログインリンクを送る' });
  await user.click(submitButton);

  expect(await screen.findByText('有効なメールアドレスを入力してください。')).toBeInTheDocument();
});

test('有効なメールアドレスを入力して送信するとsignInWithOtpが呼ばれる', async () => {
  renderWithProviders(<Login />);
  const user = userEvent.setup();

  const emailInput = screen.getByLabelText('Email');
  await user.type(emailInput, 'test@example.com');

  const submitButton = screen.getByRole('button', { name: 'ログインリンクを送る' });
  await user.click(submitButton);

  await waitFor(() => {
    expect(supabase.auth.signInWithOtp).toHaveBeenCalledWith({
      email: 'test@example.com',
      options: {
        emailRedirectTo: window.location.origin,
      },
    });
  });
});

test('GoogleログインボタンをクリックするとsignInWithOAuthが呼ばれる', async () => {
  renderWithProviders(<Login />);
  const user = userEvent.setup();

  const googleButton = screen.getByRole('button', { name: 'Googleでログイン' });
  await user.click(googleButton);

  expect(supabase.auth.signInWithOAuth).toHaveBeenCalledWith({
    provider: 'google',
    options: {
      redirectTo: window.location.origin,
    },
  });
});