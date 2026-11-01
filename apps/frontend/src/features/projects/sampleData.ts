import type { Priority } from "@/features/dashboard/data";

export type SampleDeployStatus = "live" | "running" | "failed";

export const sampleNodes = ["Production", "Staging"] as const;

export type SampleApp = { id: string; name: string; node: (typeof sampleNodes)[number]; url: string; version: string; status: SampleDeployStatus; when: string };

export const sampleApps: SampleApp[] = [
  { id: "web-production", name: "Web", node: "Production", url: "app.example.com", version: "v2.13.4", status: "live", when: "2 days ago" },
  { id: "api-production", name: "API", node: "Production", url: "api.example.com", version: "v1.8.2", status: "live", when: "5 days ago" },
  { id: "worker-production", name: "Worker", node: "Production", url: "Background service", version: "v1.3.0", status: "live", when: "1 week ago" },
  { id: "web-staging", name: "Web", node: "Staging", url: "staging.example.com", version: "v2.14.0", status: "running", when: "now" },
  { id: "api-staging", name: "API", node: "Staging", url: "api.staging.example.com", version: "v1.9.0", status: "failed", when: "5h ago" },
];

export type SampleDeploy = { id: string; appId: string; version: string; message: string; author: string; duration: string; when: string; status: SampleDeployStatus };

export const sampleDeploys: SampleDeploy[] = [
  { id: "d9", appId: "web-staging", version: "v2.14.0", message: "Add saved filters to the reports page", author: "Ana", duration: "1m 12s", when: "now", status: "running" },
  { id: "d8", appId: "api-staging", version: "v1.9.0", message: "Update payment provider SDK", author: "Bruno", duration: "48s", when: "5h ago", status: "failed" },
  { id: "d7", appId: "web-production", version: "v2.13.4", message: "Fix session timeout on mobile", author: "Ana", duration: "2m 03s", when: "2 days ago", status: "live" },
  { id: "d6", appId: "web-staging", version: "v2.13.4", message: "Fix session timeout on mobile", author: "Ana", duration: "1m 41s", when: "2 days ago", status: "live" },
  { id: "d5", appId: "api-production", version: "v1.8.2", message: "Add rate limits to public routes", author: "Carla", duration: "1m 34s", when: "5 days ago", status: "live" },
  { id: "d4", appId: "web-production", version: "v2.13.3", message: "Improve dashboard loading time", author: "Carla", duration: "2m 18s", when: "5 days ago", status: "live" },
  { id: "d3", appId: "api-staging", version: "v1.8.2", message: "Add rate limits to public routes", author: "Carla", duration: "1m 22s", when: "6 days ago", status: "live" },
  { id: "d2", appId: "worker-production", version: "v1.3.0", message: "Retry failed email jobs", author: "Bruno", duration: "58s", when: "1 week ago", status: "live" },
  { id: "d1", appId: "web-production", version: "v2.13.2", message: "Add export to CSV", author: "Bruno", duration: "1m 56s", when: "1 week ago", status: "live" },
];

export type SampleTask = { id: string; title: string; assignee: string; due: string; dueOffset: number; late?: boolean; priority: Priority };

export const sampleColumns: { name: string; tasks: SampleTask[] }[] = [
  {
    name: "To do",
    tasks: [
      { id: "t1", title: "Write release notes for v2.14", assignee: "Carla", due: "In 3 days", dueOffset: 3, priority: "normal" },
      { id: "t2", title: "Add audit log to settings", assignee: "Bruno", due: "Next week", dueOffset: 8, priority: "normal" },
      { id: "t3", title: "Review empty states on mobile", assignee: "Ana", due: "Next week", dueOffset: 9, priority: "normal" },
    ],
  },
  {
    name: "In progress",
    tasks: [
      { id: "t4", title: "Fix payment provider upgrade", assignee: "Bruno", due: "2 days late", dueOffset: -2, late: true, priority: "urgent" },
      { id: "t5", title: "Saved filters on reports", assignee: "Ana", due: "Today", dueOffset: 0, priority: "high" },
    ],
  },
  {
    name: "In review",
    tasks: [{ id: "t6", title: "Rate limit public endpoints", assignee: "Carla", due: "Tomorrow", dueOffset: 1, priority: "high" }],
  },
  {
    name: "Done",
    tasks: [
      { id: "t7", title: "Session timeout on mobile", assignee: "Ana", due: "Done", dueOffset: -1, priority: "high" },
      { id: "t8", title: "Dashboard loading time", assignee: "Carla", due: "Done", dueOffset: -4, priority: "normal" },
    ],
  },
];
