import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { vi } from 'vitest';
import { BrowserRouter } from 'react-router-dom';
import { MantineProvider } from '@mantine/core';
import { ModalsProvider } from '@mantine/modals';
import NewBeanPage from './NewBeanPage';

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
    // 各テストの前にモックをリセット
    vi.mocked(fetch).mockClear();
    mockNavigate.mockClear();
  });

  it('renders the form correctly', () => {
    renderWithProviders(<NewBeanPage />);
    expect(screen.getByText('新しいコーヒー豆を登録')).toBeInTheDocument();
    expect(screen.getByLabelText('名前')).toBeInTheDocument();
    expect(screen.getByLabelText('産地')).toBeInTheDocument();
    expect(screen.getByLabelText('価格')).toBeInTheDocument();
    expect(screen.getByLabelText('精製方法')).toBeInTheDocument();
    expect(screen.getByLabelText('焙煎度')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '登録する' })).toBeInTheDocument();
  });

  it('shows validation errors when submitting an empty form', async () => {
    renderWithProviders(<NewBeanPage />);
    const submitButton = screen.getByRole('button', { name: '登録する' });

    await userEvent.click(submitButton);

    expect(await screen.findByText('名前を入力してください')).toBeInTheDocument();
    expect(screen.getByText('産地を入力してください')).toBeInTheDocument();
    expect(screen.getByText('価格を0以上で入力してください')).toBeInTheDocument();
    expect(screen.getByText('精製方法を選択してください')).toBeInTheDocument();
    expect(screen.getByText('焙煎度を選択してください')).toBeInTheDocument();
  });

  it('submits the form successfully with valid data', async () => {
    vi.mocked(fetch).mockResolvedValueOnce(new Response(JSON.stringify({ id: 1 }), { status: 201 }));

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

    // fetchが呼ばれたか確認
    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith('/api/beans', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          name: 'Test Bean',
          origin: 'Test Origin',
          price: 1000,
          process: 'washed',
          roast_profile: 'medium',
        }),
      });
    });

    // 成功モーダルが表示されるか確認
    expect(await screen.findByText('登録完了')).toBeInTheDocument();
    expect(screen.getByText('コーヒー豆の情報が正常に登録されました。')).toBeInTheDocument();

    // OKボタンを押すとnavigateが呼ばれるか確認
    const okButton = screen.getByRole('button', { name: 'OK' });
    await userEvent.click(okButton);
    expect(mockNavigate).toHaveBeenCalledWith('/');
  });

  it('shows an error alert when API call fails', async () => {
    vi.mocked(fetch).mockResolvedValueOnce(new Response(JSON.stringify({ message: 'Failed to create bean' }), { status: 500 }));

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
