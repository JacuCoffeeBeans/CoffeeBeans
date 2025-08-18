import { Routes, Route, useNavigate } from 'react-router-dom';
import BeanListPage from './pages/BeanListPage';
import BeanDetailPage from './pages/BeanDetailPage';
import NewBeanPage from './pages/NewBeanPage';
import Login from './pages/Login';
import { useAuth } from './contexts/AuthContext';
import { useEffect } from 'react';

const App = () => {
  return (
    <Routes>
      <Route path="/" element={<BeanListPage />} />
      <Route path="/login" element={<Login />} />
      <Route path="/beans/:beanId" element={<BeanDetailPage />} />
      <Route
        path="/beans/new"
        element={
          <RequireAuth>
            <NewBeanPage />
          </RequireAuth>
        }
      />
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
