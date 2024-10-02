import { AppShell, Burger, createTheme, Group, MantineProvider, NavLink, Text } from "@mantine/core";
import { useDisclosure } from '@mantine/hooks';
import { Link, Outlet } from "react-router-dom";

import "@mantine/core/styles.css";
import classNames from "./App.module.css";

const theme = createTheme({});

function App() {
  const [opened, {toggle}] = useDisclosure();

  return (
    <MantineProvider theme={theme} defaultColorScheme="dark">
      <AppShell
        header={{ height: 60 }}
	navbar={{ width: 150, breakpoint: 'sm', collapsed: { mobile: !opened } }}
        padding="md"
      >
        <AppShell.Header className={classNames.header}>
	  <Group h="100%" px="md">
	    <Burger opened={opened} onClick={toggle} hiddenFrom="sm" size="sm" />
            <Text size="xl" fw="bold" pl="md">
              PixelGW
            </Text>
	  </Group>
        </AppShell.Header>
        <AppShell.Navbar p="md">
	  <NavLink
	    component={Link}
            to="/applets"
	    label="Applets"
	  />
	  <NavLink
	    component={Link}
            to={'/channels'}
	    label="Channels"
	  />
	  <NavLink
	    component={Link}
            to="/devices"
	    label="Devices"
	  />
      </AppShell.Navbar>

        <AppShell.Main className={classNames.main}>
	  <Outlet />
        </AppShell.Main>
      </AppShell>
    </MantineProvider>
  );
}

export default App;
