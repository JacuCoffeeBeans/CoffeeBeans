import { Outlet } from 'react-router-dom';
import { Container } from '@mantine/core';
import Header from './Header';

const Layout = () => {
  return (
    <Container mt="xl">
      <Header />
      <main>
        <Outlet />
      </main>
    </Container>
  );
};

export default Layout;
