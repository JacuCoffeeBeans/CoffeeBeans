import { useEffect, useState } from 'react';
import { loadStripe } from '@stripe/stripe-js';
import { Elements } from '@stripe/react-stripe-js';
import CheckoutForm from '../components/CheckoutForm';
import { useAuth } from '../contexts/AuthContext';
import { Center, Container, Loader, Alert } from '@mantine/core';

const stripePromise = loadStripe(import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY);

export default function CheckoutPage() {
  const [clientSecret, setClientSecret] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const { session } = useAuth();

  useEffect(() => {
    if (!session) return;

    const createPaymentIntent = async () => {
      try {
        const response = await fetch('/api/checkout/payment-intent', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${session.access_token}`,
          },
          body: JSON.stringify({}),
        });

        if (!response.ok) {
          const errorData = await response.json();
          throw new Error(errorData.message || '決済の準備に失敗しました。');
        }

        const data = await response.json();
        const secret = data.client_secret || data.clientSecret;

        if (!secret) {
          throw new Error('client_secretがレスポンスに含まれていません。');
        }
        setClientSecret(secret);

      } catch (err: unknown) {
        const message = err instanceof Error ? err.message : '不明なエラーが発生しました。';
        console.error('Failed to create PaymentIntent:', message);
        setError(message);
      }
    };

    createPaymentIntent();
  }, [session]);

  if (error) {
    return (
      <Container my="xl">
        <Alert color="red" title="エラー">
          {error}
        </Alert>
      </Container>
    );
  }

  // clientSecretが取得できるまでローダーを表示し、取得後にElementsをレンダリングする
  return (
    <Container my="xl">
      {clientSecret ? (
        <Elements options={{ clientSecret }} stripe={stripePromise}>
          <CheckoutForm />
        </Elements>
      ) : (
        <Center style={{ height: '80vh' }}>
          <Loader />
        </Center>
      )}
    </Container>
  );
}