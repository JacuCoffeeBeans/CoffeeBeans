import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { beforeAll, afterEach, afterAll, test, expect, vi } from 'vitest';
import BeanListPage from './BeanListPage';
import BeanDetailPage from './BeanDetailPage';
import { MantineProvider } from '@mantine/core';

// --- APIのモック設定を修正 ---
const mockBeanList = [{ id: 1, name: 'モック・ブルーマウンテン' }];
const mockBeanDetail = { id: 1, name: '詳細・ブルーマウンテン', origin: 'モック産地', price: 1200 };

beforeAll(() => {
  // fetchが呼ばれたURLに応じて、返すダミーデータを切り替える
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

afterEach(() => vi.clearAllMocks());
afterAll(() => vi.restoreAllMocks());

test('リストの項目をクリックすると、対応する詳細ページに遷移する', async () => {
  render(
    <MantineProvider>
      <MemoryRouter initialEntries={['/']}>
        <Routes>
          <Route path="/" element={<BeanListPage />} />
          <Route path="/beans/:beanId" element={<BeanDetailPage />} />
        </Routes>
      </MemoryRouter>
    </MantineProvider>
  );

  // 一覧ページのリスト項目が表示されるのを待つ
  const linkElement = await screen.findByText('モック・ブルーマウンテン');
  expect(linkElement).toBeInTheDocument();

  // リンクをクリック
  await userEvent.click(linkElement);

  // --- 検証部分を修正 ---
  // 遷移後の詳細ページで、詳細APIから取得したデータが表示されるのを待つ
  expect(await screen.findByText(mockBeanDetail.name)).toBeInTheDocument();
  expect(screen.getByText(`産地: ${mockBeanDetail.origin}`)).toBeInTheDocument();
});