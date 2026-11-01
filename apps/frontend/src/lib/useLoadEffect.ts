import { useEffect, type DependencyList } from "react";
import { errorMessage } from "./api";

export function useLoadEffect<T>(load: (isCurrent: () => boolean) => Promise<T>, onSuccess: (value: T) => void, onError: (message: string) => void, deps: DependencyList) {
  useEffect(() => {
    let current = true;
    load(() => current)
      .then((value) => {
        if (current) onSuccess(value);
      })
      .catch((error) => {
        if (current) onError(errorMessage(error));
      });
    return () => {
      current = false;
    };
  }, deps);
}
