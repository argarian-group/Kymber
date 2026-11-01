import { useRef, useState, type ChangeEvent } from "react";
import { FormError } from "@/components/ui/FormError";
import { ProjectMark } from "@/features/projects/components/ProjectMark";
import { useProjects } from "@/features/projects/ProjectsContext";
import { useProjectOutlet } from "@/features/projects/useProjectOutlet";
import { SettingsSection } from "@/features/settings/components/SettingsSection";
import { useAction } from "@/lib/useAction";

const maxBytes = 5 * 1024 * 1024;
const accepted = "image/png,image/jpeg,image/webp,image/gif,image/avif,image/svg+xml,image/x-icon,image/vnd.microsoft.icon,image/bmp,.ico,.svg,.avif";

export function ProjectLogoForm() {
  const { project } = useProjectOutlet();
  const { uploadProjectLogo, removeProjectLogo } = useProjects();
  const inputRef = useRef<HTMLInputElement>(null);
  const [tooLarge, setTooLarge] = useState(false);
  const { pending, error, run } = useAction();

  async function choose(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) return;
    setTooLarge(file.size > maxBytes);
    if (file.size <= maxBytes) await run(() => uploadProjectLogo(project.id, file));
  }

  return (
    <SettingsSection title="Logo" description="PNG, JPEG, WebP, GIF, AVIF, SVG, ICO or BMP, up to 5 MB.">
      <div className="flex flex-wrap items-center gap-4 p-4">
        <ProjectMark project={project} />
        <input className="sr-only" ref={inputRef} type="file" accept={accepted} aria-label="Project logo file" onChange={choose} />
        <div className="flex flex-wrap gap-2">
          <button className="btn" type="button" disabled={pending} onClick={() => inputRef.current?.click()}>
            {pending ? "Saving…" : project.logo_version ? "Replace logo" : "Upload logo"}
          </button>
          {project.logo_version && (
            <button className="btn" type="button" disabled={pending} onClick={() => run(() => removeProjectLogo(project.id))}>
              Remove
            </button>
          )}
        </div>
      </div>
      {(tooLarge || error) && (
        <div className="border-t border-line px-4 py-3">
          <FormError message={tooLarge ? "The logo must be 5 MB or smaller." : error} />
        </div>
      )}
    </SettingsSection>
  );
}
