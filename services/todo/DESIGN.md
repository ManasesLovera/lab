# Design Specification: Task Hub Minimalist UI

This document specifies the design system, color palette, layout structure, and interaction design implemented for the Task Hub application redesign.

---

## 1. Core Aesthetic Principles

The Task Hub UI has been completely rebuilt to reflect a **flat minimalist, content-focused** design. It prioritizes readability, generous whitespace, and zero visual clutter.

*   **Flat UI:** Completely eliminated all box shadows, gradients, and structural borders. Layout containers are styled border-free with flat white backgrounds.
*   **Generous Spacing:** Implemented wide paddings (`p-6` to `p-10`) and layout gaps (`gap-12`) to create a calm, focused workspace.
*   **Muted Accents:** Accents are reserved for functional status badges, interactive pills, and timeline markers.

---

## 2. Color Palette & Typography

### Colors
All color tokens utilize Tailwind CSS HSL mappings to maintain a modern, professional appearance:

| Element | Hex / CSS Class | Purpose |
| :--- | :--- | :--- |
| **Main Background** | `#FFFFFF` | Core canvas backdrop |
| **Primary Text** | `#111827` (`text-slate-900`) | Task titles, main headers, primary labels |
| **Secondary Text** | `#6B7280` (`text-slate-500`) | Descriptions, timestamps, metadata, placeholders |
| **Borders (Inputs)** | `#E2E8F0` (`border-slate-200`) | Subtle divider borders for form inputs |
| **Accent Buttons** | `#18181B` (`bg-zinc-900`) | Primary calls to action ("Create task", "New task") |

### Typography
*   **Sans-Serif (`font-sans`):** *Inter* (Google Fonts) used for all headings, body text, form elements, and task descriptions.
*   **Monospace (`font-mono`):** *JetBrains Mono* used for developer stats, MCP protocol details, and timeline labels.

---

## 3. Structural Layout

The layout uses a responsive 12-column CSS Grid to balance focus between task creation, registry view, and activity tracking:

```
+-------------------------------------------------------------------------+
|                                HEADER                                   |
|  [Task Hub / Focus Mode]            [My Tasks] [Search] [New Task] [AV] |
+-------------------------------------------------------------------------+
|                                 MAIN                                    |
|                                                                         |
|  COLUMN 1: REGISTRY & FORM (7/12 cols)  | COLUMN 2: ACTIVITY (5/12 cols) |
|                                         |                                |
|  +-----------------------------------+  |  +--------------------------+  |
|  | Task Details Form                 |  |  | Activity Timeline        |  |
|  | [Title]                           |  |  |                          |  |
|  | [Description]                     |  |  |  o Today                 |  |
|  | [Deadline]      [Owner]           |  |  |    - Agent connected     |  |
|  | [Create Task]   [Save Draft]      |  |  |    - Database updated    |  |
|  |                                   |  |  |  o Yesterday             |  |
|  +-----------------------------------+  |  |    - Status altered      |  |
|                                         |  |                          |  |
|  +-----------------------------------+  |  +--------------------------+  |
|  | Active Tasks List                 |  |                                |
|  | - Task 1 [Scheduled]              |  |                                |
|  | - Task 2 [In Review]              |  |                                |
|  +-----------------------------------+  |                                |
|                                         |                                |
+-------------------------------------------------------------------------+
|                                FOOTER                                   |
|  [>] Developer MCP (Model Context Protocol) Settings                    |
+-------------------------------------------------------------------------+
```

---

## 4. Components

### A. Header
*   **Top-Left Subtitle:** `"Task management"` (14px, muted gray) above the main title.
*   **Title:** `"Task Hub"` (30px, bold slate-900) placed beside a light purple Focus Mode badge (`bg-purple-50`, `text-purple-700`).
*   **Top-Right Navigation:** Light, flat hover links for list navigation.
*   **Search bar:** Flat, rounded search input filter with inline magnifying glass icon.
*   **Primary Action:** A compact, rounded `"New task"` button in zinc-900.

### B. Task Details Form
*   **Flat Input Fields:** Large inputs using simple bottom/thin slate-200 borders. Focus states transition smoothly to slate-400 with no ring glow.
*   **Owner & Deadline Row:** Staged side-by-side. The Go backend parses dates into `YYYY-MM-DD` / RFC3339 formats, rendered in natural short date formats (e.g. `Jun 18, 2026`) in the registry.
*   **Double Call-To-Action:**
    *   **Primary ("Create Task"):** Creates a task with a status of `scheduled`.
    *   **Secondary ("Save Draft"):** Creates a task with a status of `queued`.

### C. Active Tasks List
*   **Dynamic Counter:** Shows the active count of incomplete tasks next to the header (e.g. `"3 open"`).
*   **Status Pills:** Custom status tags utilizing 10% opacity colored backgrounds:
    *   `scheduled`: Purple (`bg-purple-50`, `text-purple-700`)
    *   `in_review`: Blue (`bg-blue-50`, `text-blue-700`)
    *   `queued`: Gray (`bg-slate-100`, `text-slate-600`)
    *   `completed`: Green (`bg-emerald-50`, `text-emerald-700`)
*   **Checked Muting:** Completed tasks show a checked status button, strike-through text, and have their container opacity reduced to 40%.

### D. Activity Log Timeline
*   **Subtle Timeline Track:** Left-aligned 1px light gray border (`border-slate-100`) running vertically down each date group.
*   **Color-Coded Bullet Points:** 8px timeline nodes indicating event types:
    *   `AGENT`: Purple (`bg-purple-500`)
    *   `USER`: Blue (`bg-blue-500`)
    *   `DATABASE`: Green (`bg-emerald-500`)
    *   `SYS`: Gray (`bg-slate-400`)
    *   `ERROR`: Red (`bg-red-500`)

---

## 5. Interaction Design

1.  **Status Cycling:** Clicking on a status pill in the Active Tasks list cycles the task through the statuses: `scheduled` ➡️ `in_review` ➡️ `queued` ➡️ `completed` and performs a partial `PUT` update to the database.
2.  **Inplace Search:** Standard search filters tasks by matching strings in the title, description, owner, or status pill text instantly.
3.  **Handoff Focus:** Clicking the "New task" button triggers a smooth scroll to the Task creation form and automatically focuses the Title input.
