import { t } from "@/lib/i18n";

export type GitProviderKind = "github_app" | "github_token" | "gitlab" | "bitbucket" | "git_ssh";

export type GitProvider = {
  id: string;
  kind: GitProviderKind;
  name: string;
  base_url?: string;
  app_slug?: string;
  public_key?: string;
  install_url?: string;
  ready: boolean;
  installations?: { id: number; account_login: string }[];
  created_at: string;
};

export function providerKindLabel(kind: GitProviderKind) {
  return {
    github_app: t("GitHub App"),
    github_token: t("GitHub token"),
    gitlab: "GitLab",
    bitbucket: "Bitbucket",
    git_ssh: t("Git over SSH"),
  }[kind];
}

export function providerHost(provider: GitProvider | undefined) {
  if (!provider || provider.kind === "github_app" || provider.kind === "github_token") return "github.com";
  if (provider.kind === "bitbucket") return "bitbucket.org";
  if (provider.kind === "gitlab") return (provider.base_url ?? "https://gitlab.com").replace(/^https:\/\//, "");
  return provider.base_url ?? "";
}
