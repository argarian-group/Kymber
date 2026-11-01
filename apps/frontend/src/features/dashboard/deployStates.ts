import { CheckCircle2, CircleAlert, LoaderCircle, type LucideIcon } from "lucide-react";
import { type DeployStatus } from "@/features/dashboard/data";

export const deployStates: Record<DeployStatus, { label: string; icon: LucideIcon; iconTone: string; labelTone: string }> = {
  running: { label: "Deploying", icon: LoaderCircle, iconTone: "text-seafoam animate-deploying motion-reduce:animate-none", labelTone: "font-medium text-ink" },
  failed: { label: "Failed", icon: CircleAlert, iconTone: "text-ember", labelTone: "font-medium text-ember" },
  live: { label: "Live", icon: CheckCircle2, iconTone: "text-ink-faint", labelTone: "text-ink-soft" },
};
