import { Routes, Route, useNavigate } from 'react-router-dom';
import BeanListPage from './pages/BeanListPage';
import BeanDetailPage from './pages/BeanDetailPage';
import NewBeanPage from './pages/NewBeanPage';
import Login from './pages/Login';
import { useAuth } from './contexts/AuthContext';
import { useEffect } from 'react';
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
      </Route>
    </Routes>
  );
};

const RequireAuth = ({ children }: { children: JSX.Element }) => {
  const { session } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    if (session === null) {
      navigate('/login', { replace: true });
    }
  }, [session, navigate]);

  return session ? children : null;
};

export default App;
