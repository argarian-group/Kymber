import { TaskAssignee } from "@/features/projects/components/tasks/TaskAssignee";
import { type TaskPerson } from "@/features/projects/tasks";

export function TaskAssignees({ people }: { people: TaskPerson[] }) {
  return (
    <span className="flex flex-none -space-x-1">
      {people.map((person) => (
        <TaskAssignee name={person.username} key={person.id} />
      ))}
    </span>
  );
}
