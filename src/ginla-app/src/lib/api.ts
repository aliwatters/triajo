import type {
  Task,
  TaskFilter,
  TasksResponse,
  CreateTaskInput,
  UpdateTaskInput,
} from "@/types/task";

const BASE_URL =
  process.env.NEXT_PUBLIC_GINLA_API_URL ??
  process.env.GINLA_API_URL ??
  "http://localhost:8080";

async function apiFetch<T>(
  path: string,
  options?: RequestInit
): Promise<T> {
  const url = `${BASE_URL}${path}`;
  const res = await fetch(url, {
    headers: {
      "Content-Type": "application/json",
      ...options?.headers,
    },
    ...options,
  });

  if (!res.ok) {
    const text = await res.text();
    throw new Error(`API error ${res.status}: ${text}`);
  }

  return res.json() as Promise<T>;
}

export async function fetchTasks(filters: TaskFilter = {}): Promise<TasksResponse> {
  const params = new URLSearchParams();
  if (filters.tag) params.set("tag", filters.tag);
  if (filters.status) params.set("status", filters.status);
  if (filters.priority) params.set("priority", filters.priority);
  if (filters.handler_id) params.set("handler_id", filters.handler_id);
  if (filters.page) params.set("page", String(filters.page));
  if (filters.limit) params.set("limit", String(filters.limit));
  if (filters.sort) params.set("sort", filters.sort);
  if (filters.order) params.set("order", filters.order);

  const query = params.toString();
  return apiFetch<TasksResponse>(`/v1/tasks${query ? `?${query}` : ""}`);
}

export async function fetchTask(id: string): Promise<Task> {
  return apiFetch<Task>(`/v1/tasks/${id}`);
}

export async function createTask(data: CreateTaskInput): Promise<Task> {
  return apiFetch<Task>("/v1/tasks", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function updateTask(
  id: string,
  data: UpdateTaskInput
): Promise<Task> {
  return apiFetch<Task>(`/v1/tasks/${id}`, {
    method: "PATCH",
    body: JSON.stringify(data),
  });
}
