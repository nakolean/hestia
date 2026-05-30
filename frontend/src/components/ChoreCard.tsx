export interface Chore {
  id: number;
  name: string;
  description: string;
  frequency_num: number;
  frequency_unit: string;
  completed: boolean;
  last_completed: string | null;
  next_due: string | null;
  created_at: string;
  updated_at: string;
}

export function getStatus(chore: Chore): "overdue" | "due_soon" | "on_track" {
  if (!chore.next_due) return "on_track";
  const due = new Date(chore.next_due);
  const now = new Date();
  const diffMs = due.getTime() - now.getTime();

  if (diffMs < 0) return "overdue";
  if (diffMs < 24 * 60 * 60 * 1000) return "due_soon";
  return "on_track";
}

export function formatCountdown(chore: Chore): string {
  if (!chore.next_due) return "no due date";
  const due = new Date(chore.next_due);
  const now = new Date();
  const diffMs = due.getTime() - now.getTime();
  const absMs = Math.abs(diffMs);

  if (diffMs > 0) {
    const days = Math.floor(absMs / (1000 * 60 * 60 * 24));
    const hours = Math.floor(absMs / (1000 * 60 * 60)) % 24;
    if (days > 0) return `in ${days}d ${hours}h`;
    return `in ${hours}h`;
  }
  const overdueHours = Math.floor(absMs / (1000 * 60 * 60));
  if (overdueHours < 24) return `${overdueHours}h overdue`;
  const days = Math.floor(absMs / (1000 * 60 * 60 * 24));
  return `${days}d overdue`;
}

interface Props {
  chore: Chore;
  onComplete: (id: number) => void;
}

export function ChoreCard({ chore, onComplete }: Props) {
  const status = getStatus(chore);
  const statusColors: Record<string, string> = {
    overdue: "#e53935",
    due_soon: "#f9a825",
    on_track: "#43a047",
  };

  return (
    <div
      class="border-l-4 rounded p-4 mb-2"
      style={{ borderLeft: `4px solid ${statusColors[status]}` }}
    >
      <div class="flex justify-between items-center">
        <div>
          <h3 class="m-0 text-lg font-semibold">{chore.name}</h3>
          {chore.description && (
            <p class="text-hestia-text-muted text-sm m-0">
              {chore.description}
            </p>
          )}
          <p class="text-hestia-text-muted text-xs m-0">
            {formatCountdown(chore)} · {chore.frequency_num}{" "}
            {chore.frequency_unit}
          </p>
        </div>
        <button
          class="px-4 py-2 border border-current rounded bg-transparent hover:bg-hestia-primary hover:text-white"
          onClick={() => onComplete(chore.id)}
        >
          Complete
        </button>
      </div>
    </div>
  );
}
