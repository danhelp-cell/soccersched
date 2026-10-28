const API = {
  async req(method, path, body) {
    const res = await fetch(path, {
      method,
      headers: {
        "X-Pitchboard": "1",
        ...(body ? { "Content-Type": "application/json" } : {}),
      },
      body: body ? JSON.stringify(body) : undefined,
    });
    const text = await res.text();
    let data = null;
    if (text) {
      try { data = JSON.parse(text); } catch { data = { error: text }; }
    }
    if (!res.ok) {
      const err = new Error(data?.error || res.statusText);
      err.status = res.status;
      err.data = data;
      throw err;
    }
    return data;
  },
  bootstrap() { return this.req("GET", "/api/bootstrap"); },
  setup(body) { return this.req("POST", "/api/setup", body); },
  reset(mode) { return this.req("POST", "/api/reset", { mode }); },
  saveSettings(settings) { return this.req("PUT", "/api/settings", settings); },
  events(start, end) {
    const q = new URLSearchParams({ start: start.toISOString(), end: end.toISOString(), cancelled: "1" });
    return this.req("GET", "/api/events?" + q.toString());
  },
  summary(start, end) {
    const q = new URLSearchParams({ start, end, cancelled: "1" });
    return this.req("GET", "/api/summary?" + q.toString());
  },
  create(body) { return this.req("POST", "/api/events", body); },
  update(id, body) { return this.req("PUT", "/api/events/" + encodeURIComponent(id), body); },
  remove(id, scope) { return this.req("DELETE", "/api/events/" + encodeURIComponent(id) + (scope ? "?scope=" + scope : "")); },
  check(body) { return this.req("POST", "/api/events/check", body); },
  batch(events, acknowledgeLate) { return this.req("POST", "/api/events/batch", { events, acknowledgeLate }); },
  suggest(body) { return this.req("POST", "/api/schedule/suggest", body); },
  fixes(start, end) { return this.req("POST", "/api/schedule/fixes", { start, end }); },
  earlier(event) { return this.req("POST", "/api/schedule/earlier", { event, limit: 4 }); },
  applyFix(proposal) { return this.req("POST", "/api/schedule/apply", proposal); },
  copyWeek(fromStart, fromEnd, weeks) { return this.req("POST", "/api/schedule/copy", { fromStart, fromEnd, weeks }); },
  saveTeam(team) { return team.id ? this.req("PUT", "/api/teams/" + team.id, team) : this.req("POST", "/api/teams", team); },
  deleteTeam(id) { return this.req("DELETE", "/api/teams/" + encodeURIComponent(id)); },
  saveVenue(venue) { return venue.id ? this.req("PUT", "/api/venues/" + venue.id, venue) : this.req("POST", "/api/venues", venue); },
  deleteVenue(id) { return this.req("DELETE", "/api/venues/" + encodeURIComponent(id)); },
  savePitch(pitch) { return pitch.id ? this.req("PUT", "/api/pitches/" + pitch.id, pitch) : this.req("POST", "/api/pitches", pitch); },
  deletePitch(id) { return this.req("DELETE", "/api/pitches/" + encodeURIComponent(id)); },
  saveAge(group) { return group.id ? this.req("PUT", "/api/age-groups/" + group.id, group) : this.req("POST", "/api/age-groups", group); },
  deleteAge(id) { return this.req("DELETE", "/api/age-groups/" + encodeURIComponent(id)); },
  saveSurface(row) { return row.id ? this.req("PUT", "/api/surfaces/" + row.id, row) : this.req("POST", "/api/surfaces", row); },
  deleteSurface(id) { return this.req("DELETE", "/api/surfaces/" + encodeURIComponent(id)); },
  saveFormat(row) { return row.id ? this.req("PUT", "/api/formats/" + row.id, row) : this.req("POST", "/api/formats", row); },
  deleteFormat(id) { return this.req("DELETE", "/api/formats/" + encodeURIComponent(id)); },
  saveType(row) { return row.id ? this.req("PUT", "/api/event-types/" + row.id, row) : this.req("POST", "/api/event-types", row); },
  deleteType(id) { return this.req("DELETE", "/api/event-types/" + encodeURIComponent(id)); },
  saveBlackout(row) { return row.id ? this.req("PUT", "/api/blackouts/" + row.id, row) : this.req("POST", "/api/blackouts", row); },
  deleteBlackout(id) { return this.req("DELETE", "/api/blackouts/" + encodeURIComponent(id)); },
  async file(path) {
    const res = await fetch(path, { headers: { "X-Pitchboard": "1" } });
    if (!res.ok) {
      const text = await res.text();
      throw new Error(text || "Download failed");
    }
    const blob = await res.blob();
    const match = /filename="([^"]+)"/.exec(res.headers.get("Content-Disposition") || "");
    return { blob, filename: match ? match[1] : "pitchboard" };
  },
  restore(data) { return this.req("POST", "/api/backup", data); },
  shutdown() { return this.req("POST", "/api/shutdown", {}); },
};
