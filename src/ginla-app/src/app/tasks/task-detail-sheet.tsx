"use client";

import { useState } from "react";
import { updateTask } from "@/lib/api";
import type { Task, Tag, Status, Priority } from "@/types/task";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from "@/components/ui/sheet";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";

interface Props {
  task: Task;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onUpdated: () => void;
}

const TAGS: Tag[] = ["ME", "AI", "VA", "FAMILY", "HOUSEKEEPER", "DELEGATE"];
const STATUSES: Status[] = ["inbox", "pending", "active", "done", "cancelled"];
const PRIORITIES: Priority[] = ["urgent", "high", "normal", "low"];

export function TaskDetailSheet({ task, open, onOpenChange, onUpdated }: Props) {
  const [saving, setSaving] = useState(false);
  const [form, setForm] = useState({
    status: task.status,
    tag: task.tag ?? ("" as Tag | ""),
    priority: task.priority,
    handler_id: task.handler_id ?? "",
    due: task.due ? task.due.split("T")[0] : "",
  });

  async function handleSave() {
    setSaving(true);
    try {
      await updateTask(task.id, {
        status: form.status,
        tag: form.tag ? (form.tag as Tag) : undefined,
        priority: form.priority,
        handler_id: form.handler_id || undefined,
        due: form.due ? new Date(form.due).toISOString() : undefined,
      });
      onUpdated();
      onOpenChange(false);
    } catch (err) {
      console.error("Failed to update task:", err);
    } finally {
      setSaving(false);
    }
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="w-full sm:max-w-xl overflow-y-auto">
        <SheetHeader>
          <SheetTitle className="pr-8">{task.title}</SheetTitle>
          {task.description && (
            <SheetDescription className="text-sm text-foreground/80 whitespace-pre-wrap">
              {task.description}
            </SheetDescription>
          )}
        </SheetHeader>

        <div className="mt-6 space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label>Status</Label>
              <Select
                value={form.status}
                onValueChange={(v) => setForm({ ...form, status: v as Status })}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {STATUSES.map((s) => (
                    <SelectItem key={s} value={s}>
                      {s}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-1.5">
              <Label>Tag</Label>
              <Select
                value={form.tag || "none"}
                onValueChange={(v) =>
                  setForm({ ...form, tag: v === "none" ? "" : (v as Tag) })
                }
              >
                <SelectTrigger>
                  <SelectValue placeholder="No tag" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="none">No tag</SelectItem>
                  {TAGS.map((t) => (
                    <SelectItem key={t} value={t}>
                      {t}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-1.5">
              <Label>Priority</Label>
              <Select
                value={form.priority}
                onValueChange={(v) =>
                  setForm({ ...form, priority: v as Priority })
                }
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {PRIORITIES.map((p) => (
                    <SelectItem key={p} value={p}>
                      {p}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-1.5">
              <Label>Due date</Label>
              <Input
                type="date"
                value={form.due}
                onChange={(e) => setForm({ ...form, due: e.target.value })}
              />
            </div>
          </div>

          <div className="flex gap-2 pt-2">
            <Button onClick={handleSave} disabled={saving}>
              {saving ? "Saving..." : "Save changes"}
            </Button>
            <Button variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
          </div>
        </div>

        {/* Metadata */}
        <Separator className="my-6" />
        <div className="space-y-2 text-sm text-muted-foreground">
          <div className="flex gap-2">
            <span className="font-medium text-foreground">Source:</span>
            {task.source ? (
              <Badge variant="outline">{task.source}</Badge>
            ) : (
              "—"
            )}
          </div>
          <div>
            <span className="font-medium text-foreground">Created:</span>{" "}
            {new Date(task.created_at).toLocaleString()}
          </div>
          <div>
            <span className="font-medium text-foreground">Updated:</span>{" "}
            {new Date(task.updated_at).toLocaleString()}
          </div>
        </div>

        {/* Activity log */}
        {task.activity && task.activity.length > 0 && (
          <>
            <Separator className="my-6" />
            <div>
              <h3 className="font-medium mb-3">Activity</h3>
              <div className="space-y-3">
                {task.activity.map((entry, i) => (
                  <div key={i} className="text-sm">
                    <div className="flex items-center gap-2">
                      <Badge variant="secondary">{entry.action}</Badge>
                      <span className="text-muted-foreground text-xs">
                        {new Date(entry.at).toLocaleString()}
                      </span>
                    </div>
                    {entry.detail && (
                      <p className="mt-1 text-muted-foreground pl-2">
                        {entry.detail}
                      </p>
                    )}
                  </div>
                ))}
              </div>
            </div>
          </>
        )}
      </SheetContent>
    </Sheet>
  );
}
