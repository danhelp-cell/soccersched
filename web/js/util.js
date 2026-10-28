const PB = {
  state: {
    settings: null,
    ageGroups: [],
    teams: [],
    venues: [],
    pitches: [],
    surfaces: [],
    formats: [],
    eventTypes: [],
    blackouts: [],
    events: [],
    summary: [],
    view: "calendar",
    form: null,
    formMode: null,
    dirty: false,
    hiddenTeams: new Set(),
    filters: { text: "", env: "", surfaces: new Set(), types: new Set(), lateOnly: false, hideCancelled: true },
    dataDir: "",
    version: "",
    undo: [],
    calendar: null,
    draggable: null,
    rangeStart: null,
    rangeEnd: null,
    viewTitle: "",
    activeCalView: "timeGridWeek",
    mini: null,
    selectedId: null,
    initialized: false,
  },
};

function esc(value) {
  return String(value ?? "").replace(/[&<>"']/g, (c) => ({
    "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;",
  }[c]));
}

function pad(n) { return String(n).padStart(2, "0"); }

function toLocalISO(date) {
  const d = date instanceof Date ? date : new Date(date);
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

function ymd(date) {
  const d = date instanceof Date ? date : parseLocal(date);
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

function parseLocal(value) {
  if (value instanceof Date) return value;
  const text = String(value || "");
  const match = text.match(/^(\d{4})-(\d{2})-(\d{2})(?:[T ](\d{2}):(\d{2})(?::(\d{2}))?)?/);
  if (!match) return new Date(text);
  return new Date(+match[1], +match[2] - 1, +match[3], +(match[4] || 0), +(match[5] || 0), +(match[6] || 0));
}

function stampInput(value) {
  const d = parseLocal(value);
  return `${ymd(d)}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function clockLabel(hhmm, format) {
  const [h, m] = String(hhmm || "00:00").split(":").map(Number);
  const d = new Date(2000, 0, 1, h || 0, m || 0);
  return format === "24" ? `${pad(d.getHours())}:${pad(d.getMinutes())}` : d.toLocaleTimeString([], { hour: "numeric", minute: "2-digit" });
}

function formatRange(start, end, allDay) {
  const a = parseLocal(start);
  const b = parseLocal(end);
  const day = a.toLocaleDateString([], { weekday: "long", month: "long", day: "numeric" });
  if (allDay) return day;
  return `${day} · ${clockLabel(`${pad(a.getHours())}:${pad(a.getMinutes())}`, PB.state.settings?.timeFormat)}–${clockLabel(`${pad(b.getHours())}:${pad(b.getMinutes())}`, PB.state.settings?.timeFormat)}`;
}

function byId(list, id) { return (list || []).find((item) => item.id === id) || null; }
function teamById(id) { return byId(PB.state.teams, id); }
function pitchById(id) { return byId(PB.state.pitches, id); }
function venueById(id) { return byId(PB.state.venues, id); }
function typeById(id) { return byId(PB.state.eventTypes, id); }
function ageById(id) { return byId(PB.state.ageGroups, id); }

function isPriorityTeam(team) {
  if (!team) return false;
  if (team.priority === "always") return true;
  if (team.priority === "never") return false;
  const group = ageById(team.ageGroupId);
  return !!group && group.maxAge < (PB.state.settings?.priorityCutoffAge ?? 11);
}

function isLateForm(form) {
  if (!form || form.allDay) return false;
  const settings = PB.state.settings || {};
  const teams = (form.teamIds || []).map(teamById).filter(Boolean);
  if (!teams.some(isPriorityTeam)) return false;
  const type = typeById(form.eventTypeId);
  const cats = settings.priorityCategories || [];
  if (cats.length && type && !cats.includes(type.category)) return false;
  const start = parseLocal(form.start);
  const days = settings.priorityDays || [];
  if (days.length && !days.includes(start.getDay())) return false;
  const [h, m] = String(settings.earlyWindowEnd || "17:30").split(":").map(Number);
  return start.getHours() * 60 + start.getMinutes() >= h * 60 + (m || 0);
}

function tint(hex, amount) {
  const raw = String(hex || "").replace("#", "");
  if (raw.length < 6) return "#e7f3c4";
  const mix = (c) => Math.round(c + (255 - c) * amount);
  return `rgb(${mix(parseInt(raw.slice(0, 2), 16))}, ${mix(parseInt(raw.slice(2, 4), 16))}, ${mix(parseInt(raw.slice(4, 6), 16))})`;
}

function toast(message) {
  const host = document.getElementById("toasts");
  const el = document.createElement("div");
  el.className = "toast";
  el.textContent = message;
  host.appendChild(el);
  setTimeout(() => el.remove(), 4200);
}

function confirmBox(title, message, okLabel) {
  return new Promise((resolve) => {
    const root = document.getElementById("modal-root");
    root.innerHTML = `<div class="modal-back"><div class="modal" role="dialog"><h2>${esc(title)}</h2><p>${esc(message)}</p><div class="row"><button type="button" class="btn-ghost" id="modal-cancel">Cancel</button><button type="button" class="btn" id="modal-ok">${esc(okLabel || "Confirm")}</button></div></div></div>`;
    root.querySelector("#modal-cancel").onclick = () => { root.innerHTML = ""; resolve(false); };
    root.querySelector("#modal-ok").onclick = () => { root.innerHTML = ""; resolve(true); };
  });
}

function dayNames() { return ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"]; }

function downloadBlob(blob, filename) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}

function errorText(err) {
  const issues = err?.data?.issues?.filter((issue) => issue.level === "error") || [];
  if (issues.length) return issues.map((issue) => issue.message).join(" ");
  return err?.data?.error || err?.message || "Something went wrong.";
}
