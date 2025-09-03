import { useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
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
  Loader,
  Center,
} from '@mantine/core';
import { useForm } from '@mantine/form';
import { useModals } from '@mantine/modals';
import { IconAlertCircle } from '@tabler/icons-react';
import { useAuth } from '../contexts/AuthContext';

interface BeanInput {
  name: string;
  origin: string;
  price: number | '';
  process: string;
  roast_profile: string;
}

export default function EditBeanPage() {
  const { beanId } = useParams<{ beanId: string }>();
  const navigate = useNavigate();
  const modals = useModals();
  const { session } = useAuth();
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState<boolean>(false);
  const [pageLoading, setPageLoading] = useState<boolean>(true);

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

  useEffect(() => {
    const fetchBean = async () => {
      setPageLoading(true);
      try {
        const response = await fetch(`/api/beans/${beanId}`);
        if (!response.ok) {
          throw new Error(`HTTP error! status: ${response.status}`);
        }
        const data = await response.json();
        form.setValues({
          name: data.name,
          origin: data.origin,
          price: data.price,
          process: data.process,
          roast_profile: data.roast_profile,
        });
      } catch (e: unknown) {
        if (e instanceof Error) {
          setError(e.message);
        } else {
          setError('不明なエラーが発生しました。');
        }
      } finally {
        setPageLoading(false);
      }
    };

    fetchBean();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [beanId]);

  const handleSubmit = async (values: BeanInput) => {
    setLoading(true);
    setError(null);

    if (!session) {
      alert('ログインが必要です。再度ログインしてください。');
      setLoading(false);
      return;
    }

    try {
      const response = await fetch(`/api/beans/${beanId}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${session.access_token}`,
        },
        body: JSON.stringify({
          ...values,
          price: Number(values.price),
        }),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.message || `HTTP error! status: ${response.status}`);
      }

      modals.openConfirmModal({
        title: '更新完了',
        centered: true,
        children: (
          <Text size="sm">
            コーヒー豆の情報が正常に更新されました。
          </Text>
        ),
        labels: { confirm: 'OK', cancel: '閉じる' },
        cancelProps: { style: { display: 'none' } },
        onConfirm: () => navigate('/my-beans'),
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

  if (pageLoading) {
    return (
      <Center style={{ height: '50vh' }}>
        <Loader />
      </Center>
    );
  }

  return (
    <Container mt="xl">
      <Title order={1} mb="lg">
        コーヒー豆を編集
      </Title>
      <Box component="form" onSubmit={form.onSubmit(handleSubmit)}>
        {error && (
          <Alert icon={<IconAlertCircle size="1rem" />} title="エラー" color="red" mb="lg">
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
            更新する
          </Button>
          <Button component={Link} to="/my-beans" variant="outline">
            キャンセル
          </Button>
        </Group>
      </Box>
    </Container>
  );
}
