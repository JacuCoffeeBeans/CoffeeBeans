import { useEffect, useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { Container, Title, Table, Loader, Alert, Center, Button, Group, Text, NumberInput } from '@mantine/core';
import { modals } from '@mantine/modals';
import { notifications } from '@mantine/notifications';
import { IconAlertCircle, IconTrash, IconArrowLeft } from '@tabler/icons-react';
import { useAuth } from '../contexts/AuthContext';

// カートアイテムの型定義
interface CartItem {
  id: string; // cart_itemsテーブルの主キーID
  bean_id: number;
  name: string;
  price: number;
  quantity: number;
}

export default function CartPage() {
  const { session } = useAuth();
  const navigate = useNavigate();
  const [items, setItems] = useState<CartItem[]>([]);
  const [originalItems, setOriginalItems] = useState<CartItem[]>([]); // 元のアイテム情報を保持
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // カートの中身を取得する関数
  const fetchCartItems = async () => {
    if (!session) return;
    try {
      setLoading(true);
      const response = await fetch('/api/cart', {
        headers: {
          Authorization: `Bearer ${session.access_token}`,
        },
      });
      if (!response.ok) {
        throw new Error('カート情報の取得に失敗しました。');
      }
      const data = await response.json();
      // APIが { "items": [...] } という形式か、あるいは [...] という配列そのものを返すか不明なため、両方に対応する
      if (Array.isArray(data)) {
        setItems(data);
        setOriginalItems(data);
      } else if (data && Array.isArray(data.items)) {
        setItems(data.items);
        setOriginalItems(data.items);
      } else {
        setItems([]); // 想定外の形式なら空にする
        setOriginalItems([]);
      }
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '不明なエラー');
    } finally {
      setLoading(false);
    }
  };

  // カートアイテムの数量をローカルで更新する関数
  const handleUpdateQuantity = (cartItemId: string, newQuantity: number) => {
    if (newQuantity <= 0) return;
    setItems(items.map(item => item.id === cartItemId ? { ...item, quantity: newQuantity } : item));
  };

  // カートからアイテムを削除する関数（確認モーダル付き）
  const handleDeleteItem = (cartItemId: string, beanName: string) => {
    modals.openConfirmModal({
      title: '削除の確認',
      centered: true,
      children: (
        <Text size="sm">
          <strong>{beanName}</strong>をカートから削除しますか？
        </Text>
      ),
      labels: { confirm: '削除', cancel: 'キャンセル' },
      confirmProps: { color: 'red' },
      onConfirm: async () => {
        if (!session) return;

        try {
          const response = await fetch(`/api/cart/items/${cartItemId}`,
            {
              method: 'DELETE',
              headers: {
                Authorization: `Bearer ${session.access_token}`,
              },
            }
          );

          if (!response.ok) {
            throw new Error('商品の削除に失敗しました。');
          }

          // 状態を更新してUIに反映
          setItems(items.filter(item => item.id !== cartItemId));
          notifications.show({ title: '成功', message: '商品をカートから削除しました。', color: 'teal' });
        } catch (e: unknown) {
          notifications.show({ title: 'エラー', message: e instanceof Error ? e.message : '不明なエラー', color: 'red' });
        }
      },
    });
  };

  useEffect(() => {
    if (!session) {
      navigate('/login');
    } else {
      fetchCartItems();
    }
  }, [session, navigate]);

  // ページ離脱時に変更を保存するためのuseEffect
  useEffect(() => {
    return () => {
      if (session) {
        const changedItems = items.filter(item => {
          const originalItem = originalItems.find(o => o.id === item.id);
          return originalItem && originalItem.quantity !== item.quantity;
        });

        if (changedItems.length > 0) {
          // Promise.allで複数の更新処理を並行して実行
          Promise.all(changedItems.map(item =>
            fetch(`/api/cart/items/${item.id}`,
              {
                method: 'PUT',
                headers: {
                  'Content-Type': 'application/json',
                  Authorization: `Bearer ${session.access_token}`,
                },
                body: JSON.stringify({ quantity: item.quantity }),
              }
            )
          )).catch(error => {
            // エラーハンドリング（例: ログ出力や通知）
            console.error("カートの更新中にエラーが発生しました:", error);
            notifications.show({ title: 'エラー', message: 'カートの同期に失敗しました。', color: 'red' });
          });
        }
      }
    }
  }, [items, originalItems, session]);

  // 合計金額の計算
  const totalPrice = items.reduce((acc, item) => acc + item.price * item.quantity, 0);

  if (loading) {
    return <Center style={{ height: '50vh' }}><Loader /></Center>;
  }

  if (error) {
    return <Container><Alert icon={<IconAlertCircle size="1rem" />} title="エラー" color="red">{error}</Alert></Container>;
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
      {items.length === 0 ? (
        <Text>カートは空です。</Text>
      ) : (
        <>
          <Table verticalSpacing="sm">
            <Table.Thead>
              <Table.Tr>
                <Table.Th>商品名</Table.Th>
                <Table.Th>単価</Table.Th>
                <Table.Th>数量</Table.Th>
                <Table.Th>小計</Table.Th>
                <Table.Th />
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {items.map((item) => (
                <Table.Tr key={item.id}>
                  <Table.Td>{item.name}</Table.Td>
                  <Table.Td>{item.price.toLocaleString()}円</Table.Td>
                  <Table.Td>
                    <NumberInput
                      value={item.quantity}
                      onChange={(value) => handleUpdateQuantity(item.id, Number(value))}
                      min={1}
                      max={99}
                      style={{ width: 80 }}
                    />
                  </Table.Td>
                  <Table.Td>{(item.price * item.quantity).toLocaleString()}円</Table.Td>
                  <Table.Td>
                    <Button variant="light" color="red" onClick={() => handleDeleteItem(item.id, item.name)} aria-label={`delete ${item.name}`}>
                      <IconTrash size={16} />
                    </Button>
                  </Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
          <Group justify="flex-end" mt="lg">
            <Title order={3}>合計: {totalPrice.toLocaleString()}円</Title>
          </Group>
          <Group justify="flex-end" mt="md">
            <Button component={Link} to="/checkout" size="lg">購入手続きへ進む</Button>
          </Group>
        </>
      )}
    </Container>
  );
}
