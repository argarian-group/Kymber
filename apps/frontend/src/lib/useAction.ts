import { useCallback, useState } from "react";
import { errorMessage } from "./api";

export function useAction() {
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const run = useCallback(async (action: () => Promise<void>) => {
    setPending(true);
    setError(null);
    try {
      await action();
      setPending(false);
      return true;
    } catch (failure) {
      setError(errorMessage(failure));
      setPending(false);
      return false;
    }
  }, []);

  return { pending, error, setError, run };
}
