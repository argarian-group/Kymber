import { AuthGate } from "@/AuthGate";
import { AuthProvider } from "@/features/auth/AuthContext";

export function App() {
  return (
    <AuthProvider>
      <AuthGate />
    </AuthProvider>
  );
}
