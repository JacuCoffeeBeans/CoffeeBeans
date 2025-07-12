import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { beforeAll, afterEach, afterAll, test, expect, vi } from 'vitest';
import BeanDetailPage from './BeanDetailPage';
import { MantineProvider } from '@mantine/core';

// APIのモック設定
const mockBeanDetail = {
  id: 1,
  name: 'モック・エチオピア',
  origin: 'モック産地',
  price: 1500,
};

beforeAll(() => {
  // fetchが呼ばれたら、常にmockBeanDetailを返すように設定
  globalThis.fetch = vi.fn().mockResolvedValue({
    ok: true,
    json: () => Promise.resolve(mockBeanDetail),
  });
});
afterEach(() => vi.clearAllMocks());
afterAll(() => vi.restoreAllMocks());

test('詳細データが正しく表示される', async () => {
  const testId = '1';

  // テスト用のルーターで、詳細ページのURLを直接指定
  render(
    <MantineProvider>
      <MemoryRouter initialEntries={[`/beans/${testId}`]}>
        <Routes>
          <Route path="/beans/:beanId" element={<BeanDetailPage />} />
        </Routes>
      </MemoryRouter>
    </MantineProvider>
  );

  // API通信とレンダリングが完了し、豆の名前が表示されるのを待つ
  expect(await screen.findByText(mockBeanDetail.name)).toBeInTheDocument();

  // その他の情報も正しく表示されていることを確認
  expect(screen.getByText(`産地: ${mockBeanDetail.origin}`)).toBeInTheDocument();
  expect(screen.getByText(`${mockBeanDetail.price}円`)).toBeInTheDocument();

  // APIが正しいIDで呼び出されたことを確認
  expect(globalThis.fetch).toHaveBeenCalledWith(`/api/beans/${testId}`);
});