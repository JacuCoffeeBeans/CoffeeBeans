import { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import {
  Container,
  Title,
  Text,
  Paper,
  Loader,
  Alert,
  Center,
  Button,
} from '@mantine/core';
import { IconAlertCircle, IconArrowLeft } from '@tabler/icons-react';

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

  // stateを定義
  const [bean, setBean] = useState<BeanDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

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
        </Paper>
      )}
    </Container>
  );
}