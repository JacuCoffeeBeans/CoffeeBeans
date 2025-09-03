import React, { useEffect } from 'react';
import { Routes, Route, useNavigate } from 'react-router-dom';
import BeanListPage from './pages/BeanListPage';
import BeanDetailPage from './pages/BeanDetailPage';
import NewBeanPage from './pages/NewBeanPage';
import MyBeansPage from './pages/MyBeansPage';
import EditBeanPage from './pages/EditBeanPage';
import Login from './pages/Login';
import { useAuth } from './contexts/AuthContext';
import { Center, Loader } from '@mantine/core';
import Layout from './components/Layout';

const App = () => {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route path="/" element={<Layout />}>
        <Route index element={<BeanListPage />} />
        <Route path="beans/:beanId" element={<BeanDetailPage />} />
        <Route
          path="beans/new"
          element={
            <RequireAuth>
              <NewBeanPage />
            </RequireAuth>
          }
        />
        <Route
          path="my-beans"
          element={
            <RequireAuth>
              <MyBeansPage />
            </RequireAuth>
          }
        />
        <Route
          path="beans/:beanId/edit"
          element={
            <RequireAuth>
              <EditBeanPage />
            </RequireAuth>
          }
        />
      </Route>
    </Routes>
  );
};

const RequireAuth = ({ children }: { children: React.ReactElement }) => {
  const { session, isLoading } = useAuth();
  const navigate = useNavigate();

  

  useEffect(() => {
    if (!isLoading && !session) {
      navigate('/login', { replace: true });
    }
  }, [isLoading, session, navigate]);

  // 認証読み込み中、またはセッションがなくリダイレクト待ちの間はローダーを表示
  if (isLoading || !session) {
    return (
      <Center style={{ height: '100vh' }}>
        <Loader />
      </Center>
    );
  }

  // 読み込みが完了し、セッションが存在する場合のみ子コンポーネントを表示
  return children;
};

export default App;
