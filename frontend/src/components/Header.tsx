import { Link, useNavigate } from 'react-router-dom';
import { Title, Button, Group, Text, ActionIcon } from '@mantine/core';
import { IconShoppingCart } from '@tabler/icons-react';
import { useAuth } from '../contexts/AuthContext';
import { supabase } from '../lib/supabaseClient';

const Header = () => {
  const { session } = useAuth();
  const navigate = useNavigate();

  const handleLogout = async () => {
    await supabase.auth.signOut();
  };

  return (
    <Group justify="space-between" mb="lg">
      <Title order={1}>
        <Link to="/" style={{ textDecoration: 'none', color: 'inherit' }}>
          コーヒー豆アプリ
        </Link>
      </Title>
      {session ? (
        <Group>
          <Text>{session.user.email}</Text>
          <Button onClick={handleLogout}>ログアウト</Button>
          <Button component={Link} to="/my-beans" variant="outline">
            マイページ
          </Button>
          <Button component={Link} to="/beans/new">
            新しい豆を登録
          </Button>
          <ActionIcon component={Link} to="/cart" variant="default" size="lg" aria-label="カート">
            <IconShoppingCart size={20} />
          </ActionIcon>
        </Group>
      ) : (
        <Button onClick={() => navigate('/login')}>ログイン</Button>
      )}
    </Group>
  );
};

export default Header;
