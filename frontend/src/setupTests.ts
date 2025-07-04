import '@testing-library/jest-dom';

// test()やexpect()などが実行される前に、全てのテスト環境で一度だけ実行されるセットアップ
beforeAll(() => {
  // window.matchMediaが未定義だった場合に、ダミーの関数を定義する
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: vi.fn().mockImplementation(query => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(), // deprecated
      removeListener: vi.fn(), // deprecated
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  });
});