import { useOutletContext } from "react-router";
import type { DatabaseTemplate, Service } from "./services";

export type AppOutlet = {
  service: Service;
  detail: Service | null;
  template: DatabaseTemplate | null;
  path: string;
  canWrite: boolean;
  canTerminal: boolean;
  appsPath: string;
  onChanged: () => void;
  deployOutput: string | null;
};

export function useAppOutlet() {
  return useOutletContext<AppOutlet>();
}
