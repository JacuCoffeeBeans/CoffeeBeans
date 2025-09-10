import { useEffect, useState } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import {
  Container,
  Title,
  Text,
  Paper,
  Loader,
  Alert,
  Center,
  Button,
  Group,
  NumberInput,
} from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { IconAlertCircle, IconArrowLeft, IconShoppingCart } from '@tabler/icons-react';
import { useAuth } from '../contexts/AuthContext';

// 表示する豆データの型を定義
interface BeanDetail {
  id: number;
  name: string;
  origin: string;
  price: number;
  process: string;
  roast_profile: string;
}

export default function BeanDetailPage() {
  // URLからbeanIdを取得
  const { beanId } = useParams();
  const navigate = useNavigate();
  const { session } = useAuth(); // 認証状態を取得

  // stateを定義
  const [bean, setBean] = useState<BeanDetail | null>(null);
  const [quantity, setQuantity] = useState<number | string>(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isAdding, setIsAdding] = useState(false); // カート追加処理中かどうかの状態

  // beanIdを使ってAPIを呼び出す
  useEffect(() => {
    if (!beanId) return;

    const fetchBeanDetail = async () => {
      try {
        setLoading(true);
        const response = await fetch(`/api/beans/${beanId}`);
        if (!response.ok) {
          throw new Error(`HTTP error! status: ${response.status}`);
        }
        const data = await response.json();
        setBean(data);
      } catch (e: unknown) {
        if (e instanceof Error) {
          setError(e.message);
        } else {
          setError('不明なエラーが発生しました');
        }
      } finally {
        setLoading(false);
      }
    };

    fetchBeanDetail();
  }, [beanId]); // beanIdが変わるたびにAPIを再実行

  // 「カートに追加」ボタンのハンドラ
  const handleAddToCart = async () => {
    if (!session) {
      // 未ログインの場合はログインページにリダイレクト
      navigate('/login');
      return;
    }

    if (!bean || typeof quantity !== 'number' || quantity <= 0) {
      notifications.show({
        title: 'エラー',
        message: '数量を正しく入力してください。',
        color: 'red',
      });
      return;
    }

    setIsAdding(true);
    try {
      const response = await fetch('/api/cart/items', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${session.access_token}`,
        },
        body: JSON.stringify({
          bean_id: bean.id,
          quantity: quantity,
        }),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.message || 'カートへの追加に失敗しました。');
      }

      notifications.show({
        title: '成功',
        message: `${bean.name}をカートに追加しました。`,
        color: 'teal',
      });
    } catch (e: unknown) {
      const errorMessage = e instanceof Error ? e.message : '不明なエラーが発生しました。';
      notifications.show({
        title: 'エラー',
        message: errorMessage,
        color: 'red',
      });
    } finally {
      setIsAdding(false);
    }
  };


  if (loading) {
    return (
      <Center style={{ height: '100vh' }}>
        <Loader />
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
      <Button
        component={Link}
        to="/"
        leftSection={<IconArrowLeft size={14} />}
        variant="subtle"
        mb="md"
      >
        一覧に戻る
      </Button>

      {bean && (
        <Paper shadow="sm" p="lg" radius="md" withBorder>
          <Title order={1}>{bean.name}</Title>
          <Text size="lg" c="dimmed" mt="xs">
            産地: {bean.origin}
          </Text>
          <Text mt="sm">精製方法: {bean.process}</Text>
          <Text mt="sm">焙煎度: {bean.roast_profile}</Text>
          <Text size="xl" fw={700} mt="md">
            {bean.price}円
          </Text>

          <Group mt="lg" align="flex-end">
            <NumberInput
              label="数量"
              value={quantity}
              onChange={setQuantity}
              min={1}
              max={99}
              style={{ width: 100 }}
            />
            <Button
              leftSection={<IconShoppingCart size={16} />}
              onClick={handleAddToCart}
              loading={isAdding}
            >
              カートに追加
            </Button>
          </Group>
        </Paper>
      )}
    </Container>
  );
}
