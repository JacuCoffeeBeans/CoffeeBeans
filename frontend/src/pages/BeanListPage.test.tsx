import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { beforeAll, afterEach, afterAll, test, expect, vi } from 'vitest';
import BeanListPage from './BeanListPage';
import BeanDetailPage from './BeanDetailPage';
import { MantineProvider } from '@mantine/core';

// APIのモック設定
const mockBeans = [{ id: 1, name: 'モック・ブルーマウンテン' }];
beforeAll(() => {
  globalThis.fetch = vi.fn().mockResolvedValue({
    ok: true,
    json: () => Promise.resolve(mockBeans),
  });
});
afterEach(() => vi.clearAllMocks());
afterAll(() => vi.restoreAllMocks());


test('リストの項目をクリックすると、対応する詳細ページに遷移する', async () => {
  // テスト用のルーターを設定
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

  // APIから取得したリスト項目が表示されるのを待つ
  const linkElement = await screen.findByText('モック・ブルーマウンテン');
  expect(linkElement).toBeInTheDocument();

  // ユーザーがリンクをクリックする操作をシミュレート
  await userEvent.click(linkElement);

  // 詳細ページに表示されるはずのテキストが表示されていることを確認
  // これにより、画面遷移が成功したことがわかる
  expect(await screen.findByText('コーヒー豆詳細ページ')).toBeInTheDocument();
  expect(screen.getByText('表示している豆のID: 1')).toBeInTheDocument();
});