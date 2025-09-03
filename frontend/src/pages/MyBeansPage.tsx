import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import {
  Container,
  List,
  Loader,
  Alert,
  Center,
  ThemeIcon,
  Text,
  Title,
  Button,
  Group,
} from '@mantine/core';
import { IconAlertCircle, IconCoffee } from '@tabler/icons-react';
import { useAuth } from '../contexts/AuthContext';

interface Bean {
  id: number;
  name: string;
}

export default function MyBeansPage() {
  const { session } = useAuth();
  const [beans, setBeans] = useState<Bean[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchBeans = async () => {
      if (!session) {
        // このページはRequireAuthで保護されているため、通常この状態にはならない
        setLoading(false);
        return;
      }

      setLoading(true);
      try {
        const response = await fetch('/api/my/beans', {
          headers: {
            Authorization: `Bearer ${session.access_token}`,
          },
        });
        if (!response.ok) {
          throw new Error(`HTTP error! status: ${response.status}`);
        }
        const data: Bean[] | null = await response.json();
        setBeans(data || []); // APIがnullを返した場合も空配列として扱う
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
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [session?.access_token]); // sessionオブジェクト全体ではなく、変更されないアクセストークンを依存配列に設定

  if (loading) {
    return (
      <Center style={{ height: '100vh' }}>
        <Loader />
        <Title order={3} ml="md">
          ローディング中...
        </Title>
      </Center>
    );
  }

  if (error) {
    return (
      <Container mt="xl">
        <Alert
          icon={<IconAlertCircle size="1rem" />}
          title="エラー"
          color="red"
        >
          データの取得に失敗しました: {error}
        </Alert>
      </Container>
    );
  }

  return (
    <Container mt="xl">
      <Title order={1} mb="lg">
        マイページ（出品した豆一覧）
      </Title>
      <List
        spacing="md"
        size="sm"
        icon={
          <ThemeIcon color="teal" size={24} radius="xl">
            <IconCoffee size="1rem" />
          </ThemeIcon>
        }
      >
        {beans.length > 0 ? (
          beans.map((bean) => (
            <List.Item key={bean.id}>
              <Group justify="space-between">
                <Link
                  to={`/beans/${bean.id}`}
                  style={{ textDecoration: 'none' }}
                >
                  <Text component="span" c="blue.7">
                    {bean.name}
                  </Text>
                </Link>
                <Group>
                  <Button
                    component={Link}
                    to={`/beans/${bean.id}/edit`}
                    size="xs"
                    variant="outline"
                  >
                    編集
                  </Button>
                  <Button size="xs" variant="outline" color="red">
                    削除
                  </Button>
                </Group>
              </Group>
            </List.Item>
          ))
        ) : (
          <Text>登録されている豆はありません。</Text>
        )}
      </List>
      <Group mt="xl">
        <Button component={Link} to="/" variant="outline">
          一覧に戻る
        </Button>
      </Group>
    </Container>
  );
}
