import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import {
  Container,
  Title,
  TextInput,
  NumberInput,
  Select,
  Button,
  Box,
  Alert,
  Group,
  Text,
} from '@mantine/core';
import { useForm } from '@mantine/form';
import { useModals } from '@mantine/modals';
import { IconAlertCircle } from '@tabler/icons-react';

interface BeanInput {
  name: string;
  origin: string;
  price: number | ''; // NumberInputは空文字を扱うことがあるため
  process: string;
  roast_profile: string;
}

export default function NewBeanPage() {
  const navigate = useNavigate();
  const modals = useModals();
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState<boolean>(false);

  const form = useForm<BeanInput>({
    initialValues: {
      name: '',
      origin: '',
      price: '',
      process: '',
      roast_profile: '',
    },
    validate: {
      name: (value) => (value.trim().length > 0 ? null : '名前を入力してください'),
      origin: (value) => (value.trim().length > 0 ? null : '産地を入力してください'),
      price: (value) => (value !== '' && Number(value) >= 0 ? null : '価格を0以上で入力してください'),
      process: (value) => (value ? null : '精製方法を選択してください'),
      roast_profile: (value) => (value ? null : '焙煎度を選択してください'),
    },
  });

  const handleSubmit = async (values: BeanInput) => {
    setLoading(true);
    setError(null);

    try {
      const response = await fetch('/api/beans', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          ...values,
          price: Number(values.price), // API送信時に数値に変換
        }),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.message || `HTTP error! status: ${response.status}`);
      }

      modals.openConfirmModal({
        title: '登録完了',
        centered: true,
        children: (
          <Text size="sm">
            コーヒー豆の情報が正常に登録されました。
          </Text>
        ),
        labels: { confirm: 'OK' },
        cancelProps: { style: { display: 'none' } },
        onConfirm: () => navigate('/'),
      });
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

  return (
    <Container mt="xl">
      <Title order={1} mb="lg">
        新しいコーヒー豆を登録
      </Title>
      <Box component="form" onSubmit={form.onSubmit(handleSubmit)}>
        {error && (
          <Alert icon={<IconAlertCircle size="1rem" />} title="登録エラー" color="red" mb="lg">
            {error}
          </Alert>
        )}
        <TextInput
          label="名前"
          placeholder="例：エチオピア イルガチェフェ"
          mb="sm"
          {...form.getInputProps('name')}
        />
        <TextInput
          label="産地"
          placeholder="例：エチオピア"
          mb="sm"
          {...form.getInputProps('origin')}
        />
        <NumberInput
          label="価格"
          placeholder="例：1500"
          mb="sm"
          min={0}
          hideControls
          {...form.getInputProps('price')}
        />
        <Select
          label="精製方法"
          placeholder="精製方法を選択してください"
          mb="sm"
          data={['natural', 'washed', 'honey']}
          {...form.getInputProps('process')}
        />
        <Select
          label="焙煎度"
          placeholder="焙煎度を選択してください"
          mb="xl"
          data={['light','cinnamon','medium','high','city','full_city','french','italian']}
          {...form.getInputProps('roast_profile')}
        />
        <Group>
          <Button type="submit" loading={loading}>
            登録する
          </Button>
          <Button component={Link} to="/" variant="outline">
            一覧に戻る
          </Button>
        </Group>
      </Box>
    </Container>
  );
}