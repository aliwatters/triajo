export type Tag = "ME" | "AI" | "VA" | "FAMILY" | "HOUSEKEEPER" | "DELEGATE";
export type Status = "inbox" | "pending" | "active" | "done" | "cancelled";
export type Priority = "urgent" | "high" | "normal" | "low";
export type Source =
  | "manual"
  | "agent"
  | "email"
  | "calendar"
  | "voice"
  | "screenshot";

export interface ChecklistItem {
  text: string;
  done: boolean;
}

export interface Recurrence {
  rrule: string;
  next_at: string;
}

export interface Attachment {
  url: string;
  name: string;
  type: string;
}

export interface ActivityEntry {
  action: string;
  by: string;
  at: string;
  detail?: string;
}

export interface Task {
  id: string;
  household_id: string;
  title: string;
  description: string;
  checklist: ChecklistItem[];
  tag?: Tag;
  handler_id?: string;
  status: Status;
  priority: Priority;
  position?: number;
  due?: string;
  source?: Source;
  meta: Record<string, unknown>;
  parent_id?: string;
  recurrence?: Recurrence;
  attachments: Attachment[];
  activity: ActivityEntry[];
  created_at: string;
  updated_at: string;
  done_at?: string;
}

export interface TaskFilter {
  tag?: Tag;
  status?: Status;
  priority?: Priority;
  handler_id?: string;
  page?: number;
  limit?: number;
  sort?: string;
  order?: "asc" | "desc";
}

export interface TasksResponse {
  tasks: Task[];
  total: number;
  page: number;
  limit: number;
}

export interface CreateTaskInput {
  title: string;
  description?: string;
  tag?: Tag;
  priority?: Priority;
  due?: string;
  handler_id?: string;
}

export interface UpdateTaskInput {
  title?: string;
  description?: string;
  tag?: Tag;
  status?: Status;
  priority?: Priority;
  handler_id?: string;
  due?: string;
}
