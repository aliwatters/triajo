"use client";

import { useState } from "react";
import { createTask } from "@/lib/api";
import type { Tag, Priority } from "@/types/task";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Plus } from "lucide-react";

const TAGS: Tag[] = ["ME", "AI", "VA", "FAMILY", "HOUSEKEEPER", "DELEGATE"];
const PRIORITIES: Priority[] = ["urgent", "high", "normal", "low"];

interface Props {
  onCreated: () => void;
}

export function CreateTaskDialog({ onCreated }: Props) {
  const [open, setOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [form, setForm] = useState({
    title: "",
    description: "",
    tag: "" as Tag | "",
    priority: "normal" as Priority,
    due: "",
    handler_id: "",
  });

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!form.title.trim()) return;

    setSaving(true);
    setError(null);
    try {
      await createTask({
        title: form.title.trim(),
        description: form.description || undefined,
        tag: form.tag || undefined,
        priority: form.priority,
        due: form.due ? new Date(form.due).toISOString() : undefined,
        handler_id: form.handler_id || undefined,
      });
      setOpen(false);
      setForm({
        title: "",
        description: "",
        tag: "",
        priority: "normal",
        due: "",
        handler_id: "",
      });
      onCreated();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create task");
    } finally {
      setSaving(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button>
          <Plus className="h-4 w-4 mr-2" />
          Add Task
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create Task</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4 mt-2">
          <div className="space-y-1.5">
            <Label htmlFor="title">Title *</Label>
            <Input
              id="title"
              value={form.title}
              onChange={(e) => setForm({ ...form, title: e.target.value })}
              placeholder="Task title"
              required
            />
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="description">Description</Label>
            <Textarea
              id="description"
              value={form.description}
              onChange={(e) => setForm({ ...form, description: e.target.value })}
              placeholder="Optional description"
              rows={3}
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
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
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="due">Due date</Label>
            <Input
              id="due"
              type="date"
              value={form.due}
              onChange={(e) => setForm({ ...form, due: e.target.value })}
            />
          </div>

          {error && (
            <p className="text-sm text-destructive">{error}</p>
          )}

          <div className="flex gap-2 pt-2">
            <Button type="submit" disabled={saving || !form.title.trim()}>
              {saving ? "Creating..." : "Create Task"}
            </Button>
            <Button
              type="button"
              variant="outline"
              onClick={() => setOpen(false)}
            >
              Cancel
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}
