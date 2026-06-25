const state = {
  token: localStorage.getItem("docflow.accessToken") || "",
  currentView: "dashboard",
  calendarDate: new Date(),
  selectedDate: new Date(),
  cache: {
    documents: [],
    tasks: [],
    users: [],
    events: [],
    notifications: []
  }
};

const titles = {
  dashboard: ["Dashboard", "Operational workspace for documents, tasks, calendar and notifications."],
  auth: ["Auth", "Registration, login, token refresh, password reset and sessions."],
  users: ["Users", "Profiles, roles, verification, banning and user stats."],
  documents: ["Documents", "Create, assign, change status, archive, export and inspect history."],
  tasks: ["Tasks", "Task creation, assignment, status transitions, filters and history."],
  calendar: ["Calendar", "Events, deadlines, month view and upcoming work."],
  notifications: ["Notifications", "User notifications, unread counts, preferences and templates."],
  mail: ["Mail", "SMTP sending, bulk mail, jobs, templates and stats."],
  tools: ["Tools", "Send direct requests to any gateway endpoint."]
};

const endpoints = {
  auth: "/api/v1/auth",
  users: "/api/v1/users",
  documents: "/api/v1/documents",
  tasks: "/api/v1/tasks",
  calendar: "/api/v1/events",
  notifications: "/api/v1/notifications",
  preferences: "/api/v1/preferences",
  templates: "/api/v1/templates",
  mail: "/api/v1/mail"
};

const el = (id) => document.getElementById(id);

function updateSessionLabel() {
  el("tokenInput").value = state.token;
  el("sessionLabel").textContent = state.token ? "token saved" : "not signed in";
}

function toast(message) {
  const box = el("toast");
  box.textContent = message;
  box.classList.add("show");
  clearTimeout(box.hideTimer);
  box.hideTimer = setTimeout(() => box.classList.remove("show"), 2800);
}

function print(value) {
  const out = el("output");
  if (!out) return;
  out.textContent = JSON.stringify(value, null, 2);
}

function asDateTime(value) {
  if (!value) return undefined;
  return new Date(value).toISOString();
}

function cleanObject(input) {
  const out = {};
  for (const [key, value] of Object.entries(input)) {
    if (value === "" || value === undefined || value === null) continue;
    out[key] = value;
  }
  return out;
}

function formData(form) {
  const data = {};
  for (const element of form.elements) {
    if (!element.name) continue;
    if (element.type === "checkbox") {
      data[element.name] = element.checked;
    } else {
      data[element.name] = element.value.trim();
    }
  }
  return data;
}

function idempotencyKey() {
  if (crypto.randomUUID) return crypto.randomUUID();
  return `${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

async function api(path, options = {}) {
  const method = options.method || "GET";
  const headers = new Headers(options.headers || {});
  if (state.token) headers.set("Authorization", `Bearer ${state.token}`);
  if (options.body !== undefined) headers.set("Content-Type", "application/json");
  if (["POST", "PUT", "PATCH"].includes(method)) headers.set("Idempotency-Key", idempotencyKey());
  const response = await fetch(path, {
    method,
    headers,
    body: options.body !== undefined ? JSON.stringify(options.body) : undefined
  });
  const text = await response.text();
  let data = text;
  try {
    data = text ? JSON.parse(text) : null;
  } catch {
    data = text;
  }
  const result = { path, method, status: response.status, ok: response.ok, data };
  print(result);
  if (!response.ok) {
    const message = data?.error?.message || data?.error || data?.message || `HTTP ${response.status}`;
    toast(String(message));
    throw new Error(String(message));
  }
  return data;
}

function normalizeList(data, keys) {
  if (Array.isArray(data)) return data;
  if (Array.isArray(data?.data)) return data.data;
  for (const key of keys) {
    if (Array.isArray(data?.[key])) return data[key];
    if (Array.isArray(data?.data?.[key])) return data.data[key];
  }
  if (Array.isArray(data?.items)) return data.items;
  return [];
}

function renderTable(targetId, rows, columns) {
  const target = el(targetId);
  if (!rows.length) {
    target.innerHTML = `<div class="item"><p>No data</p></div>`;
    return;
  }
  const head = columns.map((col) => `<th>${col.label}</th>`).join("");
  const body = rows.map((row) => {
    const cells = columns.map((col) => `<td>${formatCell(col.value(row))}</td>`).join("");
    return `<tr>${cells}</tr>`;
  }).join("");
  target.innerHTML = `<table><thead><tr>${head}</tr></thead><tbody>${body}</tbody></table>`;
}

function formatCell(value) {
  if (value === undefined || value === null || value === "") return "-";
  if (Array.isArray(value)) return value.join(", ");
  if (typeof value === "object") return `<code>${escapeHtml(JSON.stringify(value))}</code>`;
  return escapeHtml(String(value));
}

function escapeHtml(value) {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}

function renderList(targetId, rows, build) {
  const target = el(targetId);
  if (!rows.length) {
    target.innerHTML = `<div class="item"><p>No data</p></div>`;
    return;
  }
  target.innerHTML = rows.map(build).join("");
}

function setView(view) {
  state.currentView = view;
  document.querySelectorAll(".view").forEach((node) => node.classList.toggle("active", node.id === view));
  document.querySelectorAll(".nav button").forEach((node) => node.classList.toggle("active", node.dataset.view === view));
  const [title, subtitle] = titles[view];
  el("viewTitle").textContent = title;
  el("viewSubtitle").textContent = subtitle;
}

async function refreshCurrentView() {
  const view = state.currentView;
  if (view === "dashboard") return loadDashboard();
  if (view === "users") return actions["users.list"]();
  if (view === "documents") return actions["documents.list"]();
  if (view === "calendar") return renderCalendar();
  if (view === "notifications") return actions["notifications.history"]();
  if (view === "mail") return actions["mail.jobs"]();
}

async function loadDashboard() {
  const calendarUserId = el("calendarUserId").value.trim();
  const eventsPromise = calendarUserId
    ? api(`${endpoints.calendar}/upcoming/${encodeURIComponent(calendarUserId)}?days=7`)
    : Promise.resolve({ events: [], total: 0 });

  const results = await Promise.allSettled([
    api(`${endpoints.documents}?page=1&page_size=5`),
    api(`${endpoints.tasks}/stats`),
    eventsPromise,
    state.token ? api(`${endpoints.notifications}/unread-count`) : Promise.resolve(null)
  ]);
  const docs = results[0].status === "fulfilled" ? normalizeList(results[0].value, ["documents"]) : [];
  state.cache.documents = docs;
  el("metricDocuments").textContent = String(results[0].value?.total ?? docs.length ?? "-");
  el("metricTasks").textContent = String(results[1].value?.total ?? results[1].value?.data?.total ?? "-");
  el("metricEvents").textContent = String(normalizeList(results[2].value, ["events"]).length || "-");
  el("metricUnread").textContent = String(results[3].value?.count ?? results[3].value?.data?.count ?? "-");
  renderList("dashboardDocuments", docs, (doc) => `<div class="item"><strong>${formatCell(doc.title)}</strong><p>${formatCell(doc.status)} · ${formatCell(doc.type)} · ${formatCell(doc.id)}</p></div>`);
  renderList("dashboardTasks", state.cache.tasks, (task) => `<div class="item"><strong>${formatCell(task.title)}</strong><p>${formatCell(task.status)} · ${formatCell(task.priority)} · ${formatCell(task.id)}</p></div>`);
}

function renderUsers(rows) {
  state.cache.users = rows;
  renderTable("usersTable", rows, [
    { label: "ID", value: (x) => x.id },
    { label: "Email", value: (x) => x.email },
    { label: "Name", value: (x) => x.name || [x.first_name, x.last_name].filter(Boolean).join(" ") },
    { label: "Role", value: (x) => x.role },
    { label: "Active", value: (x) => x.is_active },
    { label: "Verified", value: (x) => x.is_verified }
  ]);
}

function renderDocuments(rows) {
  state.cache.documents = rows;
  renderTable("documentsTable", rows, [
    { label: "ID", value: (x) => x.id },
    { label: "Title", value: (x) => x.title },
    { label: "Type", value: (x) => x.type },
    { label: "Status", value: (x) => x.status },
    { label: "Creator", value: (x) => x.creator_id },
    { label: "Responsible", value: (x) => x.responsible_id },
    { label: "Deadline", value: (x) => x.deadline }
  ]);
}

function renderTasks(rows) {
  state.cache.tasks = rows;
  renderTable("tasksTable", rows, [
    { label: "ID", value: (x) => x.id },
    { label: "Title", value: (x) => x.title },
    { label: "Document", value: (x) => x.document_id },
    { label: "Assignee", value: (x) => x.assignee_id },
    { label: "Status", value: (x) => x.status },
    { label: "Priority", value: (x) => x.priority },
    { label: "Deadline", value: (x) => x.deadline }
  ]);
}

function renderEvents(rows) {
  state.cache.events = rows;
  renderList("eventsList", rows, (event) => `<div class="item"><strong>${formatCell(event.title)}</strong><p>${formatCell(event.event_type)} · ${formatCell(event.start_time)} · ${formatCell(event.id)}</p></div>`);
}

function renderNotifications(rows) {
  state.cache.notifications = rows;
  renderList("notificationsList", rows, (item) => `<div class="item"><strong>${formatCell(item.title)}</strong><p>${formatCell(item.body)} · read: ${formatCell(item.is_read)} · ${formatCell(item.id)}</p></div>`);
}

function renderMail(rows) {
  renderTable("mailTable", rows, [
    { label: "ID", value: (x) => x.id || x.template_id },
    { label: "Subject", value: (x) => x.subject },
    { label: "Status", value: (x) => x.status || x.is_active },
    { label: "Category", value: (x) => x.category || x.channel },
    { label: "Created", value: (x) => x.created_at || x.updated_at }
  ]);
}

function renderCalendar() {
  const grid = el("calendarGrid");
  const date = new Date(state.calendarDate.getFullYear(), state.calendarDate.getMonth(), 1);
  const month = date.getMonth();
  const firstDay = (date.getDay() + 6) % 7;
  date.setDate(date.getDate() - firstDay);
  el("calendarTitle").textContent = state.calendarDate.toLocaleDateString("en", { month: "long", year: "numeric" });
  const todayKey = dateKey(new Date());
  const selectedKey = dateKey(state.selectedDate);
  const cells = [];
  for (let i = 0; i < 42; i++) {
    const key = dateKey(date);
    const faded = date.getMonth() !== month;
    cells.push(`<button class="day ${key === todayKey ? "today" : ""} ${key === selectedKey ? "active" : ""}" data-date="${key}">
      <span>${date.toLocaleDateString("en", { weekday: "short" })}</span>
      <b style="${faded ? "color:var(--muted)" : ""}">${date.getDate()}</b>
    </button>`);
    date.setDate(date.getDate() + 1);
  }
  grid.innerHTML = cells.join("");
}

function dateKey(date) {
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, "0");
  const d = String(date.getDate()).padStart(2, "0");
  return `${y}-${m}-${d}`;
}

function dateParts(key) {
  const [year, month, day] = key.split("-").map(Number);
  return { year, month, day };
}

const actions = {
  "auth.register": async (data) => api(`${endpoints.auth}/register`, { method: "POST", body: data }),
  "auth.login": async (data) => {
    const response = await api(`${endpoints.auth}/login`, { method: "POST", body: data });
    const token = response?.access_token || response?.data?.access_token || response?.tokens?.access_token;
    if (token) {
      state.token = token;
      localStorage.setItem("docflow.accessToken", token);
      updateSessionLabel();
    }
    return response;
  },
  "auth.refresh": async (data) => api(`${endpoints.auth}/refresh`, { method: "POST", body: data }),
  "auth.logout": async (data) => api(`${endpoints.auth}/logout`, { method: "POST", body: data }),
  "auth.verifyEmail": async (data) => api(`${endpoints.auth}/verify-email`, { method: "POST", body: data }),
  "auth.forgotPassword": async (data) => api(`${endpoints.auth}/forgot-password`, { method: "POST", body: data }),
  "auth.resetPassword": async (data) => api(`${endpoints.auth}/reset-password`, { method: "POST", body: data }),
  "auth.changePassword": async (data) => api(`${endpoints.auth}/change-password`, { method: "POST", body: data }),
  "users.list": async () => {
    const role = el("userRoleFilter").value.trim();
    const data = await api(`${endpoints.users}?page=1&page_size=50${role ? `&role=${encodeURIComponent(role)}` : ""}`);
    const rows = normalizeList(data, ["users"]);
    renderUsers(rows);
    return rows;
  },
  "users.create": async (data) => api(endpoints.users, { method: "POST", body: data }),
  "users.update": async (data) => {
    const { id, ...body } = data;
    return api(`${endpoints.users}/${encodeURIComponent(id)}`, { method: "PUT", body: cleanObject(body) });
  },
  "users.delete": async (data) => api(`${endpoints.users}/${encodeURIComponent(data.id)}`, { method: "DELETE" }),
  "users.ban": async (data) => {
    const { id, ...body } = data;
    return api(`${endpoints.users}/${encodeURIComponent(id)}/ban`, { method: "PATCH", body });
  },
  "users.byEmail": async () => api(`${endpoints.users}/by-email?email=${encodeURIComponent(el("userEmailSearch").value.trim())}`),
  "users.exists": async () => api(`${endpoints.users}/exists?email=${encodeURIComponent(el("userEmailSearch").value.trim())}`),
  "documents.list": async () => {
    const params = new URLSearchParams({ page: "1", page_size: "50" });
    if (el("docCreatorFilter").value.trim()) params.set("creator_id", el("docCreatorFilter").value.trim());
    if (el("docResponsibleFilter").value.trim()) params.set("responsible_id", el("docResponsibleFilter").value.trim());
    const data = await api(`${endpoints.documents}?${params}`);
    const rows = normalizeList(data, ["documents"]);
    renderDocuments(rows);
    return rows;
  },
  "documents.create": async (data) => {
    const body = cleanObject({
      title: data.title,
      description: data.description,
      doc_type: data.type,
      creator_id: data.creator_id,
      responsible_id: data.responsible_id,
      deadline: asDateTime(data.deadline),
      file_url: data.file_url,
      tags: data.tags ? data.tags.split(",").map((x) => x.trim()).filter(Boolean) : undefined
    });
    return api(endpoints.documents, { method: "POST", body });
  },
  "documents.update": async (data) => {
    const { id, ...rest } = data;
    return api(`${endpoints.documents}/${encodeURIComponent(id)}`, { method: "PUT", body: cleanObject(rest) });
  },
  "documents.delete": async (data) => api(`${endpoints.documents}/${encodeURIComponent(data.id)}`, { method: "DELETE", body: { actor_id: data.actor_id } }),
  "documents.assign": async (data) => api(`${endpoints.documents}/${encodeURIComponent(data.id)}/assign`, { method: "POST", body: { responsible_id: data.responsible_id, actor_id: data.actor_id } }),
  "documents.status": async (data) => api(`${endpoints.documents}/${encodeURIComponent(data.id)}/status`, { method: "PATCH", body: cleanObject({ new_status: data.new_status, actor_id: data.actor_id, comment: data.comment }) }),
  "documents.archive": async (data) => api(`${endpoints.documents}/${encodeURIComponent(data.id)}/archive`, { method: "POST", body: { actor_id: data.actor_id } }),
  "documents.history": async (data) => api(`${endpoints.documents}/${encodeURIComponent(data.id)}/history`),
  "documents.filter": async () => {
    const body = cleanObject({
      status: el("docStatusFilter").value,
      type: el("docTypeFilter").value,
      responsible_id: el("docResponsibleFilter").value.trim(),
      creator_id: el("docCreatorFilter").value.trim(),
      page: 1,
      page_size: 50
    });
    const data = await api(`${endpoints.documents}/filter`, { method: "POST", body });
    const rows = normalizeList(data, ["documents"]);
    renderDocuments(rows);
    return rows;
  },
  "documents.markOverdue": async () => api(`${endpoints.documents}/mark-overdue`, { method: "POST", body: {} }),
  "documents.exportCsv": async () => window.open(`${endpoints.documents}/export/csv`, "_blank"),
  "documents.stats": async () => api(`${endpoints.documents}/stats`),
  "tasks.byDocument": async () => {
    const id = el("taskDocumentFilter").value.trim();
    const data = await api(`${endpoints.tasks}/document/${encodeURIComponent(id)}?page=1&page_size=50`);
    const rows = normalizeList(data, ["tasks"]);
    renderTasks(rows);
    return rows;
  },
  "tasks.byAssignee": async () => {
    const id = el("taskAssigneeFilter").value.trim();
    const data = await api(`${endpoints.tasks}/assignee/${encodeURIComponent(id)}?page=1&page_size=50`);
    const rows = normalizeList(data, ["tasks"]);
    renderTasks(rows);
    return rows;
  },
  "tasks.create": async (data) => api(endpoints.tasks, { method: "POST", body: cleanObject({ ...data, deadline: asDateTime(data.deadline) }) }),
  "tasks.update": async (data) => {
    const { id, ...rest } = data;
    return api(`${endpoints.tasks}/${encodeURIComponent(id)}`, { method: "PUT", body: cleanObject(rest) });
  },
  "tasks.delete": async (data) => api(`${endpoints.tasks}/${encodeURIComponent(data.id)}`, { method: "DELETE", body: { actor_id: data.actor_id } }),
  "tasks.assign": async (data) => api(`${endpoints.tasks}/${encodeURIComponent(data.id)}/assign`, { method: "POST", body: { assignee_id: data.assignee_id, actor_id: data.actor_id } }),
  "tasks.status": async (data) => api(`${endpoints.tasks}/${encodeURIComponent(data.id)}/status`, { method: "PATCH", body: { new_status: data.new_status, actor_id: data.actor_id } }),
  "tasks.history": async (data) => api(`${endpoints.tasks}/${encodeURIComponent(data.id)}/history`),
  "tasks.filter": async () => {
    const body = cleanObject({
      status: el("taskStatusFilter").value,
      priority: el("taskPriorityFilter").value,
      assignee_id: el("taskAssigneeFilter").value.trim(),
      document_id: el("taskDocumentFilter").value.trim(),
      page: 1,
      page_size: 50
    });
    const data = await api(`${endpoints.tasks}/filter`, { method: "POST", body });
    const rows = normalizeList(data, ["tasks"]);
    renderTasks(rows);
    return rows;
  },
  "tasks.markOverdue": async () => api(`${endpoints.tasks}/mark-overdue`, { method: "POST", body: {} }),
  "tasks.stats": async () => api(`${endpoints.tasks}/stats`),
  "calendar.create": async (data) => api(endpoints.calendar, { method: "POST", body: cleanObject({ ...data, start_time: asDateTime(data.start_time), end_time: asDateTime(data.end_time) }) }),
  "calendar.update": async (data) => {
    const { id, ...rest } = data;
    return api(`${endpoints.calendar}/${encodeURIComponent(id)}`, { method: "PUT", body: cleanObject({ ...rest, start_time: asDateTime(rest.start_time), end_time: asDateTime(rest.end_time) }) });
  },
  "calendar.delete": async (data) => api(`${endpoints.calendar}/${encodeURIComponent(data.id)}`, { method: "DELETE", body: { deleted_by: data.deleted_by } }),
  "calendar.upcoming": async () => {
    const data = await api(`${endpoints.calendar}/upcoming/${encodeURIComponent(el("calendarUserId").value.trim())}?days=${encodeURIComponent(el("calendarDays").value || "7")}`);
    const rows = normalizeList(data, ["events"]);
    renderEvents(rows);
    return rows;
  },
  "calendar.stats": async () => api(`${endpoints.calendar}/stats/${encodeURIComponent(el("calendarUserId").value.trim())}`),
  "notifications.history": async () => {
    const data = await api(`${endpoints.notifications}?page=1&page_size=50`);
    const rows = normalizeList(data, ["items", "notifications"]);
    renderNotifications(rows);
    return rows;
  },
  "notifications.create": async (data) => api(endpoints.notifications, { method: "POST", body: cleanObject(data) }),
  "notifications.unread": async () => api(`${endpoints.notifications}/unread-count`),
  "notifications.read": async (data) => api(`${endpoints.notifications}/${encodeURIComponent(data.id)}/read?user_id=${encodeURIComponent(data.user_id)}`, { method: "PATCH", body: {} }),
  "notifications.markAllRead": async () => api(`${endpoints.notifications}/read-all`, { method: "POST", body: {} }),
  "notifications.delete": async (data) => api(`${endpoints.notifications}/${encodeURIComponent(data.id)}`, { method: "DELETE" }),
  "notifications.preferences": async (data) => {
    const { user_id, ...body } = data;
    return api(`${endpoints.preferences}`, { method: "POST", body });
  },
  "mail.send": async (data) => api(`${endpoints.mail}/send-email`, { method: "POST", body: cleanObject(data) }),
  "mail.bulk": async (data) => api(`${endpoints.mail}/send-bulk`, { method: "POST", body: cleanObject({ ...data, to: data.to.split(/\n|,/).map((x) => x.trim()).filter(Boolean) }) }),
  "mail.jobs": async () => {
    const data = await api(`${endpoints.mail}/jobs?page=1&page_size=50`);
    const rows = normalizeList(data, ["items", "jobs"]);
    renderMail(rows);
    return rows;
  },
  "mail.templates": async () => {
    const data = await api(`${endpoints.mail}/templates`);
    const rows = normalizeList(data, ["items", "templates"]);
    renderMail(rows);
    return rows;
  },
  "mail.stats": async () => api(`${endpoints.mail}/stats`),
  "tools.raw": async (data) => {
    const body = data.body ? JSON.parse(data.body) : undefined;
    return api(data.path, { method: data.method, body });
  }
};

function bindEvents() {
  el("nav").addEventListener("click", (event) => {
    const button = event.target.closest("button[data-view]");
    if (button) setView(button.dataset.view);
  });
  document.querySelectorAll("[data-view-jump]").forEach((button) => {
    button.addEventListener("click", () => setView(button.dataset.viewJump));
  });
  document.querySelectorAll("form[data-action]").forEach((form) => {
    form.addEventListener("submit", async (event) => {
      event.preventDefault();
      const action = actions[form.dataset.action];
      if (!action) return;
      try {
        await action(formData(form));
        toast("Done");
      } catch (error) {
        print({ error: error.message });
      }
    });
  });
  document.querySelectorAll("[data-action-click]").forEach((button) => {
    button.addEventListener("click", async () => {
      try {
        await actions[button.dataset.actionClick]();
        toast("Done");
      } catch (error) {
        print({ error: error.message });
      }
    });
  });
  el("saveTokenBtn").addEventListener("click", () => {
    state.token = el("tokenInput").value.trim();
    localStorage.setItem("docflow.accessToken", state.token);
    updateSessionLabel();
    toast("Token saved");
  });
  el("clearTokenBtn").addEventListener("click", () => {
    state.token = "";
    localStorage.removeItem("docflow.accessToken");
    updateSessionLabel();
    toast("Token cleared");
  });
  el("refreshBtn").addEventListener("click", () => refreshCurrentView());
  el("healthBtn").addEventListener("click", () => api("/health"));
  el("prevMonthBtn").addEventListener("click", () => {
    state.calendarDate.setMonth(state.calendarDate.getMonth() - 1);
    renderCalendar();
  });
  el("nextMonthBtn").addEventListener("click", () => {
    state.calendarDate.setMonth(state.calendarDate.getMonth() + 1);
    renderCalendar();
  });
  el("calendarGrid").addEventListener("click", async (event) => {
    const day = event.target.closest(".day");
    if (!day) return;
    const parts = dateParts(day.dataset.date);
    state.selectedDate = new Date(parts.year, parts.month - 1, parts.day);
    renderCalendar();
    const user = el("calendarUserId").value.trim();
    if (!user) {
      toast("Set user id first");
      return;
    }
    const data = await api(`${endpoints.calendar}/by-day/${parts.year}/${parts.month}/${parts.day}?user_id=${encodeURIComponent(user)}`);
    renderEvents(normalizeList(data, ["events"]));
  });
}

function initDefaults() {
  const now = new Date();
  const later = new Date(now.getTime() + 60 * 60 * 1000);
  document.querySelectorAll('input[type="datetime-local"]').forEach((input) => {
    if (!input.value && (input.name === "start_time" || input.name === "deadline")) input.value = toLocalInput(now);
    if (!input.value && input.name === "end_time") input.value = toLocalInput(later);
  });
}

function toLocalInput(date) {
  const pad = (value) => String(value).padStart(2, "0");
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

bindEvents();
updateSessionLabel();
initDefaults();
renderCalendar();
setView("dashboard");
loadDashboard().catch(() => {});
