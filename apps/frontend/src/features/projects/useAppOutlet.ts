import { useOutletContext } from "react-router";
import type { DatabaseTemplate, Service } from "./services";

export type AppOutlet = {
  service: Service;
  detail: Service | null;
  template: DatabaseTemplate | null;
  path: string;
  canWrite: boolean;
  appsPath: string;
  onChanged: () => void;
};

export function useAppOutlet() {
  return useOutletContext<AppOutlet>();
}
