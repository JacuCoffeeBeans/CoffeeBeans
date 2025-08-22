import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { vi } from 'vitest';
import { BrowserRouter } from 'react-router-dom';
import { MantineProvider } from '@mantine/core';
import { ModalsProvider } from '@mantine/modals';
import NewBeanPage from './NewBeanPage';
import { supabase } from '../lib/supabaseClient';
vi.mock('../lib/supabaseClient');

// モックの設定
const mockNavigate = vi.fn();
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

window.fetch = vi.fn();

// テスト用のレンダー関数
const renderWithProviders = (ui: React.ReactElement) => {
  return render(
    <BrowserRouter>
      <MantineProvider>
        <ModalsProvider>{ui}</ModalsProvider>
      </MantineProvider>
    </BrowserRouter>
  );
};

describe('NewBeanPage', () => {
  beforeEach(() => {
    // window.alertが呼ばれてもクラッシュしないように、空の関数に置き換えます
    window.alert = vi.fn();

    // supabase.auth.getSessionが、常にログイン済みの偽セッションを返すように設定します
    vi.mocked(supabase.auth.getSession).mockResolvedValue({
      data: {
        session: {
          access_token: 'fake-jwt-token',
          user: { id: 'test-user-id' },
        },
      },
      error: null,
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } as any);
  });

  it('renders the form correctly', () => {
    renderWithProviders(<NewBeanPage />);
    expect(screen.getByText('新しいコーヒー豆を登録')).toBeInTheDocument();
    expect(screen.getByLabelText('名前')).toBeInTheDocument();
    expect(screen.getByLabelText('産地')).toBeInTheDocument();
    expect(screen.getByLabelText('価格')).toBeInTheDocument();
    expect(screen.getByLabelText('精製方法')).toBeInTheDocument();
    expect(screen.getByLabelText('焙煎度')).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: '登録する' })
    ).toBeInTheDocument();
  });

  it('shows validation errors when submitting an empty form', async () => {
    renderWithProviders(<NewBeanPage />);
    const submitButton = screen.getByRole('button', { name: '登録する' });

    await userEvent.click(submitButton);

    expect(
      await screen.findByText('名前を入力してください')
    ).toBeInTheDocument();
    expect(screen.getByText('産地を入力してください')).toBeInTheDocument();
    expect(
      screen.getByText('価格を0以上で入力してください')
    ).toBeInTheDocument();
    expect(screen.getByText('精製方法を選択してください')).toBeInTheDocument();
    expect(screen.getByText('焙煎度を選択してください')).toBeInTheDocument();
  });

  it('submits the form successfully with valid data', async () => {
    // fetchが成功した時の偽のレスポンスを設定
    vi.spyOn(window, 'fetch').mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ id: 1, name: 'Test Bean' }),
    } as Response);

    render(
      <MantineProvider>
        <BrowserRouter>
          <ModalsProvider>
            <NewBeanPage />
          </ModalsProvider>
        </BrowserRouter>
      </MantineProvider>
    );

    // フォームに入力
    await userEvent.type(screen.getByLabelText('名前'), 'Test Bean');
    await userEvent.type(screen.getByLabelText('産地'), 'Test Origin');
    await userEvent.type(screen.getByLabelText('価格'), '1000');

    await userEvent.click(screen.getByLabelText('精製方法'));
    await userEvent.click(screen.getByText('washed'));

    await userEvent.click(screen.getByLabelText('焙煎度'));
    await userEvent.click(screen.getByText('medium'));

    const submitButton = screen.getByRole('button', { name: '登録する' });
    await userEvent.click(submitButton);

    // fetchが呼ばれたか確認
    await waitFor(() => {
      // fetchが呼ばれたことを確認
      expect(window.fetch).toHaveBeenCalledTimes(1);
      // 正しいAuthorizationヘッダーが付与されていることを確認
      expect(window.fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/beans'), // URL
        expect.objectContaining({
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Authorization: 'Bearer fake-jwt-token', // トークンが付いているかチェック！
          },
        })
      );
    });

    // 成功モーダルが表示されるか確認
    expect(await screen.findByText('登録完了')).toBeInTheDocument();
    expect(
      screen.getByText('コーヒー豆の情報が正常に登録されました。')
    ).toBeInTheDocument();

    // OKボタンを押すとnavigateが呼ばれるか確認
    const okButton = screen.getByRole('button', { name: 'OK' });
    await userEvent.click(okButton);
    expect(mockNavigate).toHaveBeenCalledWith('/');
  });

  it('shows an error alert when API call fails', async () => {
    vi.mocked(fetch).mockResolvedValueOnce(
      new Response(JSON.stringify({ message: 'Failed to create bean' }), {
        status: 500,
      })
    );

    renderWithProviders(<NewBeanPage />);

    // フォームに入力
    await userEvent.type(screen.getByLabelText('名前'), 'Test Bean');
    await userEvent.type(screen.getByLabelText('産地'), 'Test Origin');
    await userEvent.type(screen.getByLabelText('価格'), '1000');

    await userEvent.click(screen.getByLabelText('精製方法'));
    await userEvent.click(screen.getByText('washed'));

    await userEvent.click(screen.getByLabelText('焙煎度'));
    await userEvent.click(screen.getByText('medium'));

    const submitButton = screen.getByRole('button', { name: '登録する' });
    await userEvent.click(submitButton);

    // エラーアラートが表示されるか確認
    expect(await screen.findByText('登録エラー')).toBeInTheDocument();
    expect(screen.getByText('Failed to create bean')).toBeInTheDocument();
  });
});
