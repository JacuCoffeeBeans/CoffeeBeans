import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { beforeAll, afterEach, afterAll, test, expect, vi } from 'vitest';
import App from './App';
import { MantineProvider } from '@mantine/core';
import { AuthProvider } from './contexts/AuthContext';
import { mockSupabase } from './setupTests';

// APIのモック設定
const mockBeans = [{ id: 1, name: 'モック・ブルーマウンテン' }];
beforeAll(() => {
  globalThis.fetch = vi.fn().mockResolvedValue({
    ok: true,
    json: () => Promise.resolve(mockBeans),
  });
});

afterEach(() => {
  vi.clearAllMocks();
  vi.mocked(mockSupabase.auth.getSession).mockResolvedValue({ data: { session: null }, error: null });
});

afterAll(() => vi.restoreAllMocks());

const renderWithProviders = (initialEntries: string[]) => {
  return render(
    <MemoryRouter initialEntries={initialEntries}>
      <MantineProvider>
        <AuthProvider>
          <App />
        </AuthProvider>
      </MantineProvider>
    </MemoryRouter>
  );
};

test('ルートパスにアクセスすると、コーヒー豆の一覧ページが表示される', async () => {
  renderWithProviders(['/']);

  const listTitle = await screen.findByText(/コーヒー豆リスト/i);
  const firstItem = await screen.findByText('モック・ブルーマウンテン');

  expect(listTitle).toBeInTheDocument();
  expect(firstItem).toBeInTheDocument();
});

test('未ログインで/beans/newにアクセスすると、ログインページにリダイレクトされる', async () => {
  renderWithProviders(['/beans/new']);

  // ログインページにリダイレクトされ、そのタイトルが表示されるのを待つ
  await waitFor(() => {
    expect(screen.getByText('Welcome back!')).toBeInTheDocument();
  });
});
