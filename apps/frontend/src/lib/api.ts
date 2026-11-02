import { t, translate } from "@/lib/i18n";
export class ApiError extends Error {
  constructor(
    readonly status: number,
    message: string,
  ) {
    super(message);
  }
}

export const unauthorizedEvent = "kymber:unauthorized";

export async function api<T>(path: string, options: { method?: string; body?: unknown; file?: Blob } = {}): Promise<T> {
  let response: Response;
  try {
    response = await fetch(`/api${path}`, {
      method: options.method ?? "GET",
      headers: options.body === undefined ? undefined : { "Content-Type": "application/json" },
      body: options.file ?? (options.body === undefined ? undefined : JSON.stringify(options.body)),
    });
  } catch {
    throw new ApiError(0, t("Could not reach the server. Check your connection and try again."));
  }

  if (response.status === 204) return undefined as T;
  const data = await response.json().catch(() => null);
  if (!response.ok) {
    if (response.status === 401) window.dispatchEvent(new Event(unauthorizedEvent));
    throw new ApiError(response.status, typeof data?.error === "string" ? translate(data.error) : t("The server could not complete this request. Try again."));
  }
  return data as T;
}

export function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : t("Something went wrong. Try again.");
}
