// Ginla MongoDB initialization script
// Usage: mongosh mongodb://localhost:27017/ginla scripts/init-db.js
// Idempotent — safe to re-run.

const db = db.getSiblingDB("ginla");

print("=== Ginla DB Init ===");

// --- Collections with JSON Schema validation ---

function createIfNotExists(name, validator) {
  const existing = db.getCollectionNames();
  if (existing.includes(name)) {
    print(`  ✓ ${name} already exists, updating validator`);
    db.runCommand({ collMod: name, validator });
  } else {
    db.createCollection(name, { validator });
    print(`  + ${name} created`);
  }
}

createIfNotExists("households", {
  $jsonSchema: {
    bsonType: "object",
    required: ["name", "members", "created_at"],
    properties: {
      name: { bsonType: "string" },
      members: {
        bsonType: "array",
        items: {
          bsonType: "object",
          required: ["name", "role"],
          properties: {
            user_id: { bsonType: ["string", "null"] },
            name: { bsonType: "string" },
            role: { enum: ["owner", "admin", "member", "viewer"] },
            handler_id: { bsonType: ["objectId", "null"] }
          }
        }
      },
      invites: {
        bsonType: "array",
        items: {
          bsonType: "object",
          required: ["email", "role", "token", "created_at", "expires_at"],
          properties: {
            email: { bsonType: "string" },
            role: { enum: ["admin", "member", "viewer"] },
            token: { bsonType: "string" },
            invited_by: { bsonType: "string" },
            created_at: { bsonType: "date" },
            expires_at: { bsonType: "date" }
          }
        }
      },
      created_at: { bsonType: "date" }
    }
  }
});

createIfNotExists("tasks", {
  $jsonSchema: {
    bsonType: "object",
    required: ["household_id", "title", "status", "created_at", "updated_at"],
    properties: {
      household_id: { bsonType: "objectId" },
      title: { bsonType: "string" },
      description: { bsonType: "string" },
      checklist: {
        bsonType: "array",
        items: {
          bsonType: "object",
          required: ["text", "done"],
          properties: {
            text: { bsonType: "string" },
            done: { bsonType: "bool" }
          }
        }
      },
      tag: { enum: ["ME", "AI", "VA", "FAMILY", "HOUSEKEEPER", "DELEGATE", null] },
      handler_id: { bsonType: ["objectId", "null"] },
      status: { enum: ["inbox", "pending", "active", "done", "cancelled"] },
      priority: { enum: ["urgent", "high", "normal", "low"] },
      position: { bsonType: ["int", "double", "null"] },
      due: { bsonType: ["date", "null"] },
      source: { enum: ["manual", "agent", "email", "calendar", "voice", "screenshot", null] },
      meta: { bsonType: "object" },
      parent_id: { bsonType: ["objectId", "null"] },
      recurrence: {
        bsonType: ["object", "null"],
        properties: {
          rrule: { bsonType: "string" },
          next_at: { bsonType: "date" }
        }
      },
      attachments: { bsonType: "array" },
      activity: { bsonType: "array" },
      created_at: { bsonType: "date" },
      updated_at: { bsonType: "date" },
      done_at: { bsonType: ["date", "null"] }
    }
  }
});

createIfNotExists("handlers", {
  $jsonSchema: {
    bsonType: "object",
    required: ["household_id", "name", "type", "active", "created_at"],
    properties: {
      household_id: { bsonType: "objectId" },
      name: { bsonType: "string" },
      type: { enum: ["me", "family", "va", "housekeeper", "ai", "service"] },
      tags: { bsonType: "array", items: { bsonType: "string" } },
      contact: {
        bsonType: "object",
        properties: {
          email: { bsonType: "string" },
          phone: { bsonType: "string" },
          agent_id: { bsonType: "string" }
        }
      },
      active: { bsonType: "bool" },
      created_at: { bsonType: "date" }
    }
  }
});

createIfNotExists("rules", {
  $jsonSchema: {
    bsonType: "object",
    required: ["household_id", "name", "pattern", "active", "order", "created_at"],
    properties: {
      household_id: { bsonType: "objectId" },
      name: { bsonType: "string" },
      pattern: { bsonType: "string" },
      tag: { enum: ["ME", "AI", "VA", "FAMILY", "HOUSEKEEPER", "DELEGATE", null] },
      handler_id: { bsonType: ["objectId", "null"] },
      priority: { enum: ["urgent", "high", "normal", "low", null] },
      order: { bsonType: ["int", "double"] },
      active: { bsonType: "bool" },
      created_at: { bsonType: "date" }
    }
  }
});

// --- Indexes ---

print("\n--- Indexes ---");

function idx(coll, keys, opts) {
  const name = Object.entries(keys).map(([k, v]) => `${k}_${v}`).join("_");
  db[coll].createIndex(keys, { name, ...opts });
  print(`  + ${coll}.${name}`);
}

// tasks
idx("tasks", { household_id: 1, status: 1, tag: 1 });
idx("tasks", { household_id: 1, handler_id: 1, status: 1 });
idx("tasks", { household_id: 1, due: 1 }, { sparse: true });
idx("tasks", { household_id: 1, created_at: -1 });
idx("tasks", { household_id: 1, parent_id: 1 });
idx("tasks", { household_id: 1, status: 1, position: 1 });
idx("tasks", { "recurrence.next_at": 1 }, { sparse: true });

// handlers
idx("handlers", { household_id: 1, type: 1, active: 1 });

// rules
idx("rules", { household_id: 1, active: 1, order: 1 });

// --- Seed data (idempotent) ---

print("\n--- Seed data ---");

const now = new Date();

// Household
let household = db.households.findOne({ name: "The Watters Household" });
if (!household) {
  const result = db.households.insertOne({
    name: "The Watters Household",
    members: [
      { name: "Ali", role: "owner", user_id: null, handler_id: null }
    ],
    invites: [],
    created_at: now
  });
  household = { _id: result.insertedId };
  print("  + household: The Watters Household");
} else {
  print("  ✓ household already exists");
}

const hid = household._id;

// Handlers
function seedHandler(doc) {
  const existing = db.handlers.findOne({ household_id: hid, name: doc.name });
  if (!existing) {
    const result = db.handlers.insertOne({ household_id: hid, ...doc, created_at: now });
    print(`  + handler: ${doc.name}`);
    return result.insertedId;
  } else {
    print(`  ✓ handler ${doc.name} already exists`);
    return existing._id;
  }
}

const aliHandlerId = seedHandler({
  name: "Ali",
  type: "me",
  tags: ["ME"],
  contact: { email: "ali.watters@gmail.com", phone: "720-226-7602" },
  active: true
});

seedHandler({
  name: "Claude",
  type: "ai",
  tags: ["AI"],
  contact: { agent_id: "claude-worker" },
  active: true
});

// Link Ali's handler_id to household member
db.households.updateOne(
  { _id: hid, "members.name": "Ali" },
  { $set: { "members.$.handler_id": aliHandlerId } }
);

// Rules
function seedRule(doc) {
  const existing = db.rules.findOne({ household_id: hid, name: doc.name });
  if (!existing) {
    db.rules.insertOne({ household_id: hid, ...doc, active: true, created_at: now });
    print(`  + rule: ${doc.name}`);
  } else {
    print(`  ✓ rule ${doc.name} already exists`);
  }
}

seedRule({
  name: "Dog tasks → Family",
  pattern: "dog|toni|groomer|vet|pet",
  tag: "FAMILY",
  handler_id: null,
  priority: null,
  order: 10
});

seedRule({
  name: "Filing & reports → AI",
  pattern: "annual report|filing|secretary of state|oregon|business registry",
  tag: "AI",
  handler_id: null,
  priority: null,
  order: 20
});

seedRule({
  name: "Phone calls → VA",
  pattern: "call|phone|appointment|book|schedule|reservation",
  tag: "VA",
  handler_id: null,
  priority: null,
  order: 30
});

seedRule({
  name: "Cleaning & supplies → Housekeeper",
  pattern: "clean|vacuum|mop|laundry|supplies|trash|dishes",
  tag: "HOUSEKEEPER",
  handler_id: null,
  priority: null,
  order: 40
});

// Seed a sample task (today's real-world example)
const existingTask = db.tasks.findOne({ household_id: hid, title: "File Oregon annual report for TravelBlog LLC" });
if (!existingTask) {
  db.tasks.insertOne({
    household_id: hid,
    title: "File Oregon annual report for TravelBlog LLC",
    description: "Registry #240894998, due May 14, 2026. $100 at sos.oregon.gov/business",
    checklist: [
      { text: "Navigate to Oregon SOS annual report page", done: false },
      { text: "Look up registry 240894998", done: false },
      { text: "Fill in business details", done: false },
      { text: "Pay $100 fee", done: false },
      { text: "Save confirmation receipt", done: false }
    ],
    tag: "AI",
    handler_id: null,
    status: "pending",
    priority: "normal",
    position: null,
    due: new Date("2026-05-14T00:00:00Z"),
    source: "agent",
    meta: {
      business_name: "TRAVELBLOG LLC",
      registry_number: "240894998",
      filed_date: "2025-05-14",
      details_doc: "iCloud/Important Docs/TravelBlog LLC - Business Details.md"
    },
    parent_id: null,
    recurrence: {
      rrule: "FREQ=YEARLY;BYMONTH=5;BYMONTHDAY=14",
      next_at: new Date("2027-05-14T00:00:00Z")
    },
    attachments: [],
    activity: [
      { action: "created", by: "claude-session", at: now, detail: "Created during /work-queue session" },
      { action: "tagged", by: "auto-triage", at: now, detail: "Matched rule: Filing & reports → AI" }
    ],
    created_at: now,
    updated_at: now,
    done_at: null
  });
  print("  + task: File Oregon annual report");
} else {
  print("  ✓ task already exists");
}

print("\n=== Done ===");
