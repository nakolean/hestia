import { useState, useEffect } from "preact/hooks";
import { Settings } from "lucide-preact";
import { get, post } from "../api/client";
import { ChoreCard, Chore } from "../components/ChoreCard";
import { CompletedAccordion } from "../components/CompletedAccordion";
import { subscribe } from "../lib/events";

export function ChoresTab({ path: _path }: { path?: string }) {
  const [chores, setChores] = useState<Chore[]>([]);
  const [loading, setLoading] = useState(true);

  const loadChores = async () => {
    try {
      const data = await get("/chores");
      setChores(data.chores as Chore[]);
    } catch {
      // Handle fetch error gracefully
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadChores();
  }, []);

  useEffect(() => {
    return subscribe("chore-added", loadChores);
  }, []);

  const handleComplete = async (id: number) => {
    try {
      await post(`/chores/${id}/complete`, {});
    } catch {
      alert("Failed to complete chore");
    }
    loadChores();
  };

  const active = chores.filter((c) => !c.completed);
  const completed = chores.filter((c) => c.completed);

  if (loading) return <p>Loading...</p>;

  return (
    <div>
      <div class="flex justify-between items-center p-2">
        <h2>Chores</h2>
        <button
          class="p-2 border border-current rounded bg-transparent text-hestia-text hover:bg-hestia-primary hover:text-white"
          onClick={() => window.dispatchEvent(new Event("open-settings-modal"))}
        >
          <Settings />
        </button>
      </div>
      {active.map((chore) => (
        <ChoreCard key={chore.id} chore={chore} onComplete={handleComplete} />
      ))}
      <CompletedAccordion completed={completed} />
    </div>
  );
}
