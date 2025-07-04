import { useEffect, useState } from 'react';
import {
  Container,
  Title,
  List,
  Loader,
  Alert,
  Center,
  ThemeIcon,
} from '@mantine/core';
import { IconAlertCircle, IconCoffee } from '@tabler/icons-react';

// 1. バックエンドから受け取る豆データの型を定義
interface Bean {
  id: number;
  name: string;
}

function App() {
  // 2. 豆リスト、ローディング、エラーを管理するstate
  const [beans, setBeans] = useState<Bean[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  // 3. 初回表示時にAPIを呼び出す処理
  useEffect(() => {
    const fetchBeans = async () => {
      try {
        // Viteのプロキシ設定により、'/api/beans'はバックエンド(8080番)に転送される
        const response = await fetch('/api/beans');
        if (!response.ok) {
          throw new Error(`HTTP error! status: ${response.status}`);
        }
        const data: Bean[] = await response.json();
        setBeans(data);
      } catch (e: unknown) {
        if (e instanceof Error) {
          setError(e.message);
        } else {
          setError('不明なエラーが発生しました。');
        }
      } finally {
        setLoading(false);
      }
    };

    fetchBeans();
  }, []);

  // 4. ローディング中の表示
  if (loading) {
    return (
      <Center style={{ height: '100vh' }}>
        <Loader />
        <Title order={3} ml="md">ローディング中...</Title>
      </Center>
    );
  }

  // 5. エラー発生時の表示
  if (error) {
    return (
      <Container mt="xl">
        <Alert icon={<IconAlertCircle size="1rem" />} title="エラー" color="red">
          データの取得に失敗しました: {error}
        </Alert>
      </Container>
    );
  }

  // 6. 成功時のデータ表示
  return (
    <Container mt="xl">
      <Title order={1} mb="lg">
        コーヒー豆リスト
      </Title>
      <List
        spacing="xs"
        size="sm"
        center
        icon={
          <ThemeIcon color="teal" size={24} radius="xl">
            <IconCoffee size="1rem" />
          </ThemeIcon>
        }
      >
        {beans.map((bean) => (
          <List.Item key={bean.id}>{bean.name}</List.Item>
        ))}
      </List>
    </Container>
  );
}

export default App;