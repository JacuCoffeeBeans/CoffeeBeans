import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { test, expect, vi } from 'vitest';
import Login from './Login';
import { MantineProvider } from '@mantine/core';
import { supabase } from '../lib/supabaseClient';

// テスト用のレンダリング関数
const renderWithProviders = (ui: React.ReactElement) => {
  return render(
    <MemoryRouter>
      <MantineProvider>{ui}</MantineProvider>
    </MemoryRouter>
  );
};

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