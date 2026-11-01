import type { Priority } from "@/features/dashboard/data";

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
