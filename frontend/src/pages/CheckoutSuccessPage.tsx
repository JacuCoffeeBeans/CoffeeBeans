import { Container, Title, Text, Button, Loader, Alert, Center } from '@mantine/core';
import { Link, useSearchParams, useNavigate } from 'react-router-dom';
import { useStripe } from '@stripe/react-stripe-js';
import { useEffect, useState } from 'react';
import { IconCircleCheck, IconAlertCircle } from '@tabler/icons-react';

export default function CheckoutSuccessPage() {
  const stripe = useStripe();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const [status, setStatus] = useState('loading');
  const [message, setMessage] = useState('');

  useEffect(() => {
    const clientSecret = searchParams.get('payment_intent_client_secret');

    if (!clientSecret) {
      // URLにclient_secretがない場合はトップページにリダイレクト
      navigate('/', { replace: true });
      return;
    }

    if (!stripe) {
      // Stripe.jsがまだ読み込まれていない場合は、次のeffectの実行を待つ
      return;
    }

    stripe.retrievePaymentIntent(clientSecret).then(({ paymentIntent }) => {
      switch (paymentIntent?.status) {
        case 'succeeded':
          setStatus('success');
          setMessage('ご購入ありがとうございます。お支払いが正常に完了しました。');
          break;
        case 'processing':
          setStatus('processing');
          setMessage('決済処理中です。完了までしばらくお待ちください。');
          break;
        case 'requires_payment_method':
          setStatus('error');
          setMessage(
            'お支払いが失敗しました。お支払い方法をご確認の上、再度お試しください。'
          );
          break;
        default:
          setStatus('error');
          setMessage('何らかの問題が発生しました。サポートにお問い合わせください。');
          break;
      }
    });
  }, [stripe, searchParams]);

  const renderContent = () => {
    switch (status) {
      case 'loading':
        return <Loader />;
      case 'success':
        return (
          <>
            <IconCircleCheck size={80} color="teal" />
            <Title order={2} mt="md">{message}</Title>
            <Button component={Link} to="/" mt="xl">
              トップページに戻る
            </Button>
          </>
        );
      case 'processing':
        return (
          <>
            <Loader />
            <Title order={3} mt="md">{message}</Title>
            <Text c="dimmed" size="sm" mt="sm">
              このページは自動的に更新されます。
            </Text>
          </>
        );
      case 'error':
        return (
          <>
            <IconAlertCircle size={80} color="red" />
            <Title order={3} mt="md">決済エラー</Title>
            <Alert color="red" mt="lg">
              {message}
            </Alert>
            <Button component={Link} to="/cart" mt="xl">
              カートに戻る
            </Button>
          </>
        );
      default:
        return null;
    }
  };

  return (
    <Container mt={100}>
      <Center>
        <div style={{ textAlign: 'center' }}>{renderContent()}</div>
      </Center>
    </Container>
  );
}