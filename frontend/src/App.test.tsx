import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { beforeAll, afterEach, afterAll, test, expect, vi } from 'vitest';
import App from './App';
import { MantineProvider } from '@mantine/core';

// APIのモック設定は、一覧ページが表示される際に必要なので残します
const mockBeans = [{ id: 1, name: 'モック・ブルーマウンテン' }];
beforeAll(() => {
  globalThis.fetch = vi.fn().mockResolvedValue({
    ok: true,
    json: () => Promise.resolve(mockBeans),
  });
});
afterEach(() => vi.clearAllMocks());
afterAll(() => vi.restoreAllMocks());


test('ルートパスにアクセスすると、コーヒー豆の一覧ページが表示される', async () => {
  // テスト用に、メモリ上で動作するルーターでAppコンポーネントをラップします
  render(
    <MemoryRouter initialEntries={['/']}>
      <MantineProvider>
        <App />
      </MantineProvider>
    </MemoryRouter>
  );

  // Appコンポーネントが"/"というパスを解釈し、
  // BeanListPageをレンダリングした結果、以下のテキストが表示されるのを待つ
  const listTitle = await screen.findByText(/コーヒー豆リスト/i);
  const firstItem = await screen.findByText('モック・ブルーマウンテン');

  // テキストが表示されていれば、正しくルーティングされたと判断できる
  expect(listTitle).toBeInTheDocument();
  expect(firstItem).toBeInTheDocument();
});