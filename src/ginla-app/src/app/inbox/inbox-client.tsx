"use client";

import { useEffect, useState, useCallback } from "react";
import { fetchTasks, updateTask } from "@/lib/api";
import type { Task, Tag } from "@/types/task";
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
import { Checkbox } from "@/components/ui/checkbox";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const TAGS: Tag[] = ["ME", "AI", "VA", "FAMILY", "HOUSEKEEPER", "DELEGATE"];

interface TriageState {
  tag: Tag | "";
  handler_id: string;
}

export function InboxPageClient() {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [taskTriage, setTaskTriage] = useState<Record<string, TriageState>>({});
  const [triaging, setTriaging] = useState<Set<string>>(new Set());
  const [bulkTriage, setBulkTriage] = useState<TriageState>({
    tag: "",
    handler_id: "",
  });
  const [bulkTriaging, setBulkTriaging] = useState(false);

  const loadTasks = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await fetchTasks({ status: "inbox", limit: 50 });
      setTasks(result.tasks ?? []);
      setTotal(result.total ?? 0);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load inbox");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadTasks();
  }, [loadTasks]);

  function getTaskTriage(id: string): TriageState {
    return taskTriage[id] ?? { tag: "", handler_id: "" };
  }

  function setTaskTriageField(
    id: string,
    field: keyof TriageState,
    value: string
  ) {
    setTaskTriage((prev) => ({
      ...prev,
      [id]: { ...getTaskTriage(id), [field]: value },
    }));
  }

  async function handleTriage(task: Task) {
    const triage = getTaskTriage(task.id);
    if (!triage.tag) return;

    setTriaging((prev) => new Set(prev).add(task.id));
    try {
      await updateTask(task.id, {
        tag: triage.tag as Tag,
        handler_id: triage.handler_id || undefined,
        status: "pending",
      });
      await loadTasks();
    } catch (err) {
      console.error("Triage failed:", err);
    } finally {
      setTriaging((prev) => {
        const next = new Set(prev);
        next.delete(task.id);
        return next;
      });
    }
  }

  async function handleBulkTriage() {
    if (!bulkTriage.tag || selected.size === 0) return;

    setBulkTriaging(true);
    try {
      await Promise.all(
        Array.from(selected).map((id) =>
          updateTask(id, {
            tag: bulkTriage.tag as Tag,
            handler_id: bulkTriage.handler_id || undefined,
            status: "pending",
          })
        )
      );
      setSelected(new Set());
      setBulkTriage({ tag: "", handler_id: "" });
      await loadTasks();
    } catch (err) {
      console.error("Bulk triage failed:", err);
    } finally {
      setBulkTriaging(false);
    }
  }

  function toggleSelect(id: string) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  }

  function toggleSelectAll() {
    if (selected.size === tasks.length) {
      setSelected(new Set());
    } else {
      setSelected(new Set(tasks.map((t) => t.id)));
    }
  }

  return (
    <div className="p-8">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-semibold">Inbox</h1>
          <p className="text-sm text-muted-foreground mt-1">
            {total} item{total !== 1 ? "s" : ""} to triage
          </p>
        </div>
      </div>

      {/* Bulk triage bar */}
      {selected.size > 0 && (
        <div className="flex items-center gap-3 mb-4 p-3 bg-muted rounded-md">
          <span className="text-sm font-medium">
            {selected.size} selected
          </span>
          <Select
            value={bulkTriage.tag || "none"}
            onValueChange={(v) =>
              setBulkTriage({
                ...bulkTriage,
                tag: v === "none" ? "" : (v as Tag),
              })
            }
          >
            <SelectTrigger className="w-40">
              <SelectValue placeholder="Select tag" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="none">Select tag</SelectItem>
              {TAGS.map((t) => (
                <SelectItem key={t} value={t}>
                  {t}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button
            size="sm"
            disabled={!bulkTriage.tag || bulkTriaging}
            onClick={handleBulkTriage}
          >
            {bulkTriaging ? "Triaging..." : "Triage Selected"}
          </Button>
          <Button
            size="sm"
            variant="ghost"
            onClick={() => setSelected(new Set())}
          >
            Clear selection
          </Button>
        </div>
      )}

      {error && (
        <div className="mb-4 p-3 bg-destructive/10 text-destructive rounded-md text-sm">
          {error}
        </div>
      )}

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-10">
                <Checkbox
                  checked={selected.size === tasks.length && tasks.length > 0}
                  onCheckedChange={toggleSelectAll}
                  aria-label="Select all"
                />
              </TableHead>
              <TableHead>Title</TableHead>
              <TableHead>Preview</TableHead>
              <TableHead>Source</TableHead>
              <TableHead>Created</TableHead>
              <TableHead className="w-48">Tag</TableHead>
              <TableHead className="w-24">Action</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              <TableRow>
                <TableCell
                  colSpan={7}
                  className="text-center py-8 text-muted-foreground"
                >
                  Loading...
                </TableCell>
              </TableRow>
            ) : tasks.length === 0 ? (
              <TableRow>
                <TableCell
                  colSpan={7}
                  className="text-center py-8 text-muted-foreground"
                >
                  Inbox is empty
                </TableCell>
              </TableRow>
            ) : (
              tasks.map((task) => {
                const triage = getTaskTriage(task.id);
                const isTriaging = triaging.has(task.id);
                return (
                  <TableRow
                    key={task.id}
                    className={selected.has(task.id) ? "bg-muted/30" : ""}
                  >
                    <TableCell>
                      <Checkbox
                        checked={selected.has(task.id)}
                        onCheckedChange={() => toggleSelect(task.id)}
                        aria-label={`Select ${task.title}`}
                      />
                    </TableCell>
                    <TableCell className="font-medium max-w-xs">
                      <span className="line-clamp-1">{task.title}</span>
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground max-w-sm">
                      <span className="line-clamp-2">
                        {task.description || "—"}
                      </span>
                    </TableCell>
                    <TableCell>
                      {task.source ? (
                        <Badge variant="outline">{task.source}</Badge>
                      ) : (
                        <span className="text-muted-foreground text-xs">—</span>
                      )}
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground whitespace-nowrap">
                      {new Date(task.created_at).toLocaleDateString()}
                    </TableCell>
                    <TableCell>
                      <Select
                        value={triage.tag || "none"}
                        onValueChange={(v) =>
                          setTaskTriageField(
                            task.id,
                            "tag",
                            v === "none" ? "" : v
                          )
                        }
                      >
                        <SelectTrigger className="w-36">
                          <SelectValue placeholder="Select tag" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="none">Select tag</SelectItem>
                          {TAGS.map((t) => (
                            <SelectItem key={t} value={t}>
                              {t}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </TableCell>
                    <TableCell>
                      <Button
                        size="sm"
                        disabled={!triage.tag || isTriaging}
                        onClick={() => handleTriage(task)}
                      >
                        {isTriaging ? "..." : "Triage"}
                      </Button>
                    </TableCell>
                  </TableRow>
                );
              })
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
