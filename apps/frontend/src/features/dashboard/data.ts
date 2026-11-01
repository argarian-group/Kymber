import { type Priority } from "@/features/dashboard/priorities";

export type Health = "on-track" | "at-risk" | "late";
export type DeployStatus = "live" | "running" | "failed";

export type Project = {
  name: string;
  phase: string;
  progress: number;
  health: Health;
  tasksLeft: number;
  nextDeploy: string;
};

export type Stage = { name: string; count: number };

export type Deploy = {
  project: string;
  version: string;
  environment: string;
  when: string;
  status: DeployStatus;
};

export type DueTask = { title: string; project: string; due: string; late: boolean; priority: Priority };

export const projects: Project[] = [
  { name: "Customer portal", phase: "Staging", progress: 82, health: "on-track", tasksLeft: 9, nextDeploy: "Today, 6:00 PM" },
  { name: "Mobile app", phase: "Development", progress: 54, health: "at-risk", tasksLeft: 31, nextDeploy: "Fri, 10:00 AM" },
  { name: "Payments API", phase: "Development", progress: 38, health: "late", tasksLeft: 27, nextDeploy: "TBD" },
  { name: "Company website", phase: "Final review", progress: 95, health: "on-track", tasksLeft: 2, nextDeploy: "Tomorrow, 9:00 AM" },
];

export const stages: Stage[] = [
  { name: "To do", count: 28 },
  { name: "In progress", count: 23 },
  { name: "In review", count: 18 },
  { name: "Done", count: 64 },
];

export const deploys: Deploy[] = [
  { project: "Customer portal", version: "v2.14.0", environment: "Staging", when: "now", status: "running" },
  { project: "Company website", version: "v1.9.2", environment: "Production", when: "2h ago", status: "live" },
  { project: "Payments API", version: "v0.8.1", environment: "Staging", when: "5h ago", status: "failed" },
  { project: "Mobile app", version: "v3.2.0", environment: "Staging", when: "yesterday", status: "live" },
];

export const dueTasks: DueTask[] = [
  { title: "Review checkout flow", project: "Payments API", due: "2 days late", late: true, priority: "urgent" },
  { title: "Fix SSO login", project: "Mobile app", due: "Today", late: false, priority: "high" },
  { title: "Check homepage copy", project: "Company website", due: "Today", late: false, priority: "normal" },
  { title: "Publish release notes", project: "Customer portal", due: "Tomorrow", late: false, priority: "normal" },
];
