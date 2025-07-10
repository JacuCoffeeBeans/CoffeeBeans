import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { test, expect } from 'vitest';
import BeanDetailPage from './BeanDetailPage';
import { MantineProvider } from '@mantine/core';

test('URLのIDに基づいて、正しいIDが画面に表示される', () => {
  const testId = '123';

  // テスト用のルーターを設定し、特定のURLでコンポーネントを表示する
  render(
    <MantineProvider>
      <MemoryRouter initialEntries={[`/beans/${testId}`]}>
        <Routes>
          <Route path="/beans/:beanId" element={<BeanDetailPage />} />
        </Routes>
      </MemoryRouter>
    </MantineProvider>
  );

  // 画面に「ID: 123」と表示されていることを確認
  expect(screen.getByText(`表示している豆のID: ${testId}`)).toBeInTheDocument();
});