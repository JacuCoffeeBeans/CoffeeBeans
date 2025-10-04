import { useEffect, useState } from 'react';
import { stripePromise } from '../lib/stripe';
import { Elements } from '@stripe/react-stripe-js';
import CheckoutForm from '../components/CheckoutForm';
import { useAuth } from '../contexts/AuthContext';
import { Center, Container, Loader, Alert } from '@mantine/core';



// カートアイテムの型定義
interface CartItem {
  id: string;
  bean_id: number;
  name: string;
  price: number;
  quantity: number;
}

export default function CheckoutPage() {
  const [clientSecret, setClientSecret] = useState<string | null>(null);
  const [cartItems, setCartItems] = useState<CartItem[]>([]);
  const [totalPrice, setTotalPrice] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const { session } = useAuth();

  useEffect(() => {
    if (!session) return;

    const initializeCheckout = async () => {
      try {
        // 1. カート情報を取得
        const cartResponse = await fetch('/api/cart', {
          headers: {
            Authorization: `Bearer ${session.access_token}`,
          },
        });
        if (!cartResponse.ok) {
          throw new Error('カート情報の取得に失敗しました。');
        }
        const cartData = await cartResponse.json();
        const items = Array.isArray(cartData) ? cartData : (cartData?.items || []);

        if (items.length === 0) {
          setError('カートが空のため、決済に進めません。');
          setLoading(false);
          return;
        }
        setCartItems(items);
        setTotalPrice(items.reduce((acc, item) => acc + item.price * item.quantity, 0));

        // 2. 決済情報(PaymentIntent)を作成
        const paymentIntentResponse = await fetch('/api/checkout/payment-intent', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${session.access_token}`,
          },
          body: JSON.stringify({}), // バックエンドがセッションからカート内容を判断するためボディは空
        });

        if (!paymentIntentResponse.ok) {
          const errorData = await paymentIntentResponse.json();
          throw new Error(errorData.message || '決済の準備に失敗しました。');
        }

        const paymentIntentData = await paymentIntentResponse.json();
        const secret = paymentIntentData.client_secret || paymentIntentData.clientSecret;

        if (!secret) {
          throw new Error('client_secretがレスポンスに含まれていません。');
        }
        setClientSecret(secret);

      } catch (err: unknown) {
        const message = err instanceof Error ? err.message : '不明なエラーが発生しました。';
        console.error('Failed to initialize checkout:', message);
        setError(message);
      } finally {
        setLoading(false);
      }
    };

    initializeCheckout();
  }, [session]);

  if (loading) {
    return (
      <Center style={{ height: '80vh' }}>
        <Loader />
      </Center>
    );
  }

  if (error) {
    return (
      <Container my="xl">
        <Alert color="red" title="エラー">
          {error}
        </Alert>
      </Container>
    );
  }

  return (
    <Container my="xl">
      {clientSecret && cartItems.length > 0 ? (
        <Elements options={{ clientSecret }} stripe={stripePromise}>
          <CheckoutForm cartItems={cartItems} totalPrice={totalPrice} />
        </Elements>
      ) : (
        // ローディング完了後も何も表示されない場合のフォールバック
        <Center style={{ height: '80vh' }}>
          <Loader />
        </Center>
      )}
    </Container>
  );
}
