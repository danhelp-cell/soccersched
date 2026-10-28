function matchesFilters(event) {
  const filters = PB.state.filters;
  if (event.opening) {
    if (filters.lateOnly || filters.type) return false;
    if (filters.env && event.environment && event.environment !== filters.env) return false;
    if (filters.surface && event.surfaceId && event.surfaceId !== filters.surface) return false;
    if (filters.text) {
      const blob = [event.place, event.holder, event.zone, event.notes, event.source].join(" ").toLowerCase();
      if (!blob.includes(filters.text.toLowerCase())) return false;
    }
    return true;
  }
  const teams = event.teamIds || [];
  if (PB.state.hiddenTeams.size && teams.length && teams.every((id) => PB.state.hiddenTeams.has(id))) return false;
  if (filters.hideCancelled && event.status === "cancelled") return false;
  if (filters.lateOnly && !event.latePriority) return false;
  if (filters.env && event.environment !== filters.env) return false;
  if (filters.surface && event.surfaceId !== filters.surface) return false;
  if (filters.type && event.eventTypeId !== filters.type) return false;
  if (filters.text) {
    const blob = [event.title, event.locationLabel, event.placeDetail, event.opponent, event.notes, ...(event.teamNames || [])].join(" ").toLowerCase();
    if (!blob.includes(filters.text.toLowerCase())) return false;
  }
  return true;
}

function toFC(event) {
  const classes = [];
  if (event.latePriority) classes.push("is-late");
  if (event.status === "cancelled") classes.push("is-cancelled");
  if (event.status === "tentative") classes.push("is-tentative");
  if (event.id === PB.state.selectedId) classes.push("is-selected");
  return {
    id: event.id,
    title: event.title,
    start: event.start,
    end: event.end,
    allDay: !!event.allDay,
    classNames: classes,
    extendedProps: event,
  };
}

function openToFC(opening) {
  return {
    id: opening.id,
    title: opening.kind === "swap" ? "Swap" : "Open time",
    start: opening.start,
    end: opening.end,
    editable: false,
    startEditable: false,
    durationEditable: false,
    classNames: [opening.kind === "swap" ? "is-swap" : "is-open", opening.id === PB.state.selectedId ? "is-selected" : ""].filter(Boolean),
    extendedProps: { ...opening, opening: true },
  };
}

function paintEvent(info) {
  const event = info.event.extendedProps || {};
  if (event.opening) {
    info.el.style.background = "transparent";
    info.el.style.color = "#171717";
    info.el.style.borderLeft = "0";
    return;
  }
  const color = (event.teamColors && event.teamColors[0]) || event.typeColor || "#2f6a4a";
  info.el.style.background = tint(color, 0.82);
  info.el.style.borderLeft = "4px solid " + color;
  info.el.style.color = "#1c1915";
  if (event.status === "cancelled") info.el.style.opacity = "0.45";
  if (event.status === "tentative") info.el.style.backgroundImage = "repeating-linear-gradient(135deg, transparent, transparent 6px, rgba(28,25,21,.06) 6px, rgba(28,25,21,.06) 8px)";
}

function initCalendar() {
  const settings = PB.state.settings;
  const calendar = new FullCalendar.Calendar(document.getElementById("calendar"), {
    initialView: settings.defaultView || "timeGridWeek",
    firstDay: settings.weekStartsOn ?? 1,
    weekends: settings.showWeekends !== false,
    nowIndicator: true,
    allDaySlot: true,
    editable: true,
    droppable: true,
    selectable: true,
    selectMirror: true,
    eventResizableFromStart: true,
    height: "100%",
    expandRows: false,
    headerToolbar: false,
    slotMinTime: pad(settings.dayStartHour ?? 8) + ":00:00",
    slotMaxTime: pad(settings.dayEndHour ?? 22) + ":00:00",
    slotDuration: "00:" + pad(settings.slotMinutes || 30) + ":00",
    snapDuration: "00:" + pad(settings.slotMinutes || 30) + ":00",
    scrollTime: (settings.earlyWindowStart || "08:00") + ":00",
    businessHours: {
      daysOfWeek: settings.priorityDays?.length ? settings.priorityDays : [0, 1, 2, 3, 4, 5, 6],
      startTime: settings.earlyWindowStart || "08:00",
      endTime: settings.earlyWindowEnd || "17:30",
    },
    eventTimeFormat: settings.timeFormat === "24"
      ? { hour: "2-digit", minute: "2-digit", hour12: false }
      : { hour: "numeric", minute: "2-digit", meridiem: "short" },
    slotLabelFormat: settings.timeFormat === "24"
      ? { hour: "2-digit", minute: "2-digit", hour12: false }
      : { hour: "numeric", minute: "2-digit", meridiem: "short" },
    views: {
      timeGridWorkWeek: { type: "timeGridWeek", weekends: false, buttonText: "Work week" },
    },
    events(info, success, failure) {
      API.events(info.start, info.end).then((data) => {
        PB.state.events = data.events || [];
        PB.state.openings = data.openings || [];
        const shown = PB.state.events.filter(matchesFilters).map(toFC);
        const openings = PB.state.openings.filter(matchesFilters).map(openToFC);
        success(shown.concat(openings));
        renderStatus();
        if (PB.state.formMode === "edit" && PB.state.selectedId && !PB.state.dirty) {
          const fresh = PB.state.events.find((event) => event.id === PB.state.selectedId);
          if (fresh) PB.state.form = editable(fresh);
          else { PB.state.form = null; PB.state.selectedId = null; }
          renderInspector();
        }
      }).catch((err) => failure(err));
    },
    eventContent(arg) {
      const event = arg.event.extendedProps || {};
      if (event.opening) {
        const title = event.kind === "swap"
          ? `Swap${event.holder ? " · " + event.holder : ""}`
          : (event.use === "game" ? "Open game" : event.use === "practice" ? "Open practice" : "Open time");
        return { html: `<div class="ev-title">${esc(title)}</div><div class="ev-meta">${esc([arg.timeText, event.place].filter(Boolean).join(" · "))}</div>` };
      }
      const where = event.pitchName || event.locationLabel || "";
      const surface = event.surfaceName || "";
      const env = event.environmentResolved === "Indoor" ? "In" : event.environmentResolved === "Outdoor" ? "Out" : "";
      return { html: `<div class="ev-title">${esc(arg.event.title)}</div><div class="ev-meta">${esc([arg.timeText, env, where, surface].filter(Boolean).join(" · "))}</div>` };
    },
    eventDidMount: paintEvent,
    eventClick(info) {
      editEvent(info.event.extendedProps);
    },
    select(info) {
      startDraft(info.start, info.end, info.allDay);
      calendar.unselect();
    },
    async eventDrop(info) { await moveFCEvent(info); },
    async eventResize(info) { await moveFCEvent(info); },
    async eventReceive(info) {
      const teamId = info.event.extendedProps.teamId;
      const start = info.event.start;
      const end = info.event.end || new Date(start.getTime() + 60 * 60000);
      info.event.remove();
      const team = teamById(teamId);
      const type = PB.state.eventTypes.find((item) => item.category === "practice") || PB.state.eventTypes[0];
      const pitch = preferredPitch(team);
      const body = blankEvent(start, end, false);
      body.eventTypeId = type?.id || "";
      body.teamIds = [teamId];
      body.title = "";
      if (pitch) applyPitch(body, pitch);
      try {
        const saved = await API.create(body);
        pushUndo({ type: "create", ids: (saved.events || []).map((event) => event.id) });
        if (saved.issues?.some((issue) => issue.level === "warning")) toast(saved.issues.find((issue) => issue.level === "warning").message);
        PB.state.calendar.refetchEvents();
        if (saved.events?.[0]) editEvent(saved.events[0]);
      } catch (err) {
        toast(errorText(err));
        startDraft(start, end, false, body);
        showIssues(err.data?.issues || []);
      }
    },
    datesSet(info) {
      PB.state.rangeStart = info.start;
      PB.state.rangeEnd = info.end;
      PB.state.viewTitle = info.view.title;
      PB.state.activeCalView = info.view.type;
      PB.state.mini = new Date(info.start.getFullYear(), info.start.getMonth(), 1);
      renderTop();
      renderMini();
      refreshSummary();
    },
  });
  PB.state.calendar = calendar;
  calendar.render();
}

function applyCalendarOptions() {
  const calendar = PB.state.calendar;
  const settings = PB.state.settings;
  if (!calendar) return;
  calendar.setOption("firstDay", settings.weekStartsOn ?? 1);
  calendar.setOption("weekends", settings.showWeekends !== false);
  calendar.setOption("slotMinTime", pad(settings.dayStartHour ?? 8) + ":00:00");
  calendar.setOption("slotMaxTime", pad(settings.dayEndHour ?? 22) + ":00:00");
  calendar.setOption("slotDuration", "00:" + pad(settings.slotMinutes || 30) + ":00");
  calendar.setOption("scrollTime", (settings.earlyWindowStart || "08:00") + ":00");
  calendar.setOption("businessHours", {
    daysOfWeek: settings.priorityDays?.length ? settings.priorityDays : [0, 1, 2, 3, 4, 5, 6],
    startTime: settings.earlyWindowStart || "08:00",
    endTime: settings.earlyWindowEnd || "17:30",
  });
  calendar.refetchEvents();
}

async function moveFCEvent(info) {
  if (info.event.extendedProps?.opening) {
    info.revert();
    return;
  }
  const before = editable(info.oldEvent.extendedProps);
  before.start = toLocalISO(info.oldEvent.start);
  before.end = toLocalISO(info.oldEvent.end || info.oldEvent.start);
  const body = editable(info.event.extendedProps);
  body.start = toLocalISO(info.event.start);
  body.end = toLocalISO(info.event.end || info.event.start);
  body.allDay = info.event.allDay;
  try {
    const saved = await API.update(info.event.id, body);
    pushUndo({ type: "update", before });
    if (saved.issues?.some((issue) => issue.level === "warning")) toast(saved.issues.find((issue) => issue.level === "warning").message);
    PB.state.calendar.refetchEvents();
  } catch (err) {
    info.revert();
    toast(errorText(err));
    if (err.status === 422) {
      body.acknowledgeLate = false;
      startDraft(info.event.start, info.event.end, info.event.allDay, body);
      PB.state.formMode = "edit";
      PB.state.selectedId = info.event.id;
      PB.state.form.id = info.event.id;
      renderInspector();
      showIssues(err.data?.issues || []);
    }
  }
}

function preferredPitch(team) {
  if (!team) return PB.state.pitches.find((pitch) => pitch.active) || null;
  const format = team.preferredFormatId || ageById(team.ageGroupId)?.preferredFormatId;
  return PB.state.pitches.find((pitch) => pitch.active && team.preferredVenueId && pitch.venueId === team.preferredVenueId && (!format || pitch.formatId === format))
    || PB.state.pitches.find((pitch) => pitch.active && pitch.venueId === team.preferredVenueId)
    || PB.state.pitches.find((pitch) => pitch.active && format && pitch.formatId === format)
    || PB.state.pitches.find((pitch) => pitch.active)
    || null;
}

function bindDraggables() {
  const list = document.getElementById("team-list");
  if (!list || !window.FullCalendar?.Draggable) return;
  if (PB.state.draggable) {
    try { PB.state.draggable.destroy(); } catch { /* already gone */ }
  }
  PB.state.draggable = new FullCalendar.Draggable(list, {
    itemSelector: ".team-chip",
    eventData(el) {
      const team = teamById(el.dataset.id);
      const group = ageById(team?.ageGroupId);
      const minutes = team?.practiceMinutes || group?.defaultPracticeMinutes || 60;
      return {
        title: team?.shortName || team?.name || "Practice",
        duration: { minutes },
        extendedProps: { teamId: team?.id },
      };
    },
  });
}

function editable(event) {
  return {
    id: event.id || "",
    seriesId: event.seriesId || "",
    eventTypeId: event.eventTypeId || "",
    title: event.title || "",
    teamIds: [...(event.teamIds || [])],
    venueId: event.venueId || "",
    pitchId: event.pitchId || "",
    surfaceId: event.surfaceId || "",
    environment: event.environment || "",
    formatId: event.formatId || "",
    homeAway: event.homeAway || "home",
    opponent: event.opponent || "",
    awayLocation: event.awayLocation || "",
    start: event.start,
    end: event.end,
    allDay: !!event.allDay,
    notes: event.notes || "",
    status: event.status || "scheduled",
    acknowledgeLate: false,
    applyTo: "single",
    repeat: "none",
    repeatUntil: PB.state.settings?.seasonEnd || "",
    repeatCount: 8,
  };
}

function blankEvent(start, end, allDay) {
  const type = PB.state.eventTypes.find((item) => item.category === "practice") || PB.state.eventTypes[0];
  return editable({
    eventTypeId: type?.id || "",
    start: toLocalISO(start),
    end: toLocalISO(end),
    allDay,
    homeAway: "home",
    status: "scheduled",
  });
}

function applyPitch(form, pitch) {
  form.pitchId = pitch.id;
  form.venueId = pitch.venueId;
  form.surfaceId = pitch.surfaceId;
  form.environment = pitch.environment;
  form.formatId = pitch.formatId;
  form.homeAway = "home";
}
