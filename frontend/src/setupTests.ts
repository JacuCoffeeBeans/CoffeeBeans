import { afterEach, beforeAll, vi } from 'vitest';
import { cleanup } from '@testing-library/react';
import '@testing-library/jest-dom';

const matchMediaMock = vi.fn(query => ({
  matches: false,
  media: query,
  onchange: null,
  addListener: vi.fn(), // deprecated
  removeListener: vi.fn(), // deprecated
  addEventListener: vi.fn(),
  removeEventListener: vi.fn(),
  dispatchEvent: vi.fn(),
}));
vi.stubGlobal('matchMedia', matchMediaMock);

// 各テストの後にDOMのクリーンアップを実行します
afterEach(() => {
  cleanup();
  vi.clearAllMocks(); // モックの呼び出し履歴をリセット
});

// ResizeObserverのモック
const ResizeObserverMock = vi.fn(() => ({
  observe: vi.fn(),
  unobserve: vi.fn(),
  disconnect: vi.fn(),
}));
vi.stubGlobal('ResizeObserver', ResizeObserverMock);

// test()やexpect()などが実行される前に、全てのテスト環境で一度だけ実行されるセットアップ
beforeAll(() => {
  // JSDOMで不足しているlocation.assignをモック
  Object.defineProperty(window, 'location', {
    writable: true,
    value: { ...window.location, assign: vi.fn() },
  });
});

// --- Supabase Clientのモック ---
export const mockSupabase = {
  auth: {
    signInWithOtp: vi.fn().mockResolvedValue({ data: {}, error: null }),
    signInWithOAuth: vi.fn().mockResolvedValue({ data: {}, error: null }),
    signOut: vi.fn(),
    getSession: vi.fn().mockResolvedValue({ data: { session: null }, error: null }),
    onAuthStateChange: vi.fn().mockReturnValue({
      data: { subscription: { unsubscribe: vi.fn() } },
    }),
  },
};

vi.mock('./lib/supabaseClient', () => ({
  supabase: mockSupabase,
}));
