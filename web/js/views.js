function renderTop() {
  const center = document.getElementById("top-center");
  const actions = document.getElementById("top-actions");
  document.getElementById("club-name").textContent = PB.state.settings?.clubName || "Soccer club";
  if (PB.state.view === "calendar") {
    const views = [["timeGridDay", "Day"], ["timeGridWorkWeek", "Work week"], ["timeGridWeek", "Week"], ["dayGridMonth", "Month"]];
    center.innerHTML = `
      <button type="button" class="icon-btn" id="cal-prev" aria-label="Previous">‹</button>
      <button type="button" class="icon-btn" id="cal-today">Today</button>
      <button type="button" class="icon-btn" id="cal-next" aria-label="Next">›</button>
      <strong style="margin-left:8px">${esc(PB.state.viewTitle || "")}</strong>`;
    actions.innerHTML = `
      <div class="seg">${views.map(([id, label]) => `<button type="button" data-cal="${id}" class="${PB.state.activeCalView === id ? "is-active" : ""}">${label}</button>`).join("")}</div>
      <button type="button" class="btn-line" id="import-open">Import</button>
      <button type="button" class="btn-line" id="export-open">Export</button>
      <button type="button" class="btn" id="new-event">New</button>`;
    document.getElementById("cal-prev").onclick = () => PB.state.calendar?.prev();
    document.getElementById("cal-next").onclick = () => PB.state.calendar?.next();
    document.getElementById("cal-today").onclick = () => PB.state.calendar?.today();
    actions.querySelectorAll("[data-cal]").forEach((button) => {
      button.onclick = () => PB.state.calendar?.changeView(button.dataset.cal);
    });
  } else {
    const titles = { plan: "Plan the week", teams: "Teams", places: "Places and pitches", settings: "Settings" };
    center.innerHTML = `<strong>${esc(titles[PB.state.view] || "")}</strong>`;
    actions.innerHTML = `<button type="button" class="btn-line" id="import-open">Import</button><button type="button" class="btn-line" id="export-open">Export</button>`;
  }
  document.getElementById("import-open").onclick = openImport;
  document.getElementById("export-open").onclick = openExport;
  const newer = document.getElementById("new-event");
  if (newer) newer.onclick = () => {
    const start = roundHour(new Date());
    startDraft(start, new Date(start.getTime() + 60 * 60000), false);
  };
}

function roundHour(date) {
  const next = new Date(date);
  next.setMinutes(0, 0, 0);
  next.setHours(next.getHours() + 1);
  return next;
}

function renderLegend() {
  const settings = PB.state.settings;
  const host = document.getElementById("legend");
  if (!settings?.legendEnabled) { host.textContent = ""; return; }
  const early = `${clockLabel(settings.earlyWindowStart, settings.timeFormat)}–${clockLabel(settings.earlyWindowEnd, settings.timeFormat)}`;
  host.innerHTML = `Bright hours (${esc(early)}) are booked first for players under ${esc(settings.priorityCutoffAge)}. An age group can set its own hours, such as Academy evenings. A dashed outline means a priority team starts after its window. A dashed open block is a time from an imported timetable.`;
}

function ensureSidebar() {
  const el = document.getElementById("sidebar");
  if (el.dataset.built === "1") return;
  el.dataset.built = "1";
  el.innerHTML = `
    <div class="side-block" id="mini"></div>
    <div class="side-block">
      <label class="field"><span>Search this view</span><input id="cal-search" type="text" placeholder="Team, pitch, opponent" /></label>
      <div class="side-label">Teams <span class="muted">drag onto a time</span></div>
      <div id="team-list"></div>
      <div class="side-label">Show</div>
      <div id="filters"></div>
    </div>`;
  el.querySelector("#cal-search").addEventListener("input", (event) => {
    PB.state.filters.text = event.target.value;
    PB.state.calendar?.refetchEvents();
  });
  renderMini();
  renderTeamList();
  renderFilters();
}

function renderMini() {
  const host = document.getElementById("mini");
  if (!host || !PB.state.mini) return;
  const cursor = new Date(PB.state.mini);
  const year = cursor.getFullYear();
  const month = cursor.getMonth();
  const first = new Date(year, month, 1);
  const startPad = (first.getDay() - (PB.state.settings.weekStartsOn ?? 1) + 7) % 7;
  const days = new Date(year, month + 1, 0).getDate();
  const names = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
  const head = [];
  for (let i = 0; i < 7; i++) head.push(names[(i + (PB.state.settings.weekStartsOn ?? 1)) % 7]);
  let cells = head.map((name) => `<div class="dow">${name[0]}</div>`).join("");
  for (let i = 0; i < startPad; i++) cells += `<div></div>`;
  const rangeStart = PB.state.rangeStart ? ymd(PB.state.rangeStart) : "";
  const rangeEnd = PB.state.rangeEnd ? ymd(new Date(PB.state.rangeEnd.getTime() - 86400000)) : "";
  const today = ymd(new Date());
  for (let day = 1; day <= days; day++) {
    const date = new Date(year, month, day);
    const key = ymd(date);
    const dot = (PB.state.summary || []).some((item) => item.date === key && item.late) ? " ●" : "";
    const cls = [key >= rangeStart && key <= rangeEnd ? "in-range" : "", key === today ? "is-today" : ""].filter(Boolean).join(" ");
    cells += `<button type="button" class="${cls}" data-day="${key}">${day}${dot ? `<span style="color:#a56b12">●</span>` : ""}</button>`;
  }
  host.innerHTML = `
    <div class="row" style="align-items:center;margin-bottom:6px">
      <button type="button" class="icon-btn" id="mini-prev">‹</button>
      <strong style="flex:1;text-align:center">${esc(first.toLocaleDateString([], { month: "long", year: "numeric" }))}</strong>
      <button type="button" class="icon-btn" id="mini-next">›</button>
    </div>
    <div class="mini">${cells}</div>`;
  host.querySelector("#mini-prev").onclick = () => { PB.state.mini = new Date(year, month - 1, 1); renderMini(); refreshSummary(); };
  host.querySelector("#mini-next").onclick = () => { PB.state.mini = new Date(year, month + 1, 1); renderMini(); refreshSummary(); };
  host.querySelectorAll("[data-day]").forEach((button) => {
    button.onclick = () => PB.state.calendar?.gotoDate(button.dataset.day);
  });
}

function renderTeamList() {
  const host = document.getElementById("team-list");
  if (!host) return;
  const groups = [...PB.state.ageGroups].sort((a, b) => a.sortOrder - b.sortOrder || a.maxAge - b.maxAge);
  const teams = [...PB.state.teams].filter((team) => team.active !== false);
  let html = "";
  const buckets = groups.map((group) => ({ group, teams: teams.filter((team) => team.ageGroupId === group.id) })).filter((bucket) => bucket.teams.length);
  const loose = teams.filter((team) => !ageById(team.ageGroupId));
  if (loose.length) buckets.push({ group: { name: "Other" }, teams: loose });
  for (const bucket of buckets) {
    html += `<div class="side-label">${esc(bucket.group.name)}</div>`;
    for (const team of bucket.teams) {
      const hidden = PB.state.hiddenTeams.has(team.id);
      html += `<div class="team-chip" data-id="${esc(team.id)}">
        <input type="checkbox" ${hidden ? "" : "checked"} data-hide="${esc(team.id)}" aria-label="Show ${esc(team.name)}" />
        <span class="swatch" style="background:${esc(team.color || "#2f6a4a")}"></span>
        <span><span class="t-name">${esc(team.shortName || team.name)}</span><div class="t-meta">${esc(teamMeta(team))}</div></span>
      </div>`;
    }
  }
  if (!teams.length) html = `<p class="muted">Add a team, then drag it onto the calendar.</p>`;
  host.innerHTML = html;
  host.querySelectorAll("[data-hide]").forEach((box) => {
    box.onclick = (event) => event.stopPropagation();
    box.onchange = () => {
      if (box.checked) PB.state.hiddenTeams.delete(box.dataset.hide);
      else PB.state.hiddenTeams.add(box.dataset.hide);
      PB.state.calendar?.refetchEvents();
    };
  });
  bindDraggables();
}

function renderFilters() {
  const host = document.getElementById("filters");
  if (!host) return;
  const filters = PB.state.filters;
  host.innerHTML = `
    <label class="field"><span>Place</span><select id="filter-env">
      <option value="">All places</option>
      <option value="indoor" ${filters.env === "indoor" ? "selected" : ""}>Indoor</option>
      <option value="outdoor" ${filters.env === "outdoor" ? "selected" : ""}>Outdoor</option>
    </select></label>
    <label class="field"><span>Surface</span><select id="filter-surface">
      <option value="">Any surface</option>
      ${PB.state.surfaces.map((surface) => `<option value="${esc(surface.id)}" ${filters.surface === surface.id ? "selected" : ""}>${esc(surface.name)}</option>`).join("")}
    </select></label>
    <label class="field"><span>Kind</span><select id="filter-type">
      <option value="">Anything</option>
      ${PB.state.eventTypes.map((type) => `<option value="${esc(type.id)}" ${filters.type === type.id ? "selected" : ""}>${esc(type.name)}</option>`).join("")}
    </select></label>
    <label style="display:block;margin-top:8px"><input type="checkbox" id="late-only" ${filters.lateOnly ? "checked" : ""}/> Early-hour issues only</label>
    <label style="display:block;margin-top:4px"><input type="checkbox" id="hide-cancelled" ${filters.hideCancelled ? "checked" : ""}/> Hide cancelled</label>`;
  host.querySelector("#filter-env").onchange = (event) => { filters.env = event.target.value; PB.state.calendar.refetchEvents(); };
  host.querySelector("#filter-surface").onchange = (event) => { filters.surface = event.target.value; PB.state.calendar.refetchEvents(); };
  host.querySelector("#filter-type").onchange = (event) => { filters.type = event.target.value; PB.state.calendar.refetchEvents(); };
  host.querySelector("#late-only").onchange = (event) => { filters.lateOnly = event.target.checked; PB.state.calendar.refetchEvents(); };
  host.querySelector("#hide-cancelled").onchange = (event) => { filters.hideCancelled = event.target.checked; PB.state.calendar.refetchEvents(); };
}

function renderInspector() {
  const host = document.getElementById("inspector");
  if (!host || PB.state.view !== "calendar") return;
  if (PB.state.formMode === "opening" && PB.state.opening) {
    renderOpening(host, PB.state.opening);
    return;
  }
  const form = PB.state.form;
  if (!form) {
    const late = (PB.state.events || []).filter((event) => event.latePriority).length;
    host.innerHTML = `<div class="side-block"><h2>Details</h2><p class="muted">Drag a team onto the calendar, drag across an empty hour, or import a timetable.</p>${late ? `<p><span class="badge late">${late} early-hour issue${late === 1 ? "" : "s"}</span></p><button type="button" class="btn-ghost" id="goto-plan">Review them</button>` : ""}</div>`;
    const jump = host.querySelector("#goto-plan");
    if (jump) jump.onclick = () => setView("plan");
    return;
  }
  const type = typeById(form.eventTypeId);
  const category = type?.category || "practice";
  const primary = (form.teamIds || [])[0] || "";
  const second = (form.teamIds || [])[1] || "";
  const placeLine = placeSummary(form);
  host.innerHTML = `<form class="side-block" id="event-form">
    <h2>${PB.state.formMode === "create" ? "New booking" : "Booking"}</h2>
    <p class="muted" id="when-label"></p>
    <div id="issue-box"></div>
    <label class="field"><span>Team</span><select name="team">${teamChoices(primary, "Choose a team")}</select></label>
    ${second || PB.state.showSecondTeam ? `<label class="field"><span>Also</span><select name="team2">${teamChoices(second, "No other team")}</select></label>` : `<button type="button" class="btn-ghost" id="add-team">Add another team</button>`}
    <label class="field"><span>What</span><select name="eventTypeId">${PB.state.eventTypes.map((item) => `<option value="${esc(item.id)}" ${item.id === form.eventTypeId ? "selected" : ""}>${esc(item.name)}</option>`).join("")}</select></label>
    ${category === "game" ? `<label class="field"><span>Home or away</span><select name="homeAway"><option value="home" ${form.homeAway !== "away" ? "selected" : ""}>Home</option><option value="away" ${form.homeAway === "away" ? "selected" : ""}>Away</option></select></label>
      <label class="field"><span>Opponent</span><input name="opponent" type="text" value="${esc(form.opponent)}" /></label>` : `<input type="hidden" name="homeAway" value="${esc(form.homeAway || "home")}" />`}
    ${form.homeAway === "away" && category === "game"
      ? `<label class="field"><span>Where</span><input name="awayLocation" type="text" value="${esc(form.awayLocation)}" placeholder="Opponent's field" /></label>`
      : `<label class="field"><span>Place</span><select name="pitchId">${placeChoices(form.pitchId)}</select></label>${placeLine ? `<p class="muted">${esc(placeLine)}</p>` : ""}`}
    <label class="field"><span>Starts</span><input name="start" type="${form.allDay ? "date" : "datetime-local"}" value="${form.allDay ? ymd(form.start) : stampInput(form.start)}" /></label>
    <label class="field"><span>Ends</span><input name="end" type="${form.allDay ? "date" : "datetime-local"}" value="${form.allDay ? ymd(new Date(parseLocal(form.end).getTime() - 86400000)) : stampInput(form.end)}" /></label>
    <button type="button" class="btn-ghost" id="more-toggle">${PB.state.showMore ? "Hide extra fields" : "More"}</button>
    <div id="more-fields" ${PB.state.showMore ? "" : "hidden"}>
      <label class="field"><span>Title</span><input name="title" type="text" value="${esc(form.title)}" placeholder="Filled in from the team if left blank" /></label>
      <label class="field"><span><input type="checkbox" name="allDay" ${form.allDay ? "checked" : ""}/> All day</span></label>
      <label class="field"><span>Status</span><select name="status">${["scheduled", "tentative", "cancelled"].map((status) => `<option ${form.status === status ? "selected" : ""}>${status}</option>`).join("")}</select></label>
      <label class="field"><span>Notes</span><textarea name="notes">${esc(form.notes)}</textarea></label>
      ${PB.state.formMode === "create" ? `<label class="field"><span>Repeat</span><select name="repeat"><option value="none">Does not repeat</option><option value="weekly" ${form.repeat === "weekly" ? "selected" : ""}>Weekly</option><option value="daily" ${form.repeat === "daily" ? "selected" : ""}>Daily</option></select></label>
        <div class="row"><label class="field"><span>Until</span><input name="repeatUntil" type="date" value="${esc(form.repeatUntil || "")}" /></label><label class="field"><span>Or count</span><input name="repeatCount" type="number" min="1" max="60" value="${esc(form.repeatCount || 8)}" /></label></div>` : ""}
      ${form.seriesId ? `<label class="field"><span>Repeating set</span><select name="applyTo"><option value="single">Change this date only</option><option value="series" ${form.applyTo === "series" ? "selected" : ""}>Shift the whole set</option></select></label>` : ""}
    </div>
    <div id="late-box"></div>
    <div id="earlier-box"></div>
    <div class="row">
      <button type="submit" class="btn">Save</button>
      ${PB.state.formMode === "edit" ? `<button type="button" class="btn-ghost" id="delete-event">Delete</button>` : `<button type="button" class="btn-ghost" id="cancel-draft">Cancel</button>`}
    </div>
  </form>`;
  renderLateBox();
  host.querySelector("#when-label").textContent = formatRange(form.start, form.end, form.allDay);
  host.querySelector("#event-form").onsubmit = saveForm;
  host.querySelector("#event-form").onchange = onFormChange;
  host.querySelector("#event-form").oninput = () => { PB.state.dirty = true; };
  const more = host.querySelector("#more-toggle");
  if (more) more.onclick = () => { PB.state.showMore = !PB.state.showMore; renderInspector(); };
  const addTeam = host.querySelector("#add-team");
  if (addTeam) addTeam.onclick = () => { PB.state.showSecondTeam = true; renderInspector(); };
  const cancel = host.querySelector("#cancel-draft");
  if (cancel) cancel.onclick = () => { PB.state.form = null; PB.state.selectedId = null; PB.state.showSecondTeam = false; renderInspector(); };
  const remove = host.querySelector("#delete-event");
  if (remove) remove.onclick = deleteCurrent;
}

function teamChoices(selected, blank) {
  const teams = PB.state.teams.filter((team) => team.active !== false);
  return `<option value="">${esc(blank)}</option>` + teams.map((team) => `<option value="${esc(team.id)}" ${team.id === selected ? "selected" : ""}>${esc(team.name)}</option>`).join("");
}

function placeChoices(selected) {
  const pitches = PB.state.pitches.filter((pitch) => pitch.active !== false);
  return `<option value="">Choose a place</option>` + pitches.map((pitch) => {
    const venue = venueById(pitch.venueId);
    return `<option value="${esc(pitch.id)}" ${pitch.id === selected ? "selected" : ""}>${esc((venue?.name || "Place") + " — " + pitch.name)}</option>`;
  }).join("");
}

function placeSummary(form) {
  const pitch = pitchById(form.pitchId);
  if (!pitch) return "";
  const surface = byId(PB.state.surfaces, pitch.surfaceId);
  const format = byId(PB.state.formats, pitch.formatId);
  const where = pitch.environment === "indoor" ? "Indoor" : pitch.environment === "outdoor" ? "Outdoor" : "";
  return [where, surface?.name, format?.name].filter(Boolean).join(" · ");
}

function renderOpening(host, opening) {
  const use = opening.use === "game" || opening.use === "practice" ? opening.use : "practice";
  const title = opening.kind === "swap" ? "Swap a time" : "Open time";
  const who = opening.kind === "swap"
    ? `Held by ${opening.holder || opening.zone || "another Saskatoon Youth Soccer zone"}.`
    : "Free on the imported timetable.";
  host.innerHTML = `<form class="side-block" id="opening-form">
    <h2>${esc(title)}</h2>
    <p class="muted">${esc(formatRange(opening.start, opening.end, false))}</p>
    <p>${esc(who)} ${esc(opening.place || "")}</p>
    ${opening.notes ? `<p class="muted">${esc(opening.notes)}</p>` : ""}
    <div id="issue-box"></div>
    <label class="field"><span>Team</span><select name="team">${teamChoices(opening.teamId || "", "Choose a team")}</select></label>
    <label class="field"><span>Use</span><select name="use">
      <option value="practice" ${use !== "game" ? "selected" : ""}>Practice</option>
      <option value="game" ${use === "game" ? "selected" : ""}>Game</option>
    </select></label>
    <label class="field"><span>Place</span><select name="pitchId">${placeChoices(opening.pitchId || "")}</select></label>
    <div id="opening-game" ${use === "game" ? "" : "hidden"}>
      <label class="field"><span>Home or away</span><select name="homeAway"><option value="home">Home</option><option value="away">Away</option></select></label>
      <label class="field"><span>Opponent</span><input name="opponent" type="text" value="${esc(opening.holder || "")}" /></label>
    </div>
    <div class="row">
      <button type="submit" class="btn">Book this time</button>
      <button type="button" class="btn-ghost" id="drop-opening">Remove</button>
    </div>
    <button type="button" class="btn-ghost" id="close-opening">Leave it</button>
  </form>`;
  host.querySelector("[name=use]").onchange = (event) => {
    host.querySelector("#opening-game").hidden = event.target.value !== "game";
  };
  host.querySelector("#opening-form").onsubmit = bookOpening;
  host.querySelector("#close-opening").onclick = () => { PB.state.opening = null; PB.state.formMode = "create"; PB.state.selectedId = null; renderInspector(); };
  host.querySelector("#drop-opening").onclick = async () => {
    try {
      await API.deleteOpening(opening.id);
      PB.state.opening = null;
      PB.state.selectedId = null;
      PB.state.formMode = "create";
      PB.state.calendar?.refetchEvents();
      renderInspector();
      toast("Removed from the timetable.");
    } catch (err) { toast(errorText(err)); }
  };
}

async function bookOpening(event) {
  event.preventDefault();
  const opening = PB.state.opening;
  const data = new FormData(event.target);
  try {
    const saved = await API.bookOpening(opening.id, {
      teamId: data.get("team") || "",
      use: data.get("use") || "practice",
      pitchId: data.get("pitchId") || "",
      homeAway: data.get("homeAway") || "home",
      opponent: data.get("opponent") || "",
    });
    PB.state.opening = null;
    PB.state.form = null;
    PB.state.selectedId = saved.events?.[0]?.id || null;
    PB.state.formMode = PB.state.selectedId ? "edit" : "create";
    if (saved.issues?.some((issue) => issue.level === "warning")) toast(saved.issues.find((issue) => issue.level === "warning").message);
    toast("Booked.");
    PB.state.calendar?.refetchEvents();
    renderInspector();
  } catch (err) {
    showIssues(err.data?.issues || [{ level: "error", message: errorText(err) }]);
    toast(errorText(err));
  }
}

function renderLateBox() {
  const form = PB.state.form;
  const host = document.querySelector("#late-box");
  if (!host || !form) return;
  if (!isLateForm(form)) { host.innerHTML = ""; return; }
  const teams = (form.teamIds || []).map(teamById).filter(isPriorityTeam);
  const limit = teams.length ? Math.min(...teams.map(teamWindowEnd)) : minutesOf(PB.state.settings.earlyWindowEnd || "17:30");
  const end = clockLabel(`${pad(Math.floor(limit / 60))}:${pad(limit % 60)}`, PB.state.settings.timeFormat);
  host.innerHTML = `<div class="issue">This starts after ${esc(end)}. Players under ${esc(PB.state.settings.priorityCutoffAge)} are supposed to have the earlier hours.${PB.state.settings.priorityMode === "strict" ? `<label style="display:block;margin-top:6px"><input type="checkbox" name="acknowledgeLate" ${form.acknowledgeLate ? "checked" : ""}/> Schedule this later anyway</label>` : ""}</div><button type="button" class="btn-ghost" id="find-earlier">Find an earlier time</button>`;
  const button = host.querySelector("#find-earlier");
  if (button) button.onclick = findEarlier;
}

function showIssues(issues) {
  const host = document.querySelector("#issue-box");
  if (!host) return;
  host.innerHTML = (issues || []).map((issue) => `<div class="issue ${issue.level === "error" ? "error" : ""}">${esc(issue.message)}</div>`).join("");
}

function readForm(formEl) {
  const form = PB.state.form;
  const data = new FormData(formEl);
  form.title = data.get("title") || "";
  form.eventTypeId = data.get("eventTypeId") || form.eventTypeId;
  form.teamIds = [data.get("team"), data.get("team2")].map((id) => id || "").filter((id, index, all) => id && all.indexOf(id) === index);
  form.homeAway = data.get("homeAway") || form.homeAway || "home";
  form.opponent = data.get("opponent") || "";
  form.awayLocation = data.get("awayLocation") || "";
  if (form.homeAway === "away") {
    form.pitchId = "";
    form.venueId = "";
  } else if (data.has("pitchId")) {
    form.pitchId = data.get("pitchId") || "";
  }
  form.status = data.get("status") || "scheduled";
  form.notes = data.get("notes") || "";
  const wasAllDay = form.allDay;
  form.allDay = !!formEl.querySelector("[name=allDay]")?.checked;
  form.repeat = data.get("repeat") || "none";
  form.repeatUntil = data.get("repeatUntil") || "";
  form.repeatCount = Number(data.get("repeatCount") || 8);
  form.applyTo = data.get("applyTo") || "single";
  form.acknowledgeLate = !!formEl.querySelector("[name=acknowledgeLate]")?.checked;
  const start = String(data.get("start") || "");
  const end = String(data.get("end") || "");
  if (form.allDay) {
    const startDay = start.slice(0, 10);
    const endDay = end.slice(0, 10);
    form.start = startDay + "T00:00:00";
    const endDate = parseLocal(endDay + "T00:00:00");
    if (!wasAllDay || end.includes("T")) endDate.setDate(endDate.getDate() + 1);
    else endDate.setDate(endDate.getDate() + 1);
    form.end = toLocalISO(endDate);
  } else if (start.length <= 10) {
    form.start = start + "T16:00:00";
    form.end = (end.slice(0, 10) || start) + "T17:00:00";
  } else {
    form.start = start.length === 16 ? start + ":00" : start;
    form.end = end.length === 16 ? end + ":00" : end;
  }
  return form;
}

function onFormChange(event) {
  const formEl = event.currentTarget;
  const previousPitch = PB.state.form.pitchId;
  const previousVenue = PB.state.form.venueId;
  const previousAllDay = PB.state.form.allDay;
  readForm(formEl);
  if (event.target.name === "homeAway" && PB.state.form.homeAway === "away") {
    PB.state.form.pitchId = "";
    PB.state.form.venueId = "";
  }
  if (event.target.name === "pitchId" && PB.state.form.pitchId && PB.state.form.pitchId !== previousPitch) {
    const pitch = pitchById(PB.state.form.pitchId);
    if (pitch) applyPitch(PB.state.form, pitch);
  }
  if (!PB.state.form.pitchId) {
    PB.state.form.venueId = "";
  }
  PB.state.dirty = true;
  if (["allDay", "eventTypeId", "homeAway", "pitchId", "team", "team2", "start", "end"].includes(event.target.name) || previousAllDay !== PB.state.form.allDay || previousVenue !== PB.state.form.venueId) {
    renderInspector();
    return;
  }
  renderLateBox();
}

async function saveForm(event) {
  event.preventDefault();
  const form = readForm(event.target);
  try {
    const saved = form.id ? await API.update(form.id, form) : await API.create(form);
    if (form.id) {
      const before = PB.state.events.find((item) => item.id === form.id);
      if (before) pushUndo({ type: "update", before: editable(before) });
    } else {
      pushUndo({ type: "create", ids: (saved.events || []).map((item) => item.id) });
    }
    PB.state.dirty = false;
    PB.state.form = null;
    PB.state.selectedId = saved.events?.[0]?.id || null;
    if (PB.state.selectedId) PB.state.formMode = "edit";
    toast(saved.events?.length > 1 ? `Added ${saved.events.length} bookings.` : "Saved.");
    if (saved.issues?.some((issue) => issue.level === "warning")) toast(saved.issues.find((issue) => issue.level === "warning").message);
    PB.state.calendar?.refetchEvents();
    renderInspector();
  } catch (err) {
    showIssues(err.data?.issues || [{ level: "error", message: errorText(err) }]);
    toast(errorText(err));
  }
}

async function deleteCurrent() {
  const form = PB.state.form;
  if (!form?.id) return;
  const scope = form.seriesId && form.applyTo === "series" ? "series" : "single";
  const ok = await confirmBox("Delete this booking?", scope === "series" ? "This removes every date in the set." : "This removes only this date.", "Delete");
  if (!ok) return;
  try {
    const result = await API.remove(form.id, scope);
    pushUndo({ type: "delete", events: result.deleted || [] });
    PB.state.form = null;
    PB.state.selectedId = null;
    PB.state.calendar?.refetchEvents();
    renderInspector();
    toast("Deleted.");
  } catch (err) {
    toast(errorText(err));
  }
}

async function findEarlier() {
  readForm(document.getElementById("event-form"));
  const host = document.getElementById("earlier-box");
  host.innerHTML = `<p class="muted">Looking…</p>`;
  try {
    const data = await API.earlier(PB.state.form);
    const slots = data.slots || [];
    if (!slots.length) { host.innerHTML = `<p class="muted">No open pitch in the next week.</p>`; return; }
    host.innerHTML = slots.map((slot, index) => `<button type="button" class="list-btn" data-slot="${index}">${esc(slot.label)}${slot.late ? " · late" : ""}</button>`).join("");
    host.querySelectorAll("[data-slot]").forEach((button) => button.onclick = () => {
      const slot = slots[Number(button.dataset.slot)];
      const pitch = pitchById(slot.pitchId);
      PB.state.form.start = slot.start;
      PB.state.form.end = slot.end;
      if (pitch) applyPitch(PB.state.form, pitch);
      renderInspector();
    });
  } catch (err) {
    host.innerHTML = "";
    toast(errorText(err));
  }
}

function renderStatus() {
  const late = (PB.state.events || []).filter((event) => event.latePriority && matchesFilters(event)).length;
  const shown = (PB.state.events || []).filter(matchesFilters).length;
  document.getElementById("status").innerHTML = `<span>${shown} in this view${late ? ` · ${late} after the early window` : ""}</span><span>Local only · ${esc(PB.state.dataDir || "")}</span>`;
}

function renderSetup() {
  const settings = PB.state.settings;
  const early = (PB.state.ageGroups || []).filter((group) => group.maxAge < (settings.priorityCutoffAge || 11)).map((group) => group.name);
  document.getElementById("setup").hidden = false;
  document.getElementById("shell").hidden = true;
  document.getElementById("setup").innerHTML = `<div class="setup-card">
    <div class="word">Pitchboard</div>
    <h1>Set the week’s rhythm.</h1>
    <p>Practices, camps, and games stay on this computer. Players under the cutoff get the earlier hours before older teams are placed.</p>
    <form id="setup-form">
      <div class="row">
        <label class="field"><span>Club name</span><input name="clubName" required value="${esc(settings.clubName || "Aurora Soccer Club")}" placeholder="Aurora Soccer Club" /></label>
        <label class="field"><span>Season</span><input name="seasonName" value="${esc(settings.seasonName || "")}" /></label>
      </div>
      <div class="row">
        <label class="field"><span>Season starts</span><input name="seasonStart" type="date" value="${esc(settings.seasonStart || "")}" /></label>
        <label class="field"><span>Season ends</span><input name="seasonEnd" type="date" value="${esc(settings.seasonEnd || "")}" /></label>
      </div>
      <div class="row">
        <label class="field"><span>Priority age</span><input name="priorityCutoffAge" type="number" min="4" max="30" value="${esc(settings.priorityCutoffAge || 11)}" /></label>
        <label class="field"><span>Early window</span><div class="row"><input name="earlyWindowStart" type="time" value="${esc(settings.earlyWindowStart || "08:00")}" /><input name="earlyWindowEnd" type="time" value="${esc(settings.earlyWindowEnd || "17:30")}" /></div></label>
      </div>
      <label class="field"><span>If a young team is booked later</span><select name="priorityMode"><option value="warn">Allow it and mark it</option><option value="strict">Ask before saving</option></select></label>
      <p class="muted" id="priority-preview">Early priority right now: ${esc(early.join(", ") || "none yet")}.</p>
      <div class="choice">
        <button type="submit" data-sample="1"><strong>Start with the Aurora sample</strong><span class="muted">Aurora Soccer Club’s week across SaskTel, Henk Ruys, and Forest Park. Several teams share each age group, Academy has its own evening hours, and one U8 practice is late so you can see the warning.</span></button>
        <button type="submit" data-sample="0"><strong>Start with an empty club</strong><span class="muted">Age groups and surfaces are ready. Add your own teams and pitches.</span></button>
      </div>
    </form>
  </div>`;
  const form = document.getElementById("setup-form");
  form.onsubmit = async (event) => {
    event.preventDefault();
    const data = new FormData(form);
    const sample = event.submitter?.dataset.sample === "1";
    const next = { ...PB.state.settings, clubName: data.get("clubName"), seasonName: data.get("seasonName"), seasonStart: data.get("seasonStart"), seasonEnd: data.get("seasonEnd"), priorityCutoffAge: Number(data.get("priorityCutoffAge")), earlyWindowStart: data.get("earlyWindowStart"), earlyWindowEnd: data.get("earlyWindowEnd"), priorityMode: data.get("priorityMode") };
    try {
      adopt(await API.setup({ settings: next, sample }));
      openApp();
    } catch (err) { toast(errorText(err)); }
  };
}

function renderTeams() {
  const page = document.getElementById("page");
  const editing = PB.state.teamDraft;
  page.innerHTML = `<h1>Teams</h1><p class="muted">Age and the priority setting decide who is offered the earlier hours.</p><button type="button" class="btn" id="add-team">Add a team</button><div class="card" id="team-editor"></div><div id="team-table"></div>`;
  page.querySelector("#add-team").onclick = () => { PB.state.teamDraft = { priority: "auto", active: true, color: "#2f6a4a", ageGroupId: PB.state.ageGroups[0]?.id || "" }; renderTeams(); };
  const editor = page.querySelector("#team-editor");
  if (!editing) editor.innerHTML = `<p class="muted">Select a team to edit it.</p>`;
  else editor.innerHTML = teamEditor(editing);
  page.querySelector("#team-table").innerHTML = PB.state.teams.map((team) => `<button type="button" class="list-btn" data-team="${esc(team.id)}"><span class="swatch" style="background:${esc(team.color || "#888")};display:inline-block"></span> ${esc(team.name)} <span class="muted">${esc(teamMeta(team))}</span></button>`).join("") || `<p>No teams yet.</p>`;
  page.querySelectorAll("[data-team]").forEach((button) => button.onclick = () => { PB.state.teamDraft = { ...teamById(button.dataset.team) }; renderTeams(); });
  const form = page.querySelector("#team-form");
  if (form) {
    form.onsubmit = saveTeamForm;
    page.querySelector("#delete-team").onclick = async () => {
      if (!PB.state.teamDraft.id) { PB.state.teamDraft = null; renderTeams(); return; }
      if (!await confirmBox("Delete team?", "Bookings that use this team must be removed first.", "Delete")) return;
      try { adopt(await API.deleteTeam(PB.state.teamDraft.id)); PB.state.teamDraft = null; renderTeams(); renderTeamList(); }
      catch (err) { toast(errorText(err)); }
    };
  }
}

function teamEditor(team) {
  return `<form id="team-form"><div class="row">
      <label class="field"><span>Name</span><input name="name" required value="${esc(team.name || "")}" /></label>
      <label class="field"><span>Short name</span><input name="shortName" value="${esc(team.shortName || "")}" /></label>
    </div>
    <div class="row">
      <label class="field"><span>Age group</span><select name="ageGroupId">${PB.state.ageGroups.map((group) => `<option value="${esc(group.id)}" ${group.id === team.ageGroupId ? "selected" : ""}>${esc(group.name)} (${group.minAge}–${group.maxAge})</option>`).join("")}</select></label>
      <label class="field"><span>Early hours</span><select name="priority"><option value="auto" ${team.priority !== "always" && team.priority !== "never" ? "selected" : ""}>Follow the age cutoff</option><option value="always" ${team.priority === "always" ? "selected" : ""}>Always give early hours</option><option value="never" ${team.priority === "never" ? "selected" : ""}>Never prioritize</option></select></label>
    </div>
    <div class="row">
      <label class="field"><span>Coach</span><input name="coach" value="${esc(team.coach || "")}" /></label>
      <label class="field"><span>Phone</span><input name="coachPhone" value="${esc(team.coachPhone || "")}" /></label>
      <label class="field"><span>Email</span><input name="coachEmail" type="email" value="${esc(team.coachEmail || "")}" /></label>
    </div>
    <div class="row">
      <label class="field"><span>Color</span><input name="color" type="color" value="${esc(team.color || "#2f6a4a")}" /></label>
      <label class="field"><span>Preferred location</span><select name="preferredVenueId"><option value="">Any</option>${PB.state.venues.map((venue) => `<option value="${esc(venue.id)}" ${venue.id === team.preferredVenueId ? "selected" : ""}>${esc(venue.name)}</option>`).join("")}</select></label>
      <label class="field"><span>Preferred surface</span><select name="preferredSurfaceId"><option value="">Any</option>${PB.state.surfaces.map((surface) => `<option value="${esc(surface.id)}" ${surface.id === team.preferredSurfaceId ? "selected" : ""}>${esc(surface.name)}</option>`).join("")}</select></label>
    </div>
    <label class="field"><span>Notes</span><textarea name="notes">${esc(team.notes || "")}</textarea></label>
    <h3>Online systems</h3>
    <p class="muted">RAMP InterActive needs the numeric team id from your RAMP team list. TeamLinkt, TeamSnap, and GameChanger can use a different name when it does not match the name above.</p>
    <div class="row">
      <label class="field"><span>RAMP team id</span><input name="ramp" value="${esc(team.systems?.ramp || "")}" inputmode="numeric" /></label>
      <label class="field"><span>TeamLinkt name</span><input name="teamlinkt" value="${esc(team.systems?.teamlinkt || "")}" /></label>
    </div>
    <div class="row">
      <label class="field"><span>TeamSnap name</span><input name="teamsnap" value="${esc(team.systems?.teamsnap || "")}" /></label>
      <label class="field"><span>GameChanger name</span><input name="gamechanger" value="${esc(team.systems?.gamechanger || "")}" /></label>
    </div>
    <label><input type="checkbox" name="active" ${team.active !== false ? "checked" : ""}/> Active</label>
    <div class="row" style="margin-top:10px"><button class="btn" type="submit">Save team</button><button class="btn-ghost" id="delete-team" type="button">${team.id ? "Delete" : "Cancel"}</button></div>
  </form>`;
}

async function saveTeamForm(event) {
  event.preventDefault();
  const data = new FormData(event.target);
  const body = { ...PB.state.teamDraft, name: data.get("name"), shortName: data.get("shortName"), ageGroupId: data.get("ageGroupId"), priority: data.get("priority"), coach: data.get("coach"), coachPhone: data.get("coachPhone"), coachEmail: data.get("coachEmail"), color: data.get("color"), preferredVenueId: data.get("preferredVenueId"), preferredSurfaceId: data.get("preferredSurfaceId"), notes: data.get("notes"), active: data.get("active") === "on", systems: { ramp: data.get("ramp") || "", teamlinkt: data.get("teamlinkt") || "", teamsnap: data.get("teamsnap") || "", gamechanger: data.get("gamechanger") || "" } };
  try {
    adopt(await API.saveTeam(body));
    PB.state.teamDraft = null;
    toast("Team saved.");
    renderTeams();
    renderTeamList();
  } catch (err) { toast(errorText(err)); }
}

function renderPlaces() {
  const page = document.getElementById("page");
  page.innerHTML = `<h1>Places and pitches</h1><p class="muted">A place can hold indoor turf, a hardcourt, a gym, outdoor grass, and outdoor turf. The pitch records the surface and the game size.</p>
    <button type="button" class="btn" id="add-venue">Add a location</button>
    <div id="place-list"></div>`;
  page.querySelector("#add-venue").onclick = () => { PB.state.venueDraft = { active: true, days: [0, 1, 2, 3, 4, 5, 6], openTime: "08:00", closeTime: "22:00" }; PB.state.pitchDraft = null; renderPlaces(); };
  const list = page.querySelector("#place-list");
  list.innerHTML = (PB.state.venueDraft ? venueCard(PB.state.venueDraft, true) : "") + PB.state.venues.map((venue) => venueCard(venue, false)).join("");
  list.querySelectorAll("[data-edit-venue]").forEach((button) => button.onclick = () => { PB.state.venueDraft = { ...venueById(button.dataset.editVenue) }; renderPlaces(); });
  list.querySelectorAll("[data-add-pitch]").forEach((button) => button.onclick = () => { PB.state.pitchDraft = { venueId: button.dataset.addPitch, active: true, environment: "indoor", days: [] }; renderPlaces(); });
  const venueForm = list.querySelector("#venue-form");
  if (venueForm) venueForm.onsubmit = saveVenueForm;
  const pitchForm = list.querySelector("#pitch-form");
  if (pitchForm) pitchForm.onsubmit = savePitchForm;
  list.querySelectorAll("[data-edit-pitch]").forEach((button) => button.onclick = () => { PB.state.pitchDraft = { ...pitchById(button.dataset.editPitch) }; renderPlaces(); });
  const deleteVenue = list.querySelector("#delete-venue");
  if (deleteVenue) deleteVenue.onclick = async () => {
    if (!PB.state.venueDraft?.id) { PB.state.venueDraft = null; renderPlaces(); return; }
    if (!await confirmBox("Delete location?", "Remove its pitches first if it still has any.", "Delete")) return;
    try { adopt(await API.deleteVenue(PB.state.venueDraft.id)); PB.state.venueDraft = null; renderPlaces(); } catch (err) { toast(errorText(err)); }
  };
  const deletePitch = list.querySelector("#delete-pitch");
  if (deletePitch) deletePitch.onclick = async () => {
    if (!PB.state.pitchDraft?.id) { PB.state.pitchDraft = null; renderPlaces(); return; }
    if (!await confirmBox("Delete pitch?", "Bookings on this pitch must be moved first.", "Delete")) return;
    try { adopt(await API.deletePitch(PB.state.pitchDraft.id)); PB.state.pitchDraft = null; renderPlaces(); } catch (err) { toast(errorText(err)); }
  };
}

function venueCard(venue, editing) {
  const pitches = PB.state.pitches.filter((pitch) => pitch.venueId === venue.id);
  const body = editing ? `<form id="venue-form">${venueFields(venue)}<div class="row"><button class="btn" type="submit">Save location</button><button class="btn-ghost" id="delete-venue" type="button">${venue.id ? "Delete" : "Cancel"}</button></div></form>` : `<button type="button" class="btn-ghost" data-edit-venue="${esc(venue.id)}">Edit</button> <button type="button" class="btn-ghost" data-add-pitch="${esc(venue.id)}">Add pitch</button>`;
  const pitchEdit = PB.state.pitchDraft && PB.state.pitchDraft.venueId === venue.id ? `<form id="pitch-form">${pitchFields(PB.state.pitchDraft)}<div class="row"><button class="btn" type="submit">Save pitch</button><button class="btn-ghost" id="delete-pitch" type="button">${PB.state.pitchDraft.id ? "Delete pitch" : "Cancel"}</button></div></form>` : "";
  return `<div class="card"><h2>${esc(venue.name || "New location")}</h2><p class="muted">${esc(venue.address || "")}</p>${body}${pitchEdit}<div>${pitches.map((pitch) => `<button type="button" class="list-btn" data-edit-pitch="${esc(pitch.id)}">${esc(pitch.name)} · ${esc(pitch.environment)} · ${esc(byId(PB.state.surfaces, pitch.surfaceId)?.name || "")} · ${esc(byId(PB.state.formats, pitch.formatId)?.name || "")}</button>`).join("")}</div></div>`;
}

function venueFields(venue) {
  return `<div class="row"><label class="field"><span>Name</span><input name="name" required value="${esc(venue.name || "")}" /></label><label class="field"><span>Address</span><input name="address" value="${esc(venue.address || "")}" /></label></div>
    <div class="row"><label class="field"><span>Opens</span><input name="openTime" type="time" value="${esc(venue.openTime || "08:00")}" /></label><label class="field"><span>Closes</span><input name="closeTime" type="time" value="${esc(venue.closeTime || "22:00")}" /></label></div>
    <div class="field"><span>Days open</span><div class="checks">${dayNames().map((name, index) => `<label><input type="checkbox" name="day" value="${index}" ${(venue.days || []).includes(index) ? "checked" : ""}/> ${name}</label>`).join("")}</div></div>
    <label class="field"><span>Notes</span><textarea name="notes">${esc(venue.notes || "")}</textarea></label>
    <h3>Online systems</h3>
    <p class="muted">RAMP location id is the numeric id of the arena or field. A pitch can override it. TeamLinkt uses this name when it differs from the location name.</p>
    <div class="row">
      <label class="field"><span>RAMP location id</span><input name="ramp" value="${esc(venue.systems?.ramp || "")}" inputmode="numeric" /></label>
      <label class="field"><span>TeamLinkt location</span><input name="teamlinkt" value="${esc(venue.systems?.teamlinkt || "")}" /></label>
    </div>
    <label><input type="checkbox" name="active" ${venue.active !== false ? "checked" : ""}/> Active</label>`;
}

function pitchFields(pitch) {
  return `<div class="row"><label class="field"><span>Pitch name</span><input name="name" required value="${esc(pitch.name || "")}" /></label>
    <label class="field"><span>Indoor or outdoor</span><select name="environment"><option value="indoor" ${pitch.environment !== "outdoor" ? "selected" : ""}>Indoor</option><option value="outdoor" ${pitch.environment === "outdoor" ? "selected" : ""}>Outdoor</option></select></label></div>
    <div class="row"><label class="field"><span>Surface</span><select name="surfaceId">${PB.state.surfaces.map((surface) => `<option value="${esc(surface.id)}" ${surface.id === pitch.surfaceId ? "selected" : ""}>${esc(surface.name)}</option>`).join("")}</select></label>
    <label class="field"><span>Pitch type</span><select name="formatId">${PB.state.formats.map((format) => `<option value="${esc(format.id)}" ${format.id === pitch.formatId ? "selected" : ""}>${esc(format.name)}</option>`).join("")}</select></label></div>
    <label class="field"><span>Notes</span><textarea name="notes">${esc(pitch.notes || "")}</textarea></label>
    <label class="field"><span>RAMP location id</span><input name="ramp" value="${esc(pitch.systems?.ramp || "")}" inputmode="numeric" placeholder="Leave blank to use the location id" /></label>
    <label><input type="checkbox" name="active" ${pitch.active !== false ? "checked" : ""}/> Active</label>`;
}

async function saveVenueForm(event) {
  event.preventDefault();
  const data = new FormData(event.target);
  const body = { ...PB.state.venueDraft, name: data.get("name"), address: data.get("address"), openTime: data.get("openTime"), closeTime: data.get("closeTime"), notes: data.get("notes"), days: [...event.target.querySelectorAll("[name=day]:checked")].map((box) => Number(box.value)), active: data.get("active") === "on", systems: { ...(PB.state.venueDraft.systems || {}), ramp: data.get("ramp") || "", teamlinkt: data.get("teamlinkt") || "" } };
  try { adopt(await API.saveVenue(body)); PB.state.venueDraft = null; toast("Location saved."); renderPlaces(); } catch (err) { toast(errorText(err)); }
}

async function savePitchForm(event) {
  event.preventDefault();
  const data = new FormData(event.target);
  const body = { ...PB.state.pitchDraft, name: data.get("name"), environment: data.get("environment"), surfaceId: data.get("surfaceId"), formatId: data.get("formatId"), notes: data.get("notes"), active: data.get("active") === "on", systems: { ...(PB.state.pitchDraft.systems || {}), ramp: data.get("ramp") || "" } };
  try { adopt(await API.savePitch(body)); PB.state.pitchDraft = null; toast("Pitch saved."); renderPlaces(); } catch (err) { toast(errorText(err)); }
}

function renderPlan() {
  const page = document.getElementById("page");
  const start = PB.state.rangeStart ? ymd(PB.state.rangeStart) : ymd(new Date());
  const endDefault = PB.state.settings.seasonEnd || ymd(new Date(Date.now() + 56 * 86400000));
  page.innerHTML = `<h1>Plan the week</h1>
    <p>Younger teams are placed first, as early as the pitches allow. Older teams fill in after ${esc(clockLabel(PB.state.settings.earlyWindowEnd, PB.state.settings.timeFormat))} when they can. An age group with its own hours, such as Academy, is placed inside that window instead.</p>
    <form id="plan-form" class="card">
      <div class="field"><span>Teams</span><div class="checks">${PB.state.teams.filter((team) => team.active !== false).map((team) => `<label><input type="checkbox" name="team" value="${esc(team.id)}" ${isPriorityTeam(team) ? "checked" : ""}/> ${esc(team.shortName || team.name)}</label>`).join("")}</div></div>
      <div class="row">
        <label class="field"><span>From</span><input name="rangeStart" type="date" value="${esc(start)}" /></label>
        <label class="field"><span>Through</span><input name="rangeEnd" type="date" value="${esc(endDefault)}" /></label>
        <label class="field"><span>Type</span><select name="eventTypeId">${PB.state.eventTypes.map((type) => `<option value="${esc(type.id)}">${esc(type.name)}</option>`).join("")}</select></label>
      </div>
      <div class="field"><span>Days</span><div class="checks">${dayNames().map((name, index) => `<label><input type="checkbox" name="day" value="${index}" ${index >= 1 && index <= 4 ? "checked" : ""}/> ${name}</label>`).join("")}</div></div>
      <div class="row">
        <label class="field"><span>Sessions a week</span><input name="sessionsPerWeek" type="number" min="1" max="7" value="1" /></label>
        <label class="field"><span>Minutes (blank uses each age group)</span><input name="durationMinutes" type="number" min="15" step="5" placeholder="60" /></label>
      </div>
      <button class="btn" type="submit">Preview</button>
    </form>
    <div id="plan-results"></div>
    <div class="card"><h2>Early-hour issues</h2><p class="muted">Young teams in the visible calendar range who start after the early window.</p><button type="button" class="btn-ghost" id="load-fixes">Look for earlier pitches</button><div id="fix-list"></div></div>
    <div class="card"><h2>Copy a week</h2><p class="muted">Copies whatever is currently on screen forward, and skips anything that would double-book a pitch.</p><button type="button" class="btn-ghost" id="copy-week">Copy this view forward one week</button></div>`;
  page.querySelector("#plan-form").onsubmit = previewPlan;
  page.querySelector("#load-fixes").onclick = loadFixes;
  page.querySelector("#copy-week").onclick = copyVisibleWeek;
}

async function previewPlan(event) {
  event.preventDefault();
  const data = new FormData(event.target);
  const body = {
    teamIds: [...event.target.querySelectorAll("[name=team]:checked")].map((box) => box.value),
    eventTypeId: data.get("eventTypeId"),
    rangeStart: data.get("rangeStart"),
    rangeEnd: data.get("rangeEnd"),
    daysOfWeek: [...event.target.querySelectorAll("[name=day]:checked")].map((box) => Number(box.value)),
    sessionsPerWeek: Number(data.get("sessionsPerWeek") || 1),
    durationMinutes: Number(data.get("durationMinutes") || 0),
  };
  const host = document.getElementById("plan-results");
  host.innerHTML = `<p class="muted">Building a draft…</p>`;
  try {
    const result = await API.suggest(body);
    PB.state.plan = result;
    const rows = result.proposals || [];
    host.innerHTML = `<div class="card"><h2>${rows.length} proposed</h2>
      ${rows.map((item, index) => `<label class="list-btn"><input type="checkbox" data-prop="${index}" checked/> ${esc(item.event.title)} · ${esc(formatRange(item.event.start, item.event.end, false))} · ${esc(item.event.pitchId ? (pitchById(item.event.pitchId)?.name || "") : "")} ${(item.warnings || []).map((warning) => `<div class="issue">${esc(warning)}</div>`).join("")}</label>`).join("")}
      ${(result.unscheduled || []).map((item) => `<div class="issue">${esc(item.team)}: ${esc(item.reason)}</div>`).join("")}
      <button type="button" class="btn" id="commit-plan">Add selected</button></div>`;
    host.querySelector("#commit-plan").onclick = commitPlan;
  } catch (err) {
    host.innerHTML = "";
    toast(errorText(err));
  }
}

async function commitPlan() {
  const chosen = [...document.querySelectorAll("[data-prop]:checked")].map((box) => PB.state.plan.proposals[Number(box.dataset.prop)].event);
  if (!chosen.length) { toast("Select at least one."); return; }
  try {
    const saved = await API.batch(chosen, true);
    pushUndo({ type: "create", ids: (saved.events || []).map((event) => event.id) });
    toast(`Added ${saved.events.length}.`);
    PB.state.calendar?.refetchEvents();
  } catch (err) { toast(errorText(err)); }
}

async function loadFixes() {
  const host = document.getElementById("fix-list");
  host.innerHTML = `<p class="muted">Checking…</p>`;
  try {
    const data = await API.fixes(toLocalISO(PB.state.rangeStart || new Date()), toLocalISO(PB.state.rangeEnd || new Date()));
    const proposals = data.proposals || [];
    if (!proposals.length) { host.innerHTML = `<p>No earlier opening in that range.</p>`; return; }
    host.innerHTML = proposals.map((item, index) => `<div class="card"><p>${esc(item.summary)}</p><button type="button" class="btn" data-fix="${index}">Apply</button></div>`).join("");
    host.querySelectorAll("[data-fix]").forEach((button) => button.onclick = async () => {
      try {
        await API.applyFix(proposals[Number(button.dataset.fix)]);
        toast("Updated.");
        PB.state.calendar?.refetchEvents();
        loadFixes();
      } catch (err) { toast(errorText(err)); }
    });
  } catch (err) { host.innerHTML = ""; toast(errorText(err)); }
}

async function copyVisibleWeek() {
  if (!PB.state.rangeStart) return;
  try {
    const result = await API.copyWeek(toLocalISO(PB.state.rangeStart), toLocalISO(PB.state.rangeEnd), 1);
    toast(`Copied ${result.events.length}. Skipped ${result.skipped.length}.`);
    if (result.skipped[0]) toast(result.skipped[0]);
    PB.state.calendar?.gotoDate(new Date(PB.state.rangeStart.getTime() + 7 * 86400000));
  } catch (err) { toast(errorText(err)); }
}

function renderSettings() {
  const settings = PB.state.settings;
  const page = document.getElementById("page");
  const early = PB.state.ageGroups.filter((group) => group.maxAge < settings.priorityCutoffAge).map((group) => group.name);
  const theme = settings.theme || auroraTheme();
  page.innerHTML = `<h1>Settings</h1>
    <form id="settings-form" class="card">
      <h2>Look</h2>
      <p class="muted">Aurora Soccer is the starting point: black, green, Oswald, and Merriweather, taken from aurorasoccer.ca. Change any of it for another club.</p>
      <div class="swatches">
        ${themeField("Header", "header", theme.header)}
        ${themeField("Header text", "headerInk", theme.headerInk)}
        ${themeField("Side rail", "rail", theme.rail)}
        ${themeField("Accent", "accent", theme.accent)}
        ${themeField("Accent text", "accentInk", theme.accentInk)}
        ${themeField("Page", "paper", theme.paper)}
        ${themeField("Cards", "card", theme.card)}
        ${themeField("Text", "ink", theme.ink)}
        ${themeField("Muted text", "muted", theme.muted)}
        ${themeField("Lines", "line", theme.line)}
      </div>
      <div class="row">
        <label class="field"><span>Headings</span><select name="displayFont">${themeFonts.map((font) => `<option ${theme.displayFont === font ? "selected" : ""}>${esc(font)}</option>`).join("")}</select></label>
        <label class="field"><span>Body</span><select name="bodyFont">${themeFonts.map((font) => `<option ${theme.bodyFont === font ? "selected" : ""}>${esc(font)}</option>`).join("")}</select></label>
      </div>
      <button type="button" class="btn-ghost" id="aurora-theme">Use Aurora colours</button>
      <h2>Club</h2>
      <div class="row"><label class="field"><span>Name</span><input name="clubName" value="${esc(settings.clubName)}" /></label><label class="field"><span>Season</span><input name="seasonName" value="${esc(settings.seasonName)}" /></label><label class="field"><span>Our zone</span><select name="zoneName">${zoneOptions(settings.zoneName)}</select></label></div>
      <div class="row"><label class="field"><span>Starts</span><input type="date" name="seasonStart" value="${esc(settings.seasonStart)}" /></label><label class="field"><span>Ends</span><input type="date" name="seasonEnd" value="${esc(settings.seasonEnd)}" /></label><label class="field"><span>Time label</span><input name="timezoneLabel" value="${esc(settings.timezoneLabel)}" /></label></div>
      <h2>Calendar</h2>
      <div class="row">
        <label class="field"><span>Week starts</span><select name="weekStartsOn">${dayNames().map((name, index) => `<option value="${index}" ${settings.weekStartsOn === index ? "selected" : ""}>${name}</option>`).join("")}</select></label>
        <label class="field"><span>Time</span><select name="timeFormat"><option value="12" ${settings.timeFormat !== "24" ? "selected" : ""}>12-hour</option><option value="24" ${settings.timeFormat === "24" ? "selected" : ""}>24-hour</option></select></label>
        <label class="field"><span>Default view</span><select name="defaultView">${[["timeGridDay", "Day"], ["timeGridWorkWeek", "Work week"], ["timeGridWeek", "Week"], ["dayGridMonth", "Month"]].map(([id, label]) => `<option value="${id}" ${settings.defaultView === id ? "selected" : ""}>${label}</option>`).join("")}</select></label>
      </div>
      <div class="row">
        <label class="field"><span>Day starts</span><input name="dayStartHour" type="number" min="0" max="20" value="${esc(settings.dayStartHour)}" /></label>
        <label class="field"><span>Day ends</span><input name="dayEndHour" type="number" min="1" max="24" value="${esc(settings.dayEndHour)}" /></label>
        <label class="field"><span>Slot minutes</span><select name="slotMinutes">${[10, 15, 20, 30, 60].map((n) => `<option ${settings.slotMinutes === n ? "selected" : ""}>${n}</option>`).join("")}</select></label>
      </div>
      <label><input type="checkbox" name="showWeekends" ${settings.showWeekends ? "checked" : ""}/> Show weekends in the week view</label>
      <h2>Early hours</h2>
      <p class="muted">A team is under the cutoff when its age group’s oldest age is lower than this number. U10 (max age 10) qualifies at 11. U12 does not.</p>
      <div class="row">
        <label class="field"><span>Cutoff age</span><input name="priorityCutoffAge" type="number" min="4" max="30" value="${esc(settings.priorityCutoffAge)}" /></label>
        <label class="field"><span>Window starts</span><input name="earlyWindowStart" type="time" value="${esc(settings.earlyWindowStart)}" /></label>
        <label class="field"><span>Window ends</span><input name="earlyWindowEnd" type="time" value="${esc(settings.earlyWindowEnd)}" /></label>
      </div>
      <label class="field"><span>When someone books a young team later</span><select name="priorityMode"><option value="warn" ${settings.priorityMode !== "strict" ? "selected" : ""}>Allow it and mark it</option><option value="strict" ${settings.priorityMode === "strict" ? "selected" : ""}>Require confirmation</option></select></label>
      <div class="field"><span>Rule applies on</span><div class="checks">${dayNames().map((name, index) => `<label><input type="checkbox" name="priorityDay" value="${index}" ${(settings.priorityDays || []).includes(index) ? "checked" : ""}/> ${name}</label>`).join("")}</div></div>
      <div class="field"><span>And for</span><div class="checks">${["practice", "camp", "game", "other"].map((cat) => `<label><input type="checkbox" name="priorityCat" value="${cat}" ${(settings.priorityCategories || []).includes(cat) ? "checked" : ""}/> ${cat}</label>`).join("")}</div></div>
      <p>Early priority with these ages: <strong>${esc(early.join(", ") || "nobody")}</strong>. An age group can turn that on or off, or set its own hours, such as Academy evenings.</p>
      <label><input type="checkbox" name="olderTeamsAvoidEarly" ${settings.olderTeamsAvoidEarly ? "checked" : ""}/> When planning, keep older teams out of the early window if a later pitch is free</label><br/>
      <label><input type="checkbox" name="legendEnabled" ${settings.legendEnabled ? "checked" : ""}/> Show the early-window note on the calendar</label>
      <h2>Conflicts</h2>
      <label><input type="checkbox" name="blockPitchConflicts" ${settings.blockPitchConflicts ? "checked" : ""}/> One booking per pitch</label><br/>
      <label><input type="checkbox" name="blockTeamConflicts" ${settings.blockTeamConflicts ? "checked" : ""}/> A team cannot be in two places at once</label><br/>
      <label><input type="checkbox" name="warnCoachConflicts" ${settings.warnCoachConflicts ? "checked" : ""}/> Warn when one coach is booked twice</label><br/>
      <label><input type="checkbox" name="blockBlackouts" ${settings.blockBlackouts ? "checked" : ""}/> Refuse bookings on closure dates</label>
      <div style="margin-top:12px"><button class="btn" type="submit">Save settings</button></div>
    </form>
    <div class="card"><h2>Age groups</h2><p class="muted">Several teams can share one group. Set a program such as Development, City Rec, Alliance, or Academy, and optional hours that the planner will prefer for every team in the group.</p><div id="age-editor"></div><button type="button" class="btn-ghost" id="add-age">Add age group</button></div>
    <div class="card"><h2>Exports to other systems</h2><p class="muted">RAMP InterActive, TeamLinkt, TeamSnap, and GameChanger each get a file in the shape that system asks for. A saved export can limit which kinds of bookings are included. Put RAMP’s numeric ids on each team and location.</p><div id="export-editor"></div><button type="button" class="btn-ghost" id="add-export">Save an export</button></div>
    <div class="card"><h2>Surfaces</h2><div id="surface-editor"></div><button type="button" class="btn-ghost" id="add-surface">Add surface</button></div>
    <div class="card"><h2>Pitch types</h2><div id="format-editor"></div><button type="button" class="btn-ghost" id="add-format">Add pitch type</button></div>
    <div class="card"><h2>Event types</h2><div id="type-editor"></div><button type="button" class="btn-ghost" id="add-type">Add type</button></div>
    <div class="card"><h2>Closures</h2><div id="blackout-editor"></div><button type="button" class="btn-ghost" id="add-blackout">Add a closure</button></div>
    <div class="card"><h2>This computer</h2>
      <p>The schedule is in <code>${esc(PB.state.dataDir)}</code>. Copy that folder with Pitchboard.exe to move the club to another Windows computer.</p>
      <button type="button" class="btn-ghost" id="backup">Download backup</button>
      <label class="btn-ghost">Restore backup<input id="restore" type="file" accept="application/json,.json" hidden /></label>
      <button type="button" class="btn-ghost" id="load-sample">Replace with the sample club</button>
      <button type="button" class="btn-ghost" id="load-blank">Remove teams and bookings</button>
      <button type="button" class="btn-ghost" id="exit-app">Exit Pitchboard</button>
    </div>`;
  page.querySelector("#settings-form").onsubmit = saveSettingsForm;
  page.querySelector("#aurora-theme").onclick = () => {
    const theme = auroraTheme();
    applyTheme(theme);
    Object.entries(theme).forEach(([key, value]) => {
      const input = page.querySelector(`[name="${key}"]`);
      if (input) input.value = value;
    });
  };
  page.querySelectorAll(".swatches input, [name=displayFont], [name=bodyFont]").forEach((input) => {
    input.addEventListener("input", () => applyTheme(readTheme(page)));
  });
  drawAges(); drawExports(); drawSurfaces(); drawFormats(); drawTypes(); drawBlackouts();
  page.querySelector("#add-age").onclick = async () => { try { adopt(await API.saveAge({ name: "New group", minAge: 8, maxAge: 9, preferredFormatId: PB.state.formats[0]?.id || "", defaultPracticeMinutes: 60, defaultGameMinutes: 60, sortOrder: PB.state.ageGroups.length + 1, program: "", priority: "auto", sessionsPerWeek: 1 })); drawAges(); } catch (err) { toast(errorText(err)); } };
  page.querySelector("#add-export").onclick = async () => { try { adopt(await API.saveExportProfile({ name: "RAMP practices", system: "ramp", categories: ["practice"] })); drawExports(); } catch (err) { toast(errorText(err)); } };
  page.querySelector("#add-surface").onclick = async () => { try { adopt(await API.saveSurface({ name: "New surface", environment: "outdoor" })); drawSurfaces(); } catch (err) { toast(errorText(err)); } };
  page.querySelector("#add-format").onclick = async () => { try { adopt(await API.saveFormat({ name: "New type", sort: PB.state.formats.length + 1 })); drawFormats(); } catch (err) { toast(errorText(err)); } };
  page.querySelector("#add-type").onclick = async () => { try { adopt(await API.saveType({ name: "Scrimmage", category: "other", color: "#8a4b2f", defaultMinutes: 60 })); drawTypes(); } catch (err) { toast(errorText(err)); } };
  page.querySelector("#add-blackout").onclick = async () => { const day = ymd(new Date()); try { adopt(await API.saveBlackout({ start: day, end: day, venueId: "", reason: "Closed" })); drawBlackouts(); } catch (err) { toast(errorText(err)); } };
  page.querySelector("#backup").onclick = downloadBackup;
  page.querySelector("#restore").onchange = restoreBackup;
  page.querySelector("#load-sample").onclick = async () => { if (!await confirmBox("Load the sample club?", "This replaces teams, places, and bookings.", "Replace")) return; adopt(await API.reset("sample")); toast("Sample club loaded."); PB.state.calendar?.refetchEvents(); renderSettings(); };
  page.querySelector("#load-blank").onclick = async () => { if (!await confirmBox("Clear the club?", "Teams, places, and bookings will be removed. Settings stay.", "Clear")) return; adopt(await API.reset("blank")); toast("Club cleared."); PB.state.calendar?.refetchEvents(); renderSettings(); };
  page.querySelector("#exit-app").onclick = async () => { if (!await confirmBox("Exit Pitchboard?", "The page will stop responding because the program closes.", "Exit")) return; try { await API.shutdown(); } catch { /* process exit drops the connection */ } toast("Pitchboard has closed. You can close this tab."); };
}

async function saveSettingsForm(event) {
  event.preventDefault();
  const data = new FormData(event.target);
  const on = (name) => data.get(name) === "on";
  const next = {
    ...PB.state.settings,
    clubName: data.get("clubName"),
    zoneName: data.get("zoneName") || "Aurora",
    seasonName: data.get("seasonName"),
    seasonStart: data.get("seasonStart"),
    seasonEnd: data.get("seasonEnd"),
    timezoneLabel: data.get("timezoneLabel"),
    weekStartsOn: Number(data.get("weekStartsOn")),
    timeFormat: data.get("timeFormat"),
    defaultView: data.get("defaultView"),
    dayStartHour: Number(data.get("dayStartHour")),
    dayEndHour: Number(data.get("dayEndHour")),
    slotMinutes: Number(data.get("slotMinutes")),
    showWeekends: on("showWeekends"),
    priorityCutoffAge: Number(data.get("priorityCutoffAge")),
    earlyWindowStart: data.get("earlyWindowStart"),
    earlyWindowEnd: data.get("earlyWindowEnd"),
    priorityMode: data.get("priorityMode"),
    priorityDays: [...event.target.querySelectorAll("[name=priorityDay]:checked")].map((box) => Number(box.value)),
    priorityCategories: [...event.target.querySelectorAll("[name=priorityCat]:checked")].map((box) => box.value),
    olderTeamsAvoidEarly: on("olderTeamsAvoidEarly"),
    legendEnabled: on("legendEnabled"),
    theme: readTheme(event.target),
    blockPitchConflicts: on("blockPitchConflicts"),
    blockTeamConflicts: on("blockTeamConflicts"),
    warnCoachConflicts: on("warnCoachConflicts"),
    blockBlackouts: on("blockBlackouts"),
  };
  try {
    const saved = await API.saveSettings(next);
    PB.state.settings = saved.settings;
    applyTheme(saved.settings.theme);
    applyCalendarOptions();
    renderLegend();
    toast("Settings saved.");
    renderSettings();
  } catch (err) { toast(errorText(err)); }
}

function themeField(label, name, value) {
  return `<label class="field"><span>${esc(label)}</span><input name="${name}" type="color" value="${esc(value || "#171717")}" /></label>`;
}

function readTheme(root) {
  const value = (name) => root.querySelector(`[name="${name}"]`)?.value || "";
  return {
    header: value("header"), headerInk: value("headerInk"), rail: value("rail"), accent: value("accent"), accentInk: value("accentInk"),
    paper: value("paper"), card: value("card"), ink: value("ink"), muted: value("muted"), line: value("line"),
    displayFont: value("displayFont"), bodyFont: value("bodyFont"),
  };
}

function teamMeta(team) {
  const group = ageById(team.ageGroupId);
  const bits = [];
  if (group?.name) bits.push(group.name);
  if (group?.program) bits.push(group.program);
  if (group?.windowStart && group?.windowEnd) bits.push(`${clockLabel(group.windowStart, PB.state.settings?.timeFormat)}–${clockLabel(group.windowEnd, PB.state.settings?.timeFormat)}`);
  else if (isPriorityTeam(team)) bits.push("Early");
  return bits.join(" · ");
}

function drawAges() {
  const host = document.getElementById("age-editor");
  if (!host) return;
  const formats = PB.state.formats || [];
  host.innerHTML = PB.state.ageGroups.map((group) => {
    const count = PB.state.teams.filter((team) => team.ageGroupId === group.id).length;
    return `<div class="age-card" data-age="${esc(group.id)}">
      <div class="row">
        <label class="field"><span>Name</span><input data-k="name" value="${esc(group.name)}" /></label>
        <label class="field"><span>Program</span><input data-k="program" value="${esc(group.program || "")}" placeholder="Academy, Development, City Rec" /></label>
        <label class="field"><span>Teams</span><input value="${count}" disabled /></label>
      </div>
      <div class="row">
        <label class="field"><span>Youngest</span><input data-k="minAge" type="number" value="${group.minAge}" /></label>
        <label class="field"><span>Oldest</span><input data-k="maxAge" type="number" value="${group.maxAge}" /></label>
        <label class="field"><span>Practice minutes</span><input data-k="defaultPracticeMinutes" type="number" value="${group.defaultPracticeMinutes}" /></label>
        <label class="field"><span>Game minutes</span><input data-k="defaultGameMinutes" type="number" value="${group.defaultGameMinutes || 0}" /></label>
      </div>
      <div class="row">
        <label class="field"><span>Hours from</span><input data-k="windowStart" type="time" value="${esc(group.windowStart || "")}" /></label>
        <label class="field"><span>Hours until</span><input data-k="windowEnd" type="time" value="${esc(group.windowEnd || "")}" /></label>
        <label class="field"><span>Sessions a week</span><input data-k="sessionsPerWeek" type="number" min="0" max="7" value="${group.sessionsPerWeek || 0}" /></label>
        <label class="field"><span>Early hours</span><select data-k="priority"><option value="auto" ${group.priority !== "always" && group.priority !== "never" ? "selected" : ""}>Use the age cutoff</option><option value="always" ${group.priority === "always" ? "selected" : ""}>Always</option><option value="never" ${group.priority === "never" ? "selected" : ""}>Never</option></select></label>
      </div>
      <div class="row">
        <label class="field"><span>Pitch size</span><select data-k="preferredFormatId"><option value="">Any</option>${formats.map((format) => `<option value="${esc(format.id)}" ${format.id === group.preferredFormatId ? "selected" : ""}>${esc(format.name)}</option>`).join("")}</select></label>
        <div class="field"><span>&nbsp;</span><div class="row"><button type="button" data-save>Save</button><button type="button" class="btn-ghost" data-del>Delete</button></div></div>
      </div>
    </div>`;
  }).join("");
  host.querySelectorAll("[data-age]").forEach((row) => {
    row.querySelector("[data-save]").onclick = async () => {
      const group = { ...ageById(row.dataset.age) };
      row.querySelectorAll("[data-k]").forEach((input) => { group[input.dataset.k] = input.type === "number" ? Number(input.value) : input.value; });
      try { adopt(await API.saveAge(group)); toast("Age group saved."); } catch (err) { toast(errorText(err)); }
    };
    row.querySelector("[data-del]").onclick = async () => {
      try { adopt(await API.deleteAge(row.dataset.age)); drawAges(); } catch (err) { toast(errorText(err)); }
    };
  });
}

function drawExports() {
  const host = document.getElementById("export-editor");
  if (!host) return;
  const systems = [["ramp", "RAMP InterActive"], ["teamlinkt", "TeamLinkt"], ["teamsnap", "TeamSnap"], ["gamechanger", "GameChanger"]];
  const cats = ["practice", "camp", "game", "other"];
  const saved = PB.state.exportProfiles || [];
  host.innerHTML = `<p><strong>Built in:</strong> RAMP practices (Team_ID, Event_Type, Start_Date, Start_Time, End_Time, Location_ID, Notes), TeamLinkt events, the TeamSnap schedule template, and a GameChanger organization file.</p>` + (saved.map((profile) => `<div class="age-card" data-id="${esc(profile.id)}">
      <div class="row">
        <label class="field"><span>Name</span><input data-k="name" value="${esc(profile.name)}" /></label>
        <label class="field"><span>System</span><select data-k="system">${systems.map(([id, label]) => `<option value="${id}" ${profile.system === id ? "selected" : ""}>${label}</option>`).join("")}</select></label>
      </div>
      <div class="field"><span>Include</span><div class="checks">${cats.map((cat) => `<label><input type="checkbox" data-cat="${cat}" ${(profile.categories || []).includes(cat) ? "checked" : ""}/> ${cat}</label>`).join("")}</div></div>
      <div class="row"><button type="button" data-save>Save</button><button type="button" class="btn-ghost" data-del>Delete</button></div>
    </div>`).join("") || `<p class="muted">No saved exports yet. The Export button already offers each system.</p>`);
  host.querySelectorAll("[data-id]").forEach((row) => {
    row.querySelector("[data-save]").onclick = async () => {
      const current = (PB.state.exportProfiles || []).find((item) => item.id === row.dataset.id) || {};
      const body = { ...current, name: row.querySelector("[data-k=name]").value, system: row.querySelector("[data-k=system]").value, categories: [...row.querySelectorAll("[data-cat]:checked")].map((box) => box.dataset.cat) };
      try { adopt(await API.saveExportProfile(body)); toast("Export saved."); } catch (err) { toast(errorText(err)); }
    };
    row.querySelector("[data-del]").onclick = async () => {
      try { adopt(await API.deleteExportProfile(row.dataset.id)); drawExports(); } catch (err) { toast(errorText(err)); }
    };
  });
}

function drawSurfaces() {
  const host = document.getElementById("surface-editor");
  if (!host) return;
  host.innerHTML = PB.state.surfaces.map((surface) => `<div class="row" data-id="${esc(surface.id)}"><input data-k="name" value="${esc(surface.name)}" /><select data-k="environment"><option value="indoor" ${surface.environment === "indoor" ? "selected" : ""}>Indoor</option><option value="outdoor" ${surface.environment === "outdoor" ? "selected" : ""}>Outdoor</option><option value="either" ${surface.environment === "either" ? "selected" : ""}>Either</option></select><button type="button" data-save>Save</button><button type="button" data-del>Delete</button></div>`).join("");
  wireRows(host, (id) => byId(PB.state.surfaces, id), API.saveSurface, API.deleteSurface, drawSurfaces);
}

function drawFormats() {
  const host = document.getElementById("format-editor");
  if (!host) return;
  host.innerHTML = PB.state.formats.map((format) => `<div class="row" data-id="${esc(format.id)}"><input data-k="name" value="${esc(format.name)}" /><button type="button" data-save>Save</button><button type="button" data-del>Delete</button></div>`).join("");
  wireRows(host, (id) => byId(PB.state.formats, id), API.saveFormat, API.deleteFormat, drawFormats);
}

function drawTypes() {
  const host = document.getElementById("type-editor");
  if (!host) return;
  host.innerHTML = PB.state.eventTypes.map((type) => `<div class="row" data-id="${esc(type.id)}"><input data-k="name" value="${esc(type.name)}" /><select data-k="category">${["practice", "camp", "game", "other"].map((cat) => `<option ${type.category === cat ? "selected" : ""}>${cat}</option>`).join("")}</select><input data-k="defaultMinutes" type="number" value="${type.defaultMinutes}" /><input data-k="color" type="color" value="${esc(type.color || "#2f6a4a")}" /><button type="button" data-save>Save</button><button type="button" data-del>Delete</button></div>`).join("");
  wireRows(host, (id) => typeById(id), API.saveType, API.deleteType, drawTypes);
}

function drawBlackouts() {
  const host = document.getElementById("blackout-editor");
  if (!host) return;
  host.innerHTML = PB.state.blackouts.map((item) => `<div class="row" data-id="${esc(item.id)}"><input data-k="start" type="date" value="${esc(item.start)}" /><input data-k="end" type="date" value="${esc(item.end)}" /><select data-k="venueId"><option value="">All locations</option>${PB.state.venues.map((venue) => `<option value="${esc(venue.id)}" ${venue.id === item.venueId ? "selected" : ""}>${esc(venue.name)}</option>`).join("")}</select><input data-k="reason" value="${esc(item.reason)}" /><button type="button" data-save>Save</button><button type="button" data-del>Delete</button></div>`).join("") || `<p class="muted">No closures.</p>`;
  wireRows(host, (id) => PB.state.blackouts.find((item) => item.id === id), API.saveBlackout, API.deleteBlackout, drawBlackouts);
}

function wireRows(host, find, save, remove, redraw) {
  host.querySelectorAll("[data-id]").forEach((row) => {
    row.querySelector("[data-save]").onclick = async () => {
      const body = { ...find(row.dataset.id) };
      row.querySelectorAll("[data-k]").forEach((input) => { body[input.dataset.k] = input.type === "number" ? Number(input.value) : input.value; });
      try { adopt(await save(body)); toast("Saved."); } catch (err) { toast(errorText(err)); }
    };
    row.querySelector("[data-del]").onclick = async () => {
      try { adopt(await remove(row.dataset.id)); redraw(); } catch (err) { toast(errorText(err)); }
    };
  });
}

function zoneOptions(current) {
  const zones = ["Aurora", "SUSC", "Hollandia", "Lakewood", "Eastside"];
  if (current && !zones.includes(current)) zones.push(current);
  return zones.map((zone) => `<option ${(!current && zone === "Aurora") || current === zone ? "selected" : ""}>${esc(zone)}</option>`).join("");
}

function openImport() {
  const root = document.getElementById("modal-root");
  const week = PB.state.rangeStart ? ymd(PB.state.rangeStart) : (PB.state.settings?.seasonStart || ymd(new Date()));
  root.innerHTML = `<div class="modal-back"><form class="modal wide" id="import-form">
    <h2>Import a spreadsheet</h2>
    <p class="muted">CSV or Excel. A Saskatoon Youth Soccer master timetable can include practice times, games, and matches held by SUSC, Hollandia, Lakewood, or Eastside that this club might swap.</p>
    <label class="field"><span>File</span><input name="file" type="file" accept=".csv,.xlsx,text/csv,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" required /></label>
    <div class="row">
      <label class="field"><span>This file is</span><select name="source">
        <option>Saskatoon Youth Soccer</option>
        <option>This club</option>
        <option>Another spreadsheet</option>
      </select></label>
      <label class="field"><span>Week of</span><input name="weekOf" type="date" value="${esc(week)}" /></label>
    </div>
    <p class="muted">Use Week of when a column says Monday instead of a date. The first row should name the columns. Date, start, end, field, type, zone, home, and away are recognized.</p>
    <div id="import-map" class="map-grid"></div>
    <div id="import-preview"></div>
    <div class="row">
      <button class="btn" type="submit" name="step" value="read">Read the file</button>
      <button class="btn" type="submit" name="step" value="add" id="import-add" disabled>Add to the calendar</button>
      <button class="btn-ghost" type="button" id="import-close">Close</button>
    </div>
  </form></div>`;
  const form = root.querySelector("#import-form");
  let mapping = {};
  const close = () => { root.innerHTML = ""; };
  root.querySelector(".modal-back").onclick = (event) => { if (event.target === event.currentTarget) close(); };
  root.querySelector("#import-close").onclick = close;
  form.onsubmit = async (event) => {
    event.preventDefault();
    const file = form.querySelector("[name=file]").files[0];
    if (!file) { toast("Choose a CSV or Excel file."); return; }
    const body = new FormData();
    body.append("file", file);
    body.append("source", form.querySelector("[name=source]").value);
    body.append("weekOf", form.querySelector("[name=weekOf]").value);
    body.append("mapping", JSON.stringify(mapping));
    const adding = event.submitter?.value === "add";
    try {
      const res = await API.upload(adding ? "/api/import" : "/api/import/preview", body);
      if (!adding) {
        mapping = res.mapping || {};
        drawImportMap(res, (next) => { mapping = next; form.requestSubmit(form.querySelector("[name=step][value=read]")); });
        drawImportPreview(res);
        form.querySelector("#import-add").disabled = false;
        return;
      }
      const added = (res.bookings || 0) + (res.open || 0) + (res.swaps || 0);
      const word = (n, one, many) => `${n} ${n === 1 ? one : many}`;
      toast(added ? `Added ${word(res.bookings || 0, "booking", "bookings")}, ${word(res.open || 0, "open time", "open times")}, and ${word(res.swaps || 0, "swap", "swaps")}.` : "Nothing new was added.");
      close();
      PB.state.calendar?.refetchEvents();
      renderInspector();
    } catch (err) { toast(errorText(err)); }
  };
}

function drawImportMap(res, onChange) {
  const host = document.getElementById("import-map");
  if (!host) return;
  host.innerHTML = (res.headers || []).map((header) => `<label class="field"><span>${esc(header)}</span><select data-col="${esc(header)}">${(res.fields || []).map((field) => `<option value="${esc(field.id)}" ${(res.mapping || {})[header] === field.id ? "selected" : ""}>${esc(field.label)}</option>`).join("")}</select></label>`).join("");
  host.querySelectorAll("select").forEach((select) => {
    select.onchange = () => {
      const next = { ...(res.mapping || {}) };
      host.querySelectorAll("select").forEach((item) => { next[item.dataset.col] = item.value; });
      onChange(next);
    };
  });
}

function drawImportPreview(res) {
  const host = document.getElementById("import-preview");
  if (!host) return;
  const rows = res.rows || [];
  const head = `<p><strong>${res.bookings || 0} bookings · ${res.open || 0} open times · ${res.swaps || 0} swaps · ${res.skipped || 0} skipped</strong></p>`;
  const lines = rows.slice(0, 8).map((row) => `<div>${esc(row.summary || row.kind)}</div>`).join("");
  const more = rows.length > 8 ? `<p class="muted">${rows.length - 8} more rows</p>` : "";
  host.innerHTML = head + `<div class="preview-list">${lines}</div>` + more;
}

function openExport() {
  const start = PB.state.rangeStart ? ymd(PB.state.rangeStart) : (PB.state.settings.seasonStart || ymd(new Date()));
  const end = PB.state.rangeEnd ? ymd(new Date(PB.state.rangeEnd.getTime() - 86400000)) : (PB.state.settings.seasonEnd || start);
  const root = document.getElementById("modal-root");
  const profiles = PB.state.exportProfiles || [];
  root.innerHTML = `<div class="modal-back"><form class="modal wide" id="export-form"><h2>Export</h2>
    <div class="row"><label class="field"><span>From</span><input name="start" type="date" value="${esc(start)}" /></label><label class="field"><span>Through</span><input name="end" type="date" value="${esc(end)}" /></label></div>
    <label><input type="checkbox" name="cancelled" /> Include cancelled</label>
    <h3>Files</h3>
    <div class="row" style="margin-top:8px">
      <button class="btn" name="kind" value="csv" type="submit">CSV</button>
      <button class="btn" name="kind" value="xlsx" type="submit">Excel</button>
      <button class="btn" name="kind" value="pdf" type="submit">PDF</button>
      <button class="btn" name="kind" value="ics" type="submit">Calendar</button>
    </div>
    <h3>Online systems</h3>
    <p class="muted">RAMP is a practice file: Team_ID, Event_Type (Practice), Start_Date, Start_Time, End_Time, Location_ID, Notes. Times look like 07:15 PM. TeamLinkt, TeamSnap, and GameChanger use the columns those sites import.</p>
    <div class="row">
      <button class="btn" name="kind" value="system:ramp" type="submit">RAMP</button>
      <button class="btn" name="kind" value="system:teamlinkt" type="submit">TeamLinkt</button>
      <button class="btn" name="kind" value="system:teamsnap" type="submit">TeamSnap</button>
      <button class="btn" name="kind" value="system:gamechanger" type="submit">GameChanger</button>
    </div>
    ${profiles.length ? `<label class="field"><span>Saved export</span><select name="profile"><option value="">Choose a saved export</option>${profiles.map((profile) => `<option value="${esc(profile.id)}">${esc(profile.name)}</option>`).join("")}</select></label><button class="btn" name="kind" value="profile" type="submit">Download saved export</button>` : ""}
    <button type="button" class="btn-ghost" id="export-close">Close</button>
  </form></div>`;
  root.querySelector("#export-close").onclick = () => { root.innerHTML = ""; };
  root.querySelector("#export-form").onsubmit = async (event) => {
    event.preventDefault();
    const data = new FormData(event.target);
    const kind = event.submitter.value;
    const params = new URLSearchParams({ start: data.get("start") + "T00:00:00", end: data.get("end") + "T23:59:59" });
    if (data.get("cancelled")) params.set("cancelled", "1");
    let path = "/api/export/" + kind;
    if (kind.startsWith("system:")) path = "/api/export/system/" + kind.slice(7);
    if (kind === "profile") {
      const profile = (PB.state.exportProfiles || []).find((item) => item.id === data.get("profile"));
      if (!profile) { toast("Choose a saved export."); return; }
      path = "/api/export/system/" + encodeURIComponent(profile.system);
      params.set("profile", profile.id);
    }
    try {
      const file = await API.file(path + "?" + params.toString());
      downloadBlob(file.blob, file.filename);
      if (file.notice) toast(file.notice);
      root.innerHTML = "";
    } catch (err) { toast(errorText(err)); }
  };
}

async function downloadBackup() {
  try {
    const file = await API.file("/api/backup");
    downloadBlob(file.blob, file.filename);
  } catch (err) { toast(errorText(err)); }
}

async function restoreBackup(event) {
  const file = event.target.files?.[0];
  if (!file) return;
  if (!await confirmBox("Restore this backup?", "The schedule on this computer will be replaced.", "Restore")) return;
  try {
    const text = await file.text();
    adopt(await API.restore(JSON.parse(text)));
    toast("Backup restored.");
    PB.state.calendar?.refetchEvents();
    renderSettings();
  } catch (err) { toast(errorText(err)); }
}

async function refreshSummary() {
  if (!PB.state.mini) return;
  const start = new Date(PB.state.mini.getFullYear(), PB.state.mini.getMonth(), 1);
  const end = new Date(PB.state.mini.getFullYear(), PB.state.mini.getMonth() + 1, 1);
  try {
    const data = await API.summary(toLocalISO(start), toLocalISO(end));
    PB.state.summary = data.days || [];
    renderMini();
  } catch { /* the calendar still works without dots */ }
}
