import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { beforeAll, test, expect, vi } from 'vitest';
import BeanListPage from './BeanListPage';
import BeanDetailPage from './BeanDetailPage';
import { MantineProvider } from '@mantine/core';
import { AuthProvider } from '../contexts/AuthContext';

// --- APIのモック設定 ---
const mockBeanList = [{ id: 1, name: 'モック・ブルーマウンテン' }];
const mockBeanDetail = { id: 1, name: '詳細・ブルーマウンテン', origin: 'モック産地', price: 1200 };

beforeAll(() => {
  globalThis.fetch = vi.fn((url) => {
    let body;
    if (url.toString().endsWith('/api/beans')) {
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

test('リストの項目をクリックすると、対応する詳細ページに遷移する', async () => {
  renderWithProviders(<BeanListPage />);
  const user = userEvent.setup();

  const linkElement = await screen.findByText('モック・ブルーマウンテン');
  await user.click(linkElement);

  expect(await screen.findByText(mockBeanDetail.name)).toBeInTheDocument();
  expect(screen.getByText(`産地: ${mockBeanDetail.origin}`)).toBeInTheDocument();
});