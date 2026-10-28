function adopt(data) {
  if (!data) return;
  PB.state.settings = data.settings || PB.state.settings;
  PB.state.ageGroups = data.ageGroups || [];
  PB.state.teams = data.teams || [];
  PB.state.venues = data.venues || [];
  PB.state.pitches = data.pitches || [];
  PB.state.surfaces = data.surfaces || [];
  PB.state.formats = data.formats || [];
  PB.state.eventTypes = data.eventTypes || [];
  PB.state.blackouts = data.blackouts || [];
  if (data.dataDir) PB.state.dataDir = data.dataDir;
  if (data.version) PB.state.version = data.version;
  PB.state.exportProfiles = data.exportProfiles || [];
  PB.state.initialized = !!data.initialized || !!data.settings?.initialized;
  applyTheme(PB.state.settings?.theme);
}

function openApp() {
  document.getElementById("setup").hidden = true;
  document.getElementById("shell").hidden = false;
  document.title = `Pitchboard — ${PB.state.settings.clubName}`;
  applyTheme(PB.state.settings?.theme);
  if (!PB.state.calendar) initCalendar();
  else applyCalendarOptions();
  renderLegend();
  setView("calendar");
  renderStatus();
}

function setView(name) {
  PB.state.view = name;
  document.querySelectorAll(".rail-btn").forEach((button) => button.classList.toggle("is-active", button.dataset.view === name));
  const calendar = document.getElementById("cal-wrap");
  const page = document.getElementById("page");
  const workspace = document.getElementById("workspace");
  const sidebar = document.getElementById("sidebar");
  const isCalendar = name === "calendar";
  calendar.hidden = !isCalendar;
  page.hidden = isCalendar;
  workspace.classList.toggle("wide", !isCalendar);
  if (isCalendar) {
    ensureSidebar();
    renderTeamList();
    renderFilters();
    renderInspector();
    PB.state.calendar?.updateSize();
  } else {
    sidebar.innerHTML = "";
    sidebar.dataset.built = "";
    document.getElementById("inspector").innerHTML = "";
    if (name === "plan") renderPlan();
    if (name === "teams") renderTeams();
    if (name === "places") renderPlaces();
    if (name === "settings") renderSettings();
  }
  renderTop();
  renderLegend();
}

function startDraft(start, end, allDay, extra) {
  PB.state.formMode = "create";
  PB.state.opening = null;
  PB.state.selectedId = null;
  PB.state.dirty = false;
  PB.state.form = blankEvent(start, end || new Date(start.getTime() + 60 * 60000), !!allDay);
  if (extra) Object.assign(PB.state.form, extra);
  if (PB.state.view !== "calendar") setView("calendar");
  renderInspector();
}

function editEvent(event) {
  if (event?.opening) {
    PB.state.formMode = "opening";
    PB.state.selectedId = event.id;
    PB.state.opening = event;
    PB.state.form = null;
    PB.state.dirty = false;
    if (PB.state.view !== "calendar") setView("calendar");
    renderInspector();
    return;
  }
  PB.state.opening = null;
  PB.state.formMode = "edit";
  PB.state.selectedId = event.id;
  PB.state.form = editable(event);
  PB.state.dirty = false;
  if (PB.state.view !== "calendar") setView("calendar");
  renderInspector();
  PB.state.calendar?.refetchEvents();
}

function pushUndo(entry) {
  PB.state.undo.push(entry);
  if (PB.state.undo.length > 30) PB.state.undo.shift();
}

async function undo() {
  const entry = PB.state.undo.pop();
  if (!entry) { toast("Nothing to undo."); return; }
  try {
    if (entry.type === "create") {
      for (const id of entry.ids || []) await API.remove(id, "single");
    } else if (entry.type === "delete") {
      for (const event of entry.events || []) await API.create(editable(event));
    } else if (entry.type === "update" && entry.before) {
      await API.update(entry.before.id, entry.before);
    }
    PB.state.form = null;
    PB.state.calendar?.refetchEvents();
    renderInspector();
    toast("Undone.");
  } catch (err) {
    toast(errorText(err));
  }
}

function bindChrome() {
  document.querySelectorAll(".rail-btn").forEach((button) => {
    button.onclick = () => setView(button.dataset.view);
  });
  document.addEventListener("keydown", (event) => {
    const typing = ["INPUT", "TEXTAREA", "SELECT"].includes(event.target.tagName);
    if (event.key === "Escape") {
      document.getElementById("modal-root").innerHTML = "";
      if (!typing && PB.state.form) {
        PB.state.form = null;
        PB.state.selectedId = null;
        renderInspector();
      }
    }
    if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "z" && !event.shiftKey) {
      event.preventDefault();
      undo();
    }
    if (!typing && (event.key === "Delete" || event.key === "Backspace") && PB.state.form?.id) {
      event.preventDefault();
      deleteCurrent();
    }
    if (!typing && event.key.toLowerCase() === "t") PB.state.calendar?.today();
  });
  window.addEventListener("beforeunload", (event) => {
    if (PB.state.dirty) {
      event.preventDefault();
      event.returnValue = "";
    }
  });
}

async function boot() {
  try {
    adopt(await API.bootstrap());
  } catch (err) {
    document.body.innerHTML = `<main style="padding:40px"><h1>Pitchboard could not start</h1><p>${esc(err.message)}</p></main>`;
    return;
  }
  bindChrome();
  if (!PB.state.initialized) renderSetup();
  else openApp();
}

boot();
