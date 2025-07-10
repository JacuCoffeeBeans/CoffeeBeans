import { useParams } from 'react-router-dom';
import { Container, Title, Text } from '@mantine/core';

export default function BeanDetailPage() {
  // URLのパラメータ（:beanId）を取得する
  const { beanId } = useParams();

  return (
    <Container mt="xl">
      <Title order={1}>コーヒー豆詳細ページ</Title>
      <Text mt="md">表示している豆のID: {beanId}</Text>
    </Container>
  );
}