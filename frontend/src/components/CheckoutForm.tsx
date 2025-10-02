import { useState } from 'react';
import { useStripe, useElements, PaymentElement } from '@stripe/react-stripe-js';
import { Button, Alert, Container, TextInput } from '@mantine/core';

export default function CheckoutForm() {
  const stripe = useStripe();
  const elements = useElements();

  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);

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
      },
    });

    if (error.type === "card_error" || error.type === "validation_error") {
      setErrorMessage(error.message || '決済情報の入力に誤りがあります。');
    } else {
      setErrorMessage('予期せぬエラーが発生しました。');
    }

    setIsLoading(false);
  };

  return (
    <Container size="sm">
        <form onSubmit={handleSubmit}>
            <PaymentElement />
            <TextInput
                label="カード名義"
                placeholder="TARO BAISEN"
                required
                mt="md"
                mb="lg"
            />
            <Button type="submit" disabled={!stripe || isLoading} loading={isLoading} fullWidth>
                支払う
            </Button>

            {errorMessage && <Alert color="red" mt="md">{errorMessage}</Alert>}
        </form>
    </Container>
  );
}
