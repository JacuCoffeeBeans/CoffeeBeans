import { Routes, Route } from 'react-router-dom';
import BeanListPage from './pages/BeanListPage';
import BeanDetailPage from './pages/BeanDetailPage';

function App() {
  return (
    <Routes>
      {/* ルートパス ("/") には、一覧ページを表示 */}
      <Route path="/" element={<BeanListPage />} />

      {/* "/beans/:beanId" というパスには、詳細ページを表示 */}
      {/* :beanId の部分には、1や2などの具体的なIDが入る */}
      <Route path="/beans/:beanId" element={<BeanDetailPage />} />
    </Routes>
  );
}

export default App;