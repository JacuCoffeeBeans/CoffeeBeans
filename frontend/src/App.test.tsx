import { render, screen } from '@testing-library/react';
import { beforeAll, afterEach, afterAll, test, expect, vi } from 'vitest';
import App from './App';
import { MantineProvider } from '@mantine/core';

// 成功時のダミーレスポンスを定義
const mockBeans = [
  { id: 1, name: 'モック・ブルーマウンテン' },
  { id: 2, name: 'モック・キリマンジャロ' },
];

// 'fetch'関数をグローバルにモックする設定
beforeAll(() => {
  globalThis.fetch = vi.fn().mockImplementation(() =>
    Promise.resolve({
      ok: true,
      json: () => Promise.resolve(mockBeans),
    })
  );
});

// 各テスト後にモックをリセット
afterEach(() => {
  vi.clearAllMocks();
});

// 全テスト終了後に元のfetchに戻す
afterAll(() => {
  vi.restoreAllMocks();
});


test('データ取得が成功した場合、コーヒー豆のリストが表示される', async () => {
  render(
    <MantineProvider>
      <App />
    </MantineProvider>
  );

  // API通信が完了し、モックデータが表示されるのを待つ
  const firstItem = await screen.findByText('モック・ブルーマウンテン');

  // モックデータが正しく表示されていることを確認
  expect(firstItem).toBeInTheDocument();
  expect(screen.getByText('モック・キリマンジャロ')).toBeInTheDocument();

  // 最終的に「ローディング中...」が表示されていないことを確認
  expect(screen.queryByText(/ローディング中.../i)).not.toBeInTheDocument();
});