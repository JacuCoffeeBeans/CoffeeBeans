import { Outlet } from 'react-router-dom';
import { Container } from '@mantine/core';
import { Notifications } from '@mantine/notifications';
import Header from './Header';

const Layout = () => {
  return (
    <Container mt="xl">
      <Notifications />
      <Header />
      <main>
        <Outlet />
      </main>
    </Container>
  );
};

export default Layout;
