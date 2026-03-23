"use client";

import { useEffect, useState, useCallback } from "react";
import { fetchTasks, fetchTask } from "@/lib/api";
import type { Task, TaskFilter, Tag, Status, Priority } from "@/types/task";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { TaskDetailSheet } from "./task-detail-sheet";
import { CreateTaskDialog } from "./create-task-dialog";
import { ChevronUp, ChevronDown } from "lucide-react";

const TAGS: Tag[] = ["ME", "AI", "VA", "FAMILY", "HOUSEKEEPER", "DELEGATE"];
const STATUSES: Status[] = ["inbox", "pending", "active", "done", "cancelled"];
const PRIORITIES: Priority[] = ["urgent", "high", "normal", "low"];

const statusColors: Record<Status, string> = {
  inbox: "secondary",
  pending: "outline",
  active: "default",
  done: "default",
  cancelled: "destructive",
};

const priorityColors: Record<Priority, string> = {
  urgent: "destructive",
  high: "default",
  normal: "secondary",
  low: "outline",
};

export function TasksPageClient() {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filters, setFilters] = useState<TaskFilter>({ page: 1, limit: 20 });
  const [sortField, setSortField] = useState<string>("created_at");
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("desc");
  const [selectedTask, setSelectedTask] = useState<Task | null>(null);
  const [sheetOpen, setSheetOpen] = useState(false);

  const loadTasks = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await fetchTasks({
        ...filters,
        sort: sortField,
        order: sortOrder,
      });
      setTasks(result.tasks ?? []);
      setTotal(result.total ?? 0);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load tasks");
    } finally {
      setLoading(false);
    }
  }, [filters, sortField, sortOrder]);

  useEffect(() => {
    loadTasks();
  }, [loadTasks]);

  function handleSort(field: string) {
    if (sortField === field) {
      setSortOrder(sortOrder === "asc" ? "desc" : "asc");
    } else {
      setSortField(field);
      setSortOrder("asc");
    }
  }

  async function handleRowClick(task: Task) {
    const full = await fetchTask(task.id).catch(() => task);
    setSelectedTask(full);
    setSheetOpen(true);
  }

  function setFilter<K extends keyof TaskFilter>(key: K, value: TaskFilter[K]) {
    setFilters((prev) => ({ ...prev, [key]: value, page: 1 }));
  }

  function clearFilter(key: keyof TaskFilter) {
    setFilters((prev) => {
      const next = { ...prev };
      delete next[key];
      next.page = 1;
      return next;
    });
  }

  const page = filters.page ?? 1;
  const limit = filters.limit ?? 20;
  const totalPages = Math.ceil(total / limit);

  function SortIcon({ field }: { field: string }) {
    if (sortField !== field) return null;
    return sortOrder === "asc" ? (
      <ChevronUp className="inline h-3 w-3 ml-1" />
    ) : (
      <ChevronDown className="inline h-3 w-3 ml-1" />
    );
  }

  return (
    <div className="p-8">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-semibold">Tasks</h1>
          <p className="text-sm text-muted-foreground mt-1">
            {total} task{total !== 1 ? "s" : ""} total
          </p>
        </div>
        <CreateTaskDialog onCreated={loadTasks} />
      </div>

      {/* Filters */}
      <div className="flex gap-3 mb-4 flex-wrap">
        <Select
          value={filters.tag ?? "all"}
          onValueChange={(v) =>
            v === "all" ? clearFilter("tag") : setFilter("tag", v as Tag)
          }
        >
          <SelectTrigger className="w-40">
            <SelectValue placeholder="Tag" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All tags</SelectItem>
            {TAGS.map((t) => (
              <SelectItem key={t} value={t}>
                {t}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Select
          value={filters.status ?? "all"}
          onValueChange={(v) =>
            v === "all"
              ? clearFilter("status")
              : setFilter("status", v as Status)
          }
        >
          <SelectTrigger className="w-40">
            <SelectValue placeholder="Status" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All statuses</SelectItem>
            {STATUSES.map((s) => (
              <SelectItem key={s} value={s}>
                {s}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Select
          value={filters.priority ?? "all"}
          onValueChange={(v) =>
            v === "all"
              ? clearFilter("priority")
              : setFilter("priority", v as Priority)
          }
        >
          <SelectTrigger className="w-40">
            <SelectValue placeholder="Priority" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All priorities</SelectItem>
            {PRIORITIES.map((p) => (
              <SelectItem key={p} value={p}>
                {p}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {error && (
        <div className="mb-4 p-3 bg-destructive/10 text-destructive rounded-md text-sm">
          {error}
        </div>
      )}

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead
                className="cursor-pointer"
                onClick={() => handleSort("title")}
              >
                Title <SortIcon field="title" />
              </TableHead>
              <TableHead>Tag</TableHead>
              <TableHead
                className="cursor-pointer"
                onClick={() => handleSort("status")}
              >
                Status <SortIcon field="status" />
              </TableHead>
              <TableHead
                className="cursor-pointer"
                onClick={() => handleSort("priority")}
              >
                Priority <SortIcon field="priority" />
              </TableHead>
              <TableHead>Handler</TableHead>
              <TableHead
                className="cursor-pointer"
                onClick={() => handleSort("due")}
              >
                Due <SortIcon field="due" />
              </TableHead>
              <TableHead
                className="cursor-pointer"
                onClick={() => handleSort("created_at")}
              >
                Created <SortIcon field="created_at" />
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              <TableRow>
                <TableCell colSpan={7} className="text-center py-8 text-muted-foreground">
                  Loading...
                </TableCell>
              </TableRow>
            ) : tasks.length === 0 ? (
              <TableRow>
                <TableCell colSpan={7} className="text-center py-8 text-muted-foreground">
                  No tasks found
                </TableCell>
              </TableRow>
            ) : (
              tasks.map((task) => (
                <TableRow
                  key={task.id}
                  className="cursor-pointer hover:bg-muted/50"
                  onClick={() => handleRowClick(task)}
                >
                  <TableCell className="font-medium max-w-xs truncate">
                    {task.title}
                  </TableCell>
                  <TableCell>
                    {task.tag ? (
                      <Badge variant="outline">{task.tag}</Badge>
                    ) : (
                      <span className="text-muted-foreground text-xs">—</span>
                    )}
                  </TableCell>
                  <TableCell>
                    <Badge
                      variant={
                        (statusColors[task.status] as
                          | "default"
                          | "secondary"
                          | "destructive"
                          | "outline") ?? "secondary"
                      }
                    >
                      {task.status}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <Badge
                      variant={
                        (priorityColors[task.priority] as
                          | "default"
                          | "secondary"
                          | "destructive"
                          | "outline") ?? "secondary"
                      }
                    >
                      {task.priority}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm">
                    {task.handler_id ? task.handler_id.slice(-6) : "—"}
                  </TableCell>
                  <TableCell className="text-sm">
                    {task.due
                      ? new Date(task.due).toLocaleDateString()
                      : <span className="text-muted-foreground">—</span>}
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {new Date(task.created_at).toLocaleDateString()}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between mt-4">
          <p className="text-sm text-muted-foreground">
            Page {page} of {totalPages}
          </p>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              disabled={page <= 1}
              onClick={() => setFilter("page", page - 1)}
            >
              Previous
            </Button>
            <Button
              variant="outline"
              size="sm"
              disabled={page >= totalPages}
              onClick={() => setFilter("page", page + 1)}
            >
              Next
            </Button>
          </div>
        </div>
      )}

      {selectedTask && (
        <TaskDetailSheet
          task={selectedTask}
          open={sheetOpen}
          onOpenChange={setSheetOpen}
          onUpdated={loadTasks}
        />
      )}
    </div>
  );
}
