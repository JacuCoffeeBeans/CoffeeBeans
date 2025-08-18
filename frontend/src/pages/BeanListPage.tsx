import { useEffect, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import {
  Container,
  Title,
  List,
  Loader,
  Alert,
  Center,
  ThemeIcon,
  Text,
  Button,
  Group,
} from '@mantine/core';
import { IconAlertCircle, IconCoffee } from '@tabler/icons-react';
import { useAuth } from '../contexts/AuthContext';
import { supabase } from '../lib/supabaseClient';

interface Bean {
  id: number;
  name: string;
}

export default function BeanListPage() {
  const [beans, setBeans] = useState<Bean[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const { session } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    const fetchBeans = async () => {
      try {
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

  const handleLogout = async () => {
    await supabase.auth.signOut();
  };

  if (loading) {
    return (
      <Center style={{ height: '100vh' }}>
        <Loader />
        <Title order={3} ml="md">ローディング中...</Title>
      </Center>
    );
  }

  if (error) {
    return (
      <Container mt="xl">
        <Alert icon={<IconAlertCircle size="1rem" />} title="エラー" color="red">
          データの取得に失敗しました: {error}
        </Alert>
      </Container>
    );
  }

  return (
    <Container mt="xl">
      <Group justify="space-between" mb="lg">
        <Title order={1}>
          コーヒー豆リスト
        </Title>
        {session ? (
          <Group>
            <Text>{session.user.email}</Text>
            <Button onClick={handleLogout}>ログアウト</Button>
            <Button component={Link} to="/beans/new">
              新しい豆を登録
            </Button>
          </Group>
        ) : (
          <Button onClick={() => navigate('/login')}>ログイン</Button>
        )}
      </Group>
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
          <List.Item key={bean.id}>
            <Link to={`/beans/${bean.id}`} style={{ textDecoration: 'none' }}>
              <Text component="span" c="blue.7">
                {bean.name}
              </Text>
            </Link>
          </List.Item>
        ))}
      </List>
    </Container>
  );
}