const BASE_URL = "/api";

export async function fetchAPI(
  path: string,
  options?: RequestInit,
): Promise<any> {
  const res = await fetch(`${BASE_URL}${path}`, {
    headers: { "Content-Type": "application/json" },
    ...options,
  });
  if (!res.ok) throw new Error(`API error: ${res.status}`);
  return res.json();
}

export function get(path: string): Promise<any> {
  return fetchAPI(path);
}

export async function post(path: string, body: unknown): Promise<any> {
  return fetchAPI(path, {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export async function put(path: string, body: unknown): Promise<any> {
  return fetchAPI(path, {
    method: "PUT",
    body: JSON.stringify(body),
  });
}

export async function del(path: string): Promise<void> {
  const res = await fetch(`${BASE_URL}${path}`, { method: "DELETE" });
  if (!res.ok) throw new Error(`API error: ${res.status}`);
}

export async function patch(path: string, body: unknown): Promise<any> {
  return fetchAPI(path, {
    method: "PATCH",
    body: JSON.stringify(body),
  });
}
