import { Container, Title, Text, Button } from '@mantine/core';
import { Link } from 'react-router-dom';

export default function CheckoutSuccessPage() {
  return (
    <Container style={{ textAlign: 'center' }} mt={100}>
      <Title order={2}>お支払いありがとうございます！</Title>
      <Text mt="md">ご注文が正常に完了しました。</Text>
      <Button component={Link} to="/" mt="xl">
        トップページに戻る
      </Button>
    </Container>
  );
}
