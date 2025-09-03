import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { vi } from 'vitest';
import { BrowserRouter } from 'react-router-dom';
import { MantineProvider } from '@mantine/core';
import { ModalsProvider } from '@mantine/modals';
import type { Session } from '@supabase/supabase-js';
import EditBeanPage from './EditBeanPage';
import { AuthProvider } from '../contexts/AuthContext';
import { supabase } from '../lib/supabaseClient';

// Mocks
vi.mock('../lib/supabaseClient');
const mockNavigate = vi.fn();
const mockUseParams = vi.fn();
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useNavigate: () => mockNavigate,
    useParams: () => mockUseParams(),
  };
});

window.fetch = vi.fn();
window.alert = vi.fn(); // Mock window.alert

const mockBean = {
  id: 1,
  name: 'Old Bean Name',
  origin: 'Old Origin',
  price: 1200,
  process: 'washed',
  roast_profile: 'medium',
};

const mockSession: Session = {
  access_token: 'fake-test-token',
  refresh_token: 'test-refresh-token',
  expires_in: 3600,
  token_type: 'bearer',
  user: {
    id: 'test-user',
    app_metadata: {},
    user_metadata: {},
    aud: 'authenticated',
    created_at: new Date().toISOString(),
  },
};

const renderWithProviders = (ui: React.ReactElement) => {
  return render(
    <BrowserRouter>
      <MantineProvider>
        <ModalsProvider>
          <AuthProvider>{ui}</AuthProvider>
        </ModalsProvider>
      </MantineProvider>
    </BrowserRouter>
  );
};

describe('EditBeanPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseParams.mockReturnValue({ beanId: '1' });
    
    // Mock supabase session
    vi.mocked(supabase.auth.getSession).mockResolvedValue({
      data: { session: mockSession },
      error: null,
    });
    vi.mocked(supabase.auth.onAuthStateChange).mockReturnValue({
        data: { subscription: { id: 'sub-id', callback: vi.fn(), unsubscribe: vi.fn() } },
    });


    // Mock the GET request for initial data fetching
    vi.mocked(fetch).mockImplementation((url) => {
      if (url === `/api/beans/1`) {
        return Promise.resolve(
          new Response(JSON.stringify(mockBean), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          })
        );
      }
      return Promise.resolve(new Response(null, { status: 404 }));
    });
  });

  it('fetches bean data and populates the form', async () => {
    renderWithProviders(<EditBeanPage />);

    expect(await screen.findByDisplayValue('Old Bean Name')).toBeInTheDocument();
    expect(screen.getByDisplayValue('Old Origin')).toBeInTheDocument();
    expect(screen.getByDisplayValue('1200')).toBeInTheDocument();
    
    const processSelect = screen.getByLabelText('精製方法');
    expect(processSelect).toHaveValue('washed');

    const roastSelect = screen.getByLabelText('焙煎度');
    expect(roastSelect).toHaveValue('medium');
  });

  it('submits updated data and navigates on success', async () => {
    // Mock the PUT request for submission
    vi.mocked(fetch).mockImplementation((url, options) => {
      if (url === '/api/beans/1' && options?.method === 'PUT') {
        return Promise.resolve(
          new Response(null, {
            status: 204, // No Content
          })
        );
      }
      // Still need the GET mock for initial load
      return Promise.resolve(
        new Response(JSON.stringify(mockBean), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      );
    });

    renderWithProviders(<EditBeanPage />);

    await screen.findByDisplayValue('Old Bean Name');

    await userEvent.clear(screen.getByLabelText('名前'));
    await userEvent.type(screen.getByLabelText('名前'), 'New Bean Name');
    await userEvent.click(screen.getByRole('button', { name: '更新する' }));

    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith('/api/beans/1', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${mockSession.access_token}`,
        },
        body: JSON.stringify({
          name: 'New Bean Name',
          origin: 'Old Origin',
          price: 1200,
          process: 'washed',
          roast_profile: 'medium',
        }),
      });
    });

    expect(await screen.findByText('更新完了')).toBeInTheDocument();
    
    await userEvent.click(screen.getByRole('button', { name: 'OK' }));
    expect(mockNavigate).toHaveBeenCalledWith('/my-beans');
  });

  it('shows an error message if fetching initial data fails', async () => {
    vi.mocked(fetch).mockResolvedValueOnce(
      new Response(null, { status: 500, statusText: 'Internal Server Error' })
    );
    renderWithProviders(<EditBeanPage />);
    expect(await screen.findByText('エラー')).toBeInTheDocument();
    expect(screen.getByText(/HTTP error!/)).toBeInTheDocument();
  });

  it('shows an error message if submission fails', async () => {
    vi.mocked(fetch)
      .mockResolvedValueOnce(
        new Response(JSON.stringify(mockBean), { status: 200 })
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ message: 'Update failed' }), { status: 500 })
      );

    renderWithProviders(<EditBeanPage />);
    await screen.findByDisplayValue('Old Bean Name');

    await userEvent.click(screen.getByRole('button', { name: '更新する' }));

    expect(await screen.findByText('エラー')).toBeInTheDocument();
    expect(screen.getByText('Update failed')).toBeInTheDocument();
  });
});