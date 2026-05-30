import { useLocation } from "preact-iso";

export function BottomNav() {
  const { path } = useLocation();

  return (
    <nav class="bottom-nav">
      <a href="/" class={path === "/" ? "active" : ""}>
        Chores
      </a>
      <a href="/shopping" class={path === "/shopping" ? "active" : ""}>
        List
      </a>
    </nav>
  );
}
