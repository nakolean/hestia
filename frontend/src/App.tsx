import { LocationProvider, ErrorBoundary, Router } from "preact-iso";
import { BottomNav } from "./components/BottomNav";
import { FloatingActionButton } from "./components/FloatingActionButton";
import { ChoresTab } from "./pages/ChoresTab";
import { ShoppingTab } from "./pages/ShoppingTab";
import { SettingsModal } from "./pages/SettingsTab";
import { LoginScreen } from "./pages/LoginPage.tsx";

import { useState, useEffect } from "preact/hooks";

type AuthState = { loggedIn: boolean; loading: boolean };

function AuthCheck({ children }: { children: any }) {
  const [auth, setAuth] = useState<AuthState>({
    loggedIn: false,
    loading: true,
  });

  useEffect(() => {
    fetch("/api/chores")
      .then((r) => {
        if (r.ok) {
          setAuth({ loggedIn: true, loading: false });
        } else {
          setAuth({ loggedIn: false, loading: false });
        }
      })
      .catch(() => setAuth({ loggedIn: false, loading: false }));
  }, []);

  if (auth.loading) {
    return (
      <div class="flex items-center justify-center h-screen">Loading...</div>
    );
  }

  if (!auth.loggedIn) {
    return <LoginScreen />;
  }

  return children;
}

function AppShell() {
  const [settingsOpen, setSettingsOpen] = useState(false);

  useEffect(() => {
    const open = () => setSettingsOpen(true);
    const close = () => setSettingsOpen(false);
    window.addEventListener("open-settings-modal", open);
    window.addEventListener("close-settings-modal", close);
    return () => {
      window.removeEventListener("open-settings-modal", open);
      window.removeEventListener("close-settings-modal", close);
    };
  }, []);

  return (
    <div class="app-shell">
      <main>
        <Router>
          <ChoresTab path="/" />
          <ShoppingTab path="/shopping" />
        </Router>
      </main>
      <BottomNav />
      <FloatingActionButton />
      {settingsOpen && <SettingsModal onClose={() => setSettingsOpen(false)} />}
    </div>
  );
}

export function App() {
  return (
    <LocationProvider>
      <ErrorBoundary>
        <AuthCheck>
          <AppShell />
        </AuthCheck>
      </ErrorBoundary>
    </LocationProvider>
  );
}
