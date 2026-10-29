// Helper to build ISO timestamps in the current week (Monday-based)
function currentWeekSlot(dayOffset, hour, minute = 0) {
  const now = new Date();
  const day = now.getDay();
  const diffToMonday = (day === 0 ? -6 : 1) - day;
  const target = new Date(now.getFullYear(), now.getMonth(), now.getDate() + diffToMonday + dayOffset, hour, minute, 0, 0);
  const pad = (n) => String(n).padStart(2, "0");
  return `${target.getFullYear()}-${pad(target.getMonth() + 1)}-${pad(target.getDate())}T${pad(target.getHours())}:${pad(target.getMinutes())}:00`;
}

// Dummy JSON data for soccer practices including Age Group, Surface Type (Turf/Grass), and Location
const PRACTICE_DATA = [
  {
    id: "prac-1",
    title: "U8 Green Development Practice",
    start: currentWeekSlot(0, 16, 0),
    end: currentWeekSlot(0, 17, 0),
    extendedProps: {
      ageGroup: "U8",
      surfaceType: "Turf",
      location: "SaskTel Sports Centre — Turf A",
    },
  },
  {
    id: "prac-2",
    title: "U10 Girls Technical Session",
    start: currentWeekSlot(0, 17, 15),
    end: currentWeekSlot(0, 18, 30),
    extendedProps: {
      ageGroup: "U10",
      surfaceType: "Turf",
      location: "SaskTel Sports Centre — Turf B",
    },
  },
  {
    id: "prac-3",
    title: "U12 Alliance Practice",
    start: currentWeekSlot(1, 16, 30),
    end: currentWeekSlot(1, 18, 0),
    extendedProps: {
      ageGroup: "U12",
      surfaceType: "Grass",
      location: "Forest Park — Grass 1",
    },
  },
  {
    id: "prac-4",
    title: "U9 Black Small-Sided Drills",
    start: currentWeekSlot(2, 16, 0),
    end: currentWeekSlot(2, 17, 15),
    extendedProps: {
      ageGroup: "U9",
      surfaceType: "Grass",
      location: "Forest Park — Grass 2",
    },
  },
  {
    id: "prac-5",
    title: "Academy Evening Training",
    start: currentWeekSlot(2, 18, 30),
    end: currentWeekSlot(2, 20, 0),
    extendedProps: {
      ageGroup: "Academy",
      surfaceType: "Turf",
      location: "Forest Park Turf — Field 1",
    },
  },
  {
    id: "prac-6",
    title: "U14 City Rec Practice",
    start: currentWeekSlot(3, 17, 30),
    end: currentWeekSlot(3, 19, 0),
    extendedProps: {
      ageGroup: "U14",
      surfaceType: "Grass",
      location: "Forest Park — Grass 1",
    },
  },
  {
    id: "prac-7",
    title: "U16 Alliance Tactical Prep",
    start: currentWeekSlot(4, 18, 0),
    end: currentWeekSlot(4, 19, 30),
    extendedProps: {
      ageGroup: "U16",
      surfaceType: "Turf",
      location: "SaskTel Sports Centre — Turf A",
    },
  },
];

let practices = [...PRACTICE_DATA];
let calendar;

function escapeHtml(text) {
  const div = document.createElement("div");
  div.textContent = text ?? "";
  return div.innerHTML;
}

function formatRange(start, end) {
  if (!start) return "";
  const day = start.toLocaleDateString([], { weekday: "short", month: "short", day: "numeric" });
  const from = start.toLocaleTimeString([], { hour: "numeric", minute: "2-digit" });
  const to = end ? end.toLocaleTimeString([], { hour: "numeric", minute: "2-digit" }) : "";
  return to ? `${day}, ${from} – ${to}` : `${day}, ${from}`;
}

function renderSessionDetails(fcEvent, actionLabel = "Selected") {
  const host = document.getElementById("session-details");
  const statusLog = document.getElementById("status-log");
  if (!fcEvent) {
    host.innerHTML = `<p class="muted">Click, drag, or resize any practice on the calendar to inspect its details.</p>`;
    return;
  }

  const { ageGroup, surfaceType, location } = fcEvent.extendedProps || {};
  const when = formatRange(fcEvent.start, fcEvent.end);

  host.innerHTML = `
    <p class="session-title">${escapeHtml(fcEvent.title)}</p>
    <dl>
      <dt>When</dt>
      <dd>${escapeHtml(when)}</dd>
      <dt>Age Group</dt>
      <dd>${escapeHtml(ageGroup || "—")}</dd>
      <dt>Surface</dt>
      <dd>${escapeHtml(surfaceType || "—")}</dd>
      <dt>Location</dt>
      <dd>${escapeHtml(location || "—")}</dd>
    </dl>
  `;

  statusLog.textContent = `${actionLabel}: ${fcEvent.title} (${when})`;
}

function populateFilterOptions() {
  const ageSelect = document.getElementById("filter-age");
  const locSelect = document.getElementById("filter-location");

  const ages = [...new Set(practices.map((p) => p.extendedProps.ageGroup))].sort();
  const locations = [...new Set(practices.map((p) => p.extendedProps.location))].sort();

  const currentAge = ageSelect.value;
  const currentLoc = locSelect.value;

  ageSelect.innerHTML =
    `<option value="">All Age Groups</option>` +
    ages.map((a) => `<option value="${escapeHtml(a)}">${escapeHtml(a)}</option>`).join("");
  locSelect.innerHTML =
    `<option value="">All Locations</option>` +
    locations.map((l) => `<option value="${escapeHtml(l)}">${escapeHtml(l)}</option>`).join("");

  ageSelect.value = currentAge;
  locSelect.value = currentLoc;
}

function filteredEvents() {
  const age = document.getElementById("filter-age").value;
  const surface = document.getElementById("filter-surface").value;
  const location = document.getElementById("filter-location").value;

  return practices
    .filter((item) => {
      if (age && item.extendedProps.ageGroup !== age) return false;
      if (surface && item.extendedProps.surfaceType !== surface) return false;
      if (location && item.extendedProps.location !== location) return false;
      return true;
    })
    .map((item) => ({
      ...item,
      classNames: [
        item.extendedProps.surfaceType === "Grass" ? "surface-grass" : "surface-turf",
      ],
    }));
}

function syncEventTimes(fcEvent) {
  const target = practices.find((p) => p.id === fcEvent.id);
  if (target && fcEvent.start && fcEvent.end) {
    target.start = fcEvent.start.toISOString();
    target.end = fcEvent.end.toISOString();
  }
}

document.addEventListener("DOMContentLoaded", () => {
  populateFilterOptions();

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
      return {
        html: `
          <div class="fc-practice-card">
            <div class="fc-practice-top">
              <span>${escapeHtml(arg.event.title)}</span>
              <span class="fc-practice-badge">${escapeHtml(props.ageGroup)} · ${escapeHtml(props.surfaceType)}</span>
            </div>
            <div class="fc-practice-meta">${escapeHtml(arg.timeText)} · ${escapeHtml(props.location)}</div>
          </div>
        `,
      };
    },
    eventClick(info) {
      renderSessionDetails(info.event, "Selected");
    },
    eventDrop(info) {
      syncEventTimes(info.event);
      renderSessionDetails(info.event, "Moved");
    },
    eventResize(info) {
      syncEventTimes(info.event);
      renderSessionDetails(info.event, "Resized");
    },
  });

  calendar.render();

  document.getElementById("btn-prev").addEventListener("click", () => calendar.prev());
  document.getElementById("btn-today").addEventListener("click", () => calendar.today());
  document.getElementById("btn-next").addEventListener("click", () => calendar.next());

  ["filter-age", "filter-surface", "filter-location"].forEach((id) => {
    document.getElementById(id).addEventListener("change", () => {
      calendar.refetchEvents();
    });
  });

  const modal = document.getElementById("add-modal");
  const form = document.getElementById("add-form");

  document.getElementById("btn-add-practice").addEventListener("click", () => {
    document.getElementById("input-start").value = currentWeekSlot(3, 16, 0).slice(0, 16);
    document.getElementById("input-end").value = currentWeekSlot(3, 17, 15).slice(0, 16);
    modal.showModal();
  });

  document.getElementById("btn-cancel").addEventListener("click", () => modal.close());

  form.addEventListener("submit", (event) => {
    event.preventDefault();
    const newPractice = {
      id: "prac-" + (practices.length + 1),
      title: document.getElementById("input-title").value.trim(),
      start: document.getElementById("input-start").value + ":00",
      end: document.getElementById("input-end").value + ":00",
      extendedProps: {
        ageGroup: document.getElementById("input-age").value,
        surfaceType: document.getElementById("input-surface").value,
        location: document.getElementById("input-location").value,
      },
    };
    practices.push(newPractice);
    populateFilterOptions();
    calendar.refetchEvents();
    modal.close();
    form.reset();
    document.getElementById("status-log").textContent = `Added: ${newPractice.title}`;
  });
});
