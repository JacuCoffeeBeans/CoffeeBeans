import { useState, useEffect } from 'react';
import {
  useStripe,
  useElements,
  PaymentElement,
} from '@stripe/react-stripe-js';
import {
  Button,
  Alert,
  Container,
  TextInput,
  Title,
  List,
  ThemeIcon,
  Text,
  SimpleGrid,
  Divider,
} from '@mantine/core';
import { IconCircleCheck } from '@tabler/icons-react';

// 型定義
interface CartItem {
  name: string;
  price: number;
  quantity: number;
}

interface CheckoutFormProps {
  cartItems: CartItem[];
  totalPrice: number;
}

export default function CheckoutForm({
  cartItems,
  totalPrice,
}: CheckoutFormProps) {
  const stripe = useStripe();
  const elements = useElements();

  // フォーム全体の入力状態
  const [name, setName] = useState('');
  const [postalCode, setPostalCode] = useState('');
  const [prefecture, setPrefecture] = useState('');
  const [city, setCity] = useState('');
  const [address1, setAddress1] = useState('');
  const [phone, setPhone] = useState('');
  const [cardholderName, setCardholderName] = useState('');

  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  // 郵便番号入力のハンドラ
  const handlePostalCodeChange = (
    event: React.ChangeEvent<HTMLInputElement>
  ) => {
    const value = event.currentTarget.value;
    const isDeleting = value.length < postalCode.length;

    const digits = value.replace(/[^0-9]/g, '').slice(0, 7);

    if (digits.length < 3) {
      setPostalCode(digits);
    } else if (digits.length === 3) {
      // 3桁目を入力したときはハイフンを追加
      // 4桁目から削除して3桁になったときはハイフンをつけない
      if (!isDeleting) {
        setPostalCode(`${digits}-`);
      } else {
        setPostalCode(digits);
      }
    } else {
      // 4桁以上
      setPostalCode(`${digits.slice(0, 3)}-${digits.slice(3)}`);
    }
  };

  // 郵便番号が7桁になったら住所を自動入力するuseEffect
  useEffect(() => {
    const digits = postalCode.replace(/-/g, '');
    if (digits.length === 7) {
      const fetchAddress = async () => {
        try {
          const response = await fetch(
            `https://zipcloud.ibsnet.co.jp/api/search?zipcode=${digits}`
          );
          const data = await response.json();

          if (data.status === 200 && data.results) {
            const result = data.results[0];
            setPrefecture(result.address1 || '');
            setCity((result.address2 || '') + (result.address3 || ''));
          } else {
            // 該当する住所が見つからなかった場合、フォームをクリア
            console.warn('該当する住所が見つかりませんでした。', data.message);
            setPrefecture('');
            setCity('');
          }
        } catch (error) {
          // 通信エラーが発生した場合もフォームをクリア
          console.error('住所の取得中にエラーが発生しました:', error);
          setPrefecture('');
          setCity('');
        }
      };
      fetchAddress();
    }
  }, [postalCode]);

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    if (!stripe || !elements) {
      return;
    }

    setIsLoading(true);

    const { error } = await stripe.confirmPayment({
      elements,
      confirmParams: {
        return_url: `${window.location.origin}/checkout/success`,
        shipping: {
          name: name,
          phone: phone,
          address: {
            line1: address1,
            city: city,
            state: prefecture,
            postal_code: postalCode,
            country: 'JP',
          },
        },
      },
    });

    if (error.type === 'card_error' || error.type === 'validation_error') {
      setErrorMessage(error.message || '決済情報の入力に誤りがあります。');
    } else {
      setErrorMessage('予期せぬエラーが発生しました。');
    }

    setIsLoading(false);
  };

  return (
    <form onSubmit={handleSubmit}>
      <Title order={2} mb="lg">
        ご注文内容
      </Title>
      <List
        spacing="xs"
        size="sm"
        mb="lg"
        icon={
          <ThemeIcon color="teal" size={24} radius="xl">
            <IconCircleCheck size="1rem" />
          </ThemeIcon>
        }
      >
        {cartItems.map((item, index) => (
          <List.Item key={index}>
            {item.name} ( {item.quantity}点 ) :{' '}
            {(item.price * item.quantity).toLocaleString()}円
          </List.Item>
        ))}
      </List>
      <Text ta="right" fw={700} size="lg">
        合計金額 : {totalPrice.toLocaleString()}円
      </Text>

      <Divider my="xl" />

      <Title order={2} mb="lg">
        配送先情報
      </Title>
      <SimpleGrid cols={1} spacing="md">
        <TextInput
          label="氏名"
          value={name}
          onChange={(e) => setName(e.currentTarget.value)}
          placeholder="焙煎 太郎"
          required
        />
        <TextInput
          label="郵便番号"
          value={postalCode}
          onChange={handlePostalCodeChange}
          placeholder="123-4567"
          maxLength={8}
          required
        />
        <TextInput
          label="都道府県"
          value={prefecture}
          onChange={(e) => setPrefecture(e.currentTarget.value)}
          placeholder="〇〇県"
          required
        />
        <TextInput
          label="市区町村"
          value={city}
          onChange={(e) => setCity(e.currentTarget.value)}
          placeholder="〇〇市"
          required
        />
        <TextInput
          label="番地以降の住所"
          value={address1}
          onChange={(e) => setAddress1(e.currentTarget.value)}
          placeholder="1-1"
          required
        />
        <TextInput
          label="電話番号"
          value={phone}
          onChange={(e) => setPhone(e.currentTarget.value)}
          placeholder="000-1111-2222"
          required
        />
      </SimpleGrid>

      <Divider my="xl" />

      <Title order={2} mb="lg">
        お支払い情報
      </Title>
      <PaymentElement />
      <TextInput
        label="カード名義"
        placeholder="TARO BAISEN"
        value={cardholderName}
        onChange={(e) => setCardholderName(e.currentTarget.value)}
        required
        mt="md"
        mb="lg"
      />

      <Button
        type="submit"
        disabled={!stripe || isLoading}
        loading={isLoading}
        fullWidth
        size="lg"
        mt="xl"
      >
        支払う
      </Button>

      {errorMessage && (
        <Alert color="red" mt="md">
          {errorMessage}
        </Alert>
      )}
    </form>
  );
}
