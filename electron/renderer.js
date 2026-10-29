// Helper to build ISO timestamps in the current week (Monday-based)
function currentWeekSlot(dayOffset, hour, minute = 0) {
  const now = new Date();
  const day = now.getDay();
  const diffToMonday = (day === 0 ? -6 : 1) - day;
  const target = new Date(
    now.getFullYear(),
    now.getMonth(),
    now.getDate() + diffToMonday + dayOffset,
    hour,
    minute,
    0,
    0
  );
  const pad = (n) => String(n).padStart(2, "0");
  return `${target.getFullYear()}-${pad(target.getMonth() + 1)}-${pad(target.getDate())}T${pad(target.getHours())}:${pad(target.getMinutes())}:00`;
}

// Default Settings: priorityGroups array + priorityHourLimit
const DEFAULT_SETTINGS = {
  priorityGroups: ["U9", "U11", "Academy"],
  otherGroups: ["U13", "U15", "U17"],
  priorityHourLimit: "17:30",
};

// Default Teams Database: each team has Name, assigned Group, and external mapping IDs (rampTeamId, externalId)
const DEFAULT_TEAMS = [
  {
    id: "team-u9-green",
    name: "Aurora U9 Green",
    group: "U9",
    rampTeamId: "1001",
    externalId: "TL-U9-GRN",
  },
  {
    id: "team-u9-black",
    name: "Aurora U9 Black",
    group: "U9",
    rampTeamId: "1004",
    externalId: "TL-U9-BLK",
  },
  {
    id: "team-u11-green",
    name: "Aurora U11 Green",
    group: "U11",
    rampTeamId: "1007",
    externalId: "TL-U11-GRN",
  },
  {
    id: "team-u11-black",
    name: "Aurora U11 Black",
    group: "U11",
    rampTeamId: "1008",
    externalId: "TL-U11-BLK",
  },
  {
    id: "team-academy",
    name: "Aurora Academy Red",
    group: "Academy",
    rampTeamId: "1012",
    externalId: "TL-ACAD-RED",
  },
  {
    id: "team-u15-alliance",
    name: "Aurora U15 Alliance",
    group: "U15",
    rampTeamId: "1015",
    externalId: "TL-U15-ALL",
  },
];

// Dummy JSON data for soccer practices linked to Teams Database entries
const DEFAULT_PRACTICES = [
  {
    id: "prac-1",
    title: "Aurora U9 Green Practice",
    start: currentWeekSlot(0, 16, 0),
    end: currentWeekSlot(0, 17, 15),
    extendedProps: {
      teamId: "team-u9-green",
      ageGroup: "U9",
      surfaceType: "Turf",
      location: "SaskTel Sports Centre — Turf A",
    },
  },
  {
    id: "prac-2",
    title: "Aurora U9 Black Practice (Late Slot)",
    start: currentWeekSlot(0, 19, 0),
    end: currentWeekSlot(0, 20, 15),
    extendedProps: {
      teamId: "team-u9-black",
      ageGroup: "U9",
      surfaceType: "Turf",
      location: "SaskTel Sports Centre — Turf B",
    },
  },
  {
    id: "prac-3",
    title: "Aurora U11 Green Technical Session",
    start: currentWeekSlot(1, 16, 15),
    end: currentWeekSlot(1, 17, 30),
    extendedProps: {
      teamId: "team-u11-green",
      ageGroup: "U11",
      surfaceType: "Grass",
      location: "Forest Park — Grass 1",
    },
  },
  {
    id: "prac-4",
    title: "Aurora U11 Black Practice (Late Slot)",
    start: currentWeekSlot(2, 18, 0),
    end: currentWeekSlot(2, 19, 15),
    extendedProps: {
      teamId: "team-u11-black",
      ageGroup: "U11",
      surfaceType: "Grass",
      location: "Forest Park — Grass 2",
    },
  },
  {
    id: "prac-5",
    title: "Aurora Academy Red Evening Training",
    start: currentWeekSlot(3, 18, 30),
    end: currentWeekSlot(3, 20, 0),
    extendedProps: {
      teamId: "team-academy",
      ageGroup: "Academy",
      surfaceType: "Turf",
      location: "Forest Park Turf — Field 1",
    },
  },
  {
    id: "prac-6",
    title: "Aurora U15 Alliance Tactical Prep",
    start: currentWeekSlot(4, 19, 0),
    end: currentWeekSlot(4, 20, 30),
    extendedProps: {
      teamId: "team-u15-alliance",
      ageGroup: "U15",
      surfaceType: "Turf",
      location: "SaskTel Sports Centre — Turf A",
    },
  },
];

const STORAGE_KEYS = {
  settings: "soccer_scheduler_settings_v1",
  teams: "soccer_scheduler_teams_v1",
  practices: "soccer_scheduler_practices_v1",
};

function loadState(key, fallback) {
  try {
    const raw = localStorage.getItem(key);
    if (!raw) return structuredClone(fallback);
    return JSON.parse(raw);
  } catch {
    return structuredClone(fallback);
  }
}

function saveState() {
  try {
    localStorage.setItem(STORAGE_KEYS.settings, JSON.stringify(settings));
    localStorage.setItem(STORAGE_KEYS.teams, JSON.stringify(teamsDatabase));
    localStorage.setItem(STORAGE_KEYS.practices, JSON.stringify(practices));
  } catch {
    // ignore storage errors in restricted environments
  }
}

let settings = loadState(STORAGE_KEYS.settings, DEFAULT_SETTINGS);
let teamsDatabase = loadState(STORAGE_KEYS.teams, DEFAULT_TEAMS);
let practices = loadState(STORAGE_KEYS.practices, DEFAULT_PRACTICES);
let calendar;

function escapeHtml(text) {
  const div = document.createElement("div");
  div.textContent = text ?? "";
  return div.innerHTML;
}

function parseTimeToMinutes(hhmm) {
  const [h, m] = String(hhmm || "17:30")
    .split(":")
    .map((n) => parseInt(n, 10) || 0);
  return h * 60 + m;
}

// Look up the team associated with a FullCalendar event or raw practice object
function findTeamForEvent(eventOrProps) {
  const props = eventOrProps?.extendedProps || eventOrProps || {};
  if (props.teamId) {
    const byId = teamsDatabase.find((t) => t.id === props.teamId);
    if (byId) return byId;
  }
  const title = (eventOrProps?.title || "").toLowerCase();
  return (
    teamsDatabase.find((t) => title.includes(t.name.toLowerCase())) || null
  );
}

// Check if an event's associated Team belongs to a group in settings.priorityGroups
function isPriorityGroupTeam(team, fallbackGroup) {
  const groupName = team ? team.group : fallbackGroup;
  if (!groupName) return false;
  return settings.priorityGroups.includes(groupName);
}

// Check if an event starts later than settings.priorityHourLimit
function isLaterThanPriorityLimit(startDate) {
  if (!startDate) return false;
  const eventMinutes = startDate.getHours() * 60 + startDate.getMinutes();
  const limitMinutes = parseTimeToMinutes(settings.priorityHourLimit);
  return eventMinutes > limitMinutes;
}

// Combined check used by FullCalendar eventDidMount and inspector
function isLatePriorityEvent(fcEvent) {
  const team = findTeamForEvent(fcEvent);
  const group = team ? team.group : fcEvent.extendedProps?.ageGroup;
  if (!isPriorityGroupTeam(team, group)) {
    return false;
  }
  return isLaterThanPriorityLimit(fcEvent.start);
}

function formatRange(start, end) {
  if (!start) return "";
  const day = start.toLocaleDateString([], {
    weekday: "short",
    month: "short",
    day: "numeric",
  });
  const from = start.toLocaleTimeString([], {
    hour: "numeric",
    minute: "2-digit",
  });
  const to = end
    ? end.toLocaleTimeString([], { hour: "numeric", minute: "2-digit" })
    : "";
  return to ? `${day}, ${from} – ${to}` : `${day}, ${from}`;
}

function renderSessionDetails(fcEvent, actionLabel = "Selected") {
  const host = document.getElementById("session-details");
  const statusLog = document.getElementById("status-log");
  if (!fcEvent) {
    host.innerHTML = `<p class="muted">Click, drag, or resize any practice on the calendar to inspect its details.</p>`;
    return;
  }

  const props = fcEvent.extendedProps || {};
  const team = findTeamForEvent(fcEvent);
  const group = team ? team.group : props.ageGroup || "—";
  const inPriorityGroup = isPriorityGroupTeam(team, group);
  const late = isLatePriorityEvent(fcEvent);
  const when = formatRange(fcEvent.start, fcEvent.end);

  host.innerHTML = `
    <p class="session-title">${escapeHtml(fcEvent.title)}</p>
    <dl>
      <dt>When</dt>
      <dd>${escapeHtml(when)}</dd>
      <dt>Team</dt>
      <dd>${escapeHtml(team ? team.name : "Unassigned")}</dd>
      <dt>Group</dt>
      <dd>${escapeHtml(group)} ${inPriorityGroup ? "(Priority)" : "(Standard)"}</dd>
      <dt>rampTeamId</dt>
      <dd>${escapeHtml(team?.rampTeamId || "—")}</dd>
      <dt>Surface</dt>
      <dd>${escapeHtml(props.surfaceType || "—")}</dd>
      <dt>Location</dt>
      <dd>${escapeHtml(props.location || "—")}</dd>
    </dl>
    ${
      late
        ? `<div class="late-alert">Priority group (${escapeHtml(group)}) scheduled after ${escapeHtml(settings.priorityHourLimit)}</div>`
        : ""
    }
  `;

  statusLog.textContent = `${actionLabel}: ${fcEvent.title} (${when})`;
}

function allConfiguredGroups() {
  const combined = [
    ...settings.priorityGroups,
    ...(settings.otherGroups || []),
  ];
  return [...new Set(combined)];
}

function populateFilterOptions() {
  const teamSelect = document.getElementById("filter-team");
  const ageSelect = document.getElementById("filter-age");
  const locSelect = document.getElementById("filter-location");

  const currentTeam = teamSelect.value;
  const currentAge = ageSelect.value;
  const currentLoc = locSelect.value;

  teamSelect.innerHTML =
    `<option value="">All Teams</option>` +
    teamsDatabase
      .map(
        (t) =>
          `<option value="${escapeHtml(t.id)}">${escapeHtml(t.name)} (${escapeHtml(t.group)})</option>`
      )
      .join("");

  const groups = allConfiguredGroups();
  ageSelect.innerHTML =
    `<option value="">All Groups</option>` +
    groups
      .map(
        (g) =>
          `<option value="${escapeHtml(g)}">${escapeHtml(g)}${
            settings.priorityGroups.includes(g) ? " (Priority)" : ""
          }</option>`
      )
      .join("");

  const locations = [
    ...new Set(practices.map((p) => p.extendedProps.location)),
  ].sort();
  locSelect.innerHTML =
    `<option value="">All Locations</option>` +
    locations
      .map((l) => `<option value="${escapeHtml(l)}">${escapeHtml(l)}</option>`)
      .join("");

  teamSelect.value = currentTeam;
  ageSelect.value = currentAge;
  locSelect.value = currentLoc;

  // Also populate Quick Add modal team dropdown
  const modalTeamSelect = document.getElementById("input-team");
  if (modalTeamSelect) {
    modalTeamSelect.innerHTML = teamsDatabase
      .map(
        (t) =>
          `<option value="${escapeHtml(t.id)}">${escapeHtml(t.name)} — ${escapeHtml(t.group)}${
            t.rampTeamId ? ` (RAMP #${escapeHtml(t.rampTeamId)})` : ""
          }</option>`
      )
      .join("");
  }

  // Update legend pill text
  const legendEl = document.getElementById("legend-priority");
  if (legendEl) {
    legendEl.textContent = `Late Priority (> ${settings.priorityHourLimit})`;
  }
}

function filteredEvents() {
  const teamFilter = document.getElementById("filter-team").value;
  const ageFilter = document.getElementById("filter-age").value;
  const surfaceFilter = document.getElementById("filter-surface").value;
  const locationFilter = document.getElementById("filter-location").value;

  return practices
    .filter((item) => {
      const team = findTeamForEvent(item);
      const group = team ? team.group : item.extendedProps.ageGroup;
      if (teamFilter && item.extendedProps.teamId !== teamFilter) return false;
      if (ageFilter && group !== ageFilter) return false;
      if (surfaceFilter && item.extendedProps.surfaceType !== surfaceFilter)
        return false;
      if (locationFilter && item.extendedProps.location !== locationFilter)
        return false;
      return true;
    })
    .map((item) => {
      const team = findTeamForEvent(item);
      const group = team ? team.group : item.extendedProps.ageGroup;
      return {
        ...item,
        extendedProps: {
          ...item.extendedProps,
          ageGroup: group,
        },
        classNames: [
          item.extendedProps.surfaceType === "Grass"
            ? "surface-grass"
            : "surface-turf",
        ],
      };
    });
}

function syncEventTimes(fcEvent) {
  const target = practices.find((p) => p.id === fcEvent.id);
  if (target && fcEvent.start && fcEvent.end) {
    const pad = (n) => String(n).padStart(2, "0");
    const toLocalIso = (d) =>
      `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}:00`;
    target.start = toLocalIso(fcEvent.start);
    target.end = toLocalIso(fcEvent.end);
    saveState();
  }
}

// ---------------- TEAMS DATABASE UI ----------------

function populateTeamGroupDropdown() {
  const select = document.getElementById("team-group");
  if (!select) return;
  const previous = select.value;

  const priorityOpts = settings.priorityGroups
    .map(
      (g) =>
        `<option value="${escapeHtml(g)}">${escapeHtml(g)} (Priority Group)</option>`
    )
    .join("");

  const otherOpts = (settings.otherGroups || [])
    .filter((g) => !settings.priorityGroups.includes(g))
    .map(
      (g) =>
        `<option value="${escapeHtml(g)}">${escapeHtml(g)} (Standard Group)</option>`
    )
    .join("");

  select.innerHTML =
    `<optgroup label="Priority Groups (priorityGroups)">${priorityOpts}</optgroup>` +
    (otherOpts
      ? `<optgroup label="Standard Groups">${otherOpts}</optgroup>`
      : "");

  if (previous && allConfiguredGroups().includes(previous)) {
    select.value = previous;
  } else if (settings.priorityGroups.length > 0) {
    select.value = settings.priorityGroups[0];
  }
}

function renderTeamsDatabase() {
  populateTeamGroupDropdown();

  const tbody = document.getElementById("teams-tbody");
  const badge = document.getElementById("teams-count-badge");
  if (badge) {
    badge.textContent = `${teamsDatabase.length} team${
      teamsDatabase.length === 1 ? "" : "s"
    }`;
  }

  if (!tbody) return;
  tbody.innerHTML = teamsDatabase
    .map((team) => {
      const isPriority = settings.priorityGroups.includes(team.group);
      return `
        <tr data-team-row="${escapeHtml(team.id)}">
          <td><strong>${escapeHtml(team.name)}</strong></td>
          <td>${escapeHtml(team.group)}</td>
          <td>
            <span class="${isPriority ? "badge-priority" : "badge-standard"}">
              ${isPriority ? "Priority Group" : "Standard"}
            </span>
          </td>
          <td><code>${escapeHtml(team.rampTeamId || "—")}</code></td>
          <td><code>${escapeHtml(team.externalId || "—")}</code></td>
          <td>
            <button type="button" class="btn btn-sm" data-edit-team="${escapeHtml(team.id)}">Edit</button>
            <button type="button" class="btn btn-sm btn-danger" data-delete-team="${escapeHtml(team.id)}">Delete</button>
          </td>
        </tr>
      `;
    })
    .join("");

  tbody.querySelectorAll("[data-edit-team]").forEach((btn) => {
    btn.addEventListener("click", () => {
      const team = teamsDatabase.find((t) => t.id === btn.dataset.editTeam);
      if (!team) return;
      document.getElementById("team-id").value = team.id;
      document.getElementById("team-name").value = team.name;
      populateTeamGroupDropdown();
      document.getElementById("team-group").value = team.group;
      document.getElementById("team-ramp-id").value = team.rampTeamId || "";
      document.getElementById("team-external-id").value = team.externalId || "";
      document.getElementById("team-form-heading").textContent = `Edit Team: ${team.name}`;
      document.getElementById("team-submit-btn").textContent = "Update Team";
      document.getElementById("team-cancel-edit").hidden = false;
    });
  });

  tbody.querySelectorAll("[data-delete-team]").forEach((btn) => {
    btn.addEventListener("click", () => {
      const id = btn.dataset.deleteTeam;
      teamsDatabase = teamsDatabase.filter((t) => t.id !== id);
      saveState();
      renderTeamsDatabase();
      populateFilterOptions();
      calendar?.refetchEvents();
    });
  });
}

function resetTeamForm() {
  document.getElementById("team-id").value = "";
  document.getElementById("team-name").value = "";
  document.getElementById("team-ramp-id").value = "";
  document.getElementById("team-external-id").value = "";
  document.getElementById("team-form-heading").textContent = "Create / Edit Team";
  document.getElementById("team-submit-btn").textContent = "Save Team";
  document.getElementById("team-cancel-edit").hidden = true;
  populateTeamGroupDropdown();
}

// ---------------- SETTINGS UI ----------------

function renderSettingsPanel() {
  const priorityTags = document.getElementById("priority-group-tags");
  const otherTags = document.getElementById("other-group-tags");
  const limitInput = document.getElementById("setting-priority-limit");
  const jsonInput = document.getElementById("setting-priority-json");
  const preview = document.getElementById("settings-rule-preview");

  if (priorityTags) {
    priorityTags.innerHTML = settings.priorityGroups
      .map(
        (group) => `
          <span class="tag-chip">
            <span>${escapeHtml(group)}</span>
            <button type="button" data-remove-priority="${escapeHtml(group)}" title="Remove ${escapeHtml(group)}">×</button>
          </span>
        `
      )
      .join("");

    priorityTags.querySelectorAll("[data-remove-priority]").forEach((btn) => {
      btn.addEventListener("click", () => {
        const group = btn.dataset.removePriority;
        settings.priorityGroups = settings.priorityGroups.filter(
          (g) => g !== group
        );
        if (!settings.otherGroups.includes(group)) {
          settings.otherGroups.push(group);
        }
        saveState();
        renderSettingsPanel();
        renderTeamsDatabase();
        populateFilterOptions();
        calendar?.refetchEvents();
      });
    });
  }

  if (otherTags) {
    const nonPriority = (settings.otherGroups || []).filter(
      (g) => !settings.priorityGroups.includes(g)
    );
    otherTags.innerHTML = nonPriority
      .map(
        (group) => `
          <span class="tag-chip standard">
            <span>${escapeHtml(group)}</span>
            <button type="button" data-promote-group="${escapeHtml(group)}" title="Move ${escapeHtml(group)} to priorityGroups">↑ Priority</button>
            <button type="button" data-remove-other="${escapeHtml(group)}" title="Remove ${escapeHtml(group)}">×</button>
          </span>
        `
      )
      .join("");

    otherTags.querySelectorAll("[data-promote-group]").forEach((btn) => {
      btn.addEventListener("click", () => {
        const group = btn.dataset.promoteGroup;
        if (!settings.priorityGroups.includes(group)) {
          settings.priorityGroups.push(group);
        }
        settings.otherGroups = settings.otherGroups.filter((g) => g !== group);
        saveState();
        renderSettingsPanel();
        renderTeamsDatabase();
        populateFilterOptions();
        calendar?.refetchEvents();
      });
    });

    otherTags.querySelectorAll("[data-remove-other]").forEach((btn) => {
      btn.addEventListener("click", () => {
        const group = btn.dataset.removeOther;
        settings.otherGroups = settings.otherGroups.filter((g) => g !== group);
        saveState();
        renderSettingsPanel();
        renderTeamsDatabase();
        populateFilterOptions();
      });
    });
  }

  if (limitInput) {
    limitInput.value = settings.priorityHourLimit;
  }
  if (jsonInput) {
    jsonInput.value = JSON.stringify(settings.priorityGroups);
  }

  if (preview) {
    const affectedTeams = teamsDatabase.filter((t) =>
      settings.priorityGroups.includes(t.group)
    );
    preview.innerHTML = `
      <strong>Active Rule Summary:</strong><br/>
      • <code>priorityGroups</code> = <code>${escapeHtml(JSON.stringify(settings.priorityGroups))}</code><br/>
      • <code>priorityHourLimit</code> = <code>${escapeHtml(settings.priorityHourLimit)}</code><br/>
      • Teams in priority groups (${affectedTeams.length}): ${
        affectedTeams.map((t) => escapeHtml(t.name)).join(", ") || "None"
      }
    `;
  }
}

// ---------------- VIEW NAVIGATION ----------------

function switchView(viewName) {
  const views = ["calendar", "teams", "settings"];
  views.forEach((v) => {
    const section = document.getElementById(`view-${v}`);
    if (section) section.hidden = v !== viewName;
  });

  document.querySelectorAll(".nav-item[data-view]").forEach((btn) => {
    btn.classList.toggle("active", btn.dataset.view === viewName);
  });

  if (viewName === "calendar") {
    calendar?.updateSize();
    calendar?.refetchEvents();
  } else if (viewName === "teams") {
    renderTeamsDatabase();
  } else if (viewName === "settings") {
    renderSettingsPanel();
  }
}

// ---------------- INITIALIZATION ----------------

document.addEventListener("DOMContentLoaded", () => {
  populateFilterOptions();
  renderTeamsDatabase();
  renderSettingsPanel();

  const calendarEl = document.getElementById("calendar");
  calendar = new FullCalendar.Calendar(calendarEl, {
    initialView: "timeGridWeek",
    firstDay: 1,
    headerToolbar: false,
    editable: true,
    eventStartEditable: true,
    eventDurationEditable: true,
    eventResizableFromStart: true,
    selectable: true,
    nowIndicator: true,
    allDaySlot: false,
    slotMinTime: "08:00:00",
    slotMaxTime: "22:00:00",
    slotDuration: "00:30:00",
    height: "100%",
    events(info, successCallback) {
      successCallback(filteredEvents());
    },
    datesSet(info) {
      document.getElementById("calendar-title").textContent = info.view.title;
    },
    eventContent(arg) {
      const props = arg.event.extendedProps || {};
      const team = findTeamForEvent(arg.event);
      const group = team ? team.group : props.ageGroup;
      const late = isLatePriorityEvent(arg.event);
      return {
        html: `
          <div class="fc-practice-card">
            <div class="fc-practice-top">
              <span>${escapeHtml(arg.event.title)}</span>
              <span class="fc-practice-badge ${late ? "late-tag" : ""}">
                ${escapeHtml(group)} · ${escapeHtml(props.surfaceType)}${late ? " · LATE" : ""}
              </span>
            </div>
            <div class="fc-practice-meta">
              ${escapeHtml(arg.timeText)} · ${escapeHtml(props.location)}${
                team?.rampTeamId ? ` · RAMP #${escapeHtml(team.rampTeamId)}` : ""
              }
            </div>
          </div>
        `,
      };
    },
    // Check if the event's associated Team belongs to a group in priorityGroups
    // and whether its start time is later than priorityHourLimit.
    eventDidMount(info) {
      const team = findTeamForEvent(info.event);
      const group = team ? team.group : info.event.extendedProps?.ageGroup;
      const inPriorityGroup =
        Boolean(group) && settings.priorityGroups.includes(group);
      const isLate =
        inPriorityGroup && isLaterThanPriorityLimit(info.event.start);

      if (isLate) {
        info.el.classList.add("late-priority-outline");
        info.el.setAttribute(
          "title",
          `Late priority session: ${team ? team.name : info.event.title} (${group}) starts after ${settings.priorityHourLimit}`
        );
      } else {
        info.el.classList.remove("late-priority-outline");
      }
    },
    eventClick(info) {
      renderSessionDetails(info.event, "Selected");
    },
    eventDrop(info) {
      syncEventTimes(info.event);
      calendar.refetchEvents();
      renderSessionDetails(info.event, "Moved");
    },
    eventResize(info) {
      syncEventTimes(info.event);
      calendar.refetchEvents();
      renderSessionDetails(info.event, "Resized");
    },
  });

  calendar.render();

  // Sidebar & Jump View Navigation
  document.querySelectorAll(".nav-item[data-view]").forEach((btn) => {
    btn.addEventListener("click", () => switchView(btn.dataset.view));
  });
  document.querySelectorAll("[data-jump-view]").forEach((btn) => {
    btn.addEventListener("click", () => switchView(btn.dataset.jumpView));
  });

  // Calendar Week Controls
  document
    .getElementById("btn-prev")
    .addEventListener("click", () => calendar.prev());
  document
    .getElementById("btn-today")
    .addEventListener("click", () => calendar.today());
  document
    .getElementById("btn-next")
    .addEventListener("click", () => calendar.next());

  // Calendar Sidebar Filters
  ["filter-team", "filter-age", "filter-surface", "filter-location"].forEach(
    (id) => {
      document.getElementById(id).addEventListener("change", () => {
        calendar.refetchEvents();
      });
    }
  );

  // Teams Database Form
  document.getElementById("team-form").addEventListener("submit", (event) => {
    event.preventDefault();
    const idField = document.getElementById("team-id").value.trim();
    const name = document.getElementById("team-name").value.trim();
    const group = document.getElementById("team-group").value;
    const rampTeamId = document.getElementById("team-ramp-id").value.trim();
    const externalId = document.getElementById("team-external-id").value.trim();
    if (!name || !group) return;

    if (idField) {
      const existing = teamsDatabase.find((t) => t.id === idField);
      if (existing) {
        existing.name = name;
        existing.group = group;
        existing.rampTeamId = rampTeamId;
        existing.externalId = externalId;
      }
    } else {
      teamsDatabase.push({
        id: "team-" + Date.now(),
        name,
        group,
        rampTeamId,
        externalId,
      });
    }

    saveState();
    resetTeamForm();
    renderTeamsDatabase();
    renderSettingsPanel();
    populateFilterOptions();
    calendar.refetchEvents();
    document.getElementById("status-log").textContent = `Saved team: ${name} (${group})`;
  });

  document
    .getElementById("team-cancel-edit")
    .addEventListener("click", () => resetTeamForm());

  // Settings: Add Priority Group
  document
    .getElementById("add-priority-group-form")
    .addEventListener("submit", (event) => {
      event.preventDefault();
      const input = document.getElementById("new-priority-group");
      const val = input.value.trim();
      if (!val) return;
      if (!settings.priorityGroups.includes(val)) {
        settings.priorityGroups.push(val);
      }
      settings.otherGroups = (settings.otherGroups || []).filter(
        (g) => g !== val
      );
      input.value = "";
      saveState();
      renderSettingsPanel();
      renderTeamsDatabase();
      populateFilterOptions();
      calendar.refetchEvents();
    });

  // Settings: Add Standard (Non-Priority) Group
  document
    .getElementById("add-other-group-form")
    .addEventListener("submit", (event) => {
      event.preventDefault();
      const input = document.getElementById("new-other-group");
      const val = input.value.trim();
      if (!val) return;
      if (
        !settings.priorityGroups.includes(val) &&
        !settings.otherGroups.includes(val)
      ) {
        settings.otherGroups.push(val);
      }
      input.value = "";
      saveState();
      renderSettingsPanel();
      renderTeamsDatabase();
      populateFilterOptions();
    });

  // Settings: Save Priority Hour Limit
  document
    .getElementById("settings-limit-form")
    .addEventListener("submit", (event) => {
      event.preventDefault();
      const newLimit = document.getElementById("setting-priority-limit").value;
      if (newLimit) {
        settings.priorityHourLimit = newLimit;
        saveState();
        renderSettingsPanel();
        populateFilterOptions();
        calendar.refetchEvents();
        document.getElementById("status-log").textContent = `Updated priorityHourLimit to ${newLimit}`;
      }
    });

  // Settings: Reset Defaults
  document
    .getElementById("btn-reset-defaults")
    .addEventListener("click", () => {
      settings = structuredClone(DEFAULT_SETTINGS);
      teamsDatabase = structuredClone(DEFAULT_TEAMS);
      practices = structuredClone(DEFAULT_PRACTICES);
      saveState();
      renderSettingsPanel();
      renderTeamsDatabase();
      populateFilterOptions();
      calendar.refetchEvents();
      document.getElementById("status-log").textContent =
        "Reset settings, teams, and practices to defaults";
    });

  // Quick Add Practice Modal
  const modal = document.getElementById("add-modal");
  const form = document.getElementById("add-form");

  document.getElementById("btn-add-practice").addEventListener("click", () => {
    populateFilterOptions();
    document.getElementById("input-start").value = currentWeekSlot(3, 18, 0).slice(0, 16);
    document.getElementById("input-end").value = currentWeekSlot(3, 19, 15).slice(0, 16);
    modal.showModal();
  });

  document
    .getElementById("btn-cancel")
    .addEventListener("click", () => modal.close());

  form.addEventListener("submit", (event) => {
    event.preventDefault();
    const teamId = document.getElementById("input-team").value;
    const team = teamsDatabase.find((t) => t.id === teamId);
    const customTitle = document.getElementById("input-title").value.trim();
    const title = customTitle || `${team ? team.name : "Club"} Practice`;

    const newPractice = {
      id: "prac-" + Date.now(),
      title,
      start: document.getElementById("input-start").value + ":00",
      end: document.getElementById("input-end").value + ":00",
      extendedProps: {
        teamId: team ? team.id : "",
        ageGroup: team ? team.group : "U9",
        surfaceType: document.getElementById("input-surface").value,
        location: document.getElementById("input-location").value,
      },
    };
    practices.push(newPractice);
    saveState();
    populateFilterOptions();
    calendar.refetchEvents();
    modal.close();
    form.reset();
    document.getElementById("status-log").textContent = `Added: ${newPractice.title}`;
  });
});
