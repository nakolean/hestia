import { useState } from "preact/hooks";
import { Chore } from "./ChoreCard";

interface Props {
  completed: Chore[];
  onUndo?: (id: number) => void;
}

export function CompletedAccordion({ completed, onUndo }: Props) {
  const [expanded, setExpanded] = useState(false);

  if (!completed.length) return null;

  return (
    <section class="mt-6">
      <details
        open={expanded}
        onToggle={(e) => setExpanded((e.target as HTMLDetailsElement).open)}
      >
        <summary class="cursor-pointer p-2 font-semibold">
          Completed ({completed.length})
        </summary>
        <div>
          {completed.map((chore) => (
            <div
              key={chore.id}
              class="grid grid-cols-2 gap-2 p-2 border-t border-hestia-border opacity-60"
            >
              <span>{chore.name}</span>
              <small>
                {chore.last_completed
                  ? new Date(chore.last_completed).toLocaleString()
                  : "-"}
              </small>
              {onUndo && (
                <button
                  class="text-xs px-2 py-1 border border-current rounded bg-transparent hover:bg-hestia-primary hover:text-white"
                  onClick={() => onUndo(chore.id)}
                >
                  Undo
                </button>
              )}
            </div>
          ))}
        </div>
      </details>
    </section>
  );
}
