# Pitchboard

Pitchboard is a portable soccer-club scheduler for one Windows computer. It keeps practices, camps, and games on an Outlook-style calendar, with drag and drop, indoor and outdoor pitches, and earlier hours reserved first for players under 11.

No account and no network are required. The program listens only on this computer.

## Run it

1. Build `release\Pitchboard.exe` with `scripts\build-windows.bat` (Go 1.22 or newer), or use a copy that is already built.
2. Double-click `Pitchboard.exe`, or `Start Pitchboard.bat`.
3. The club opens in your browser at an address like `http://127.0.0.1:47821/`.
4. Closing the browser leaves the program running. Open the exe again to bring the calendar back. Use **Exit Pitchboard** in Settings when you want it to stop.

The schedule is stored in a `PitchboardData` folder next to the exe (`pitchboard.json`). Copy the exe and that folder to move the club to another computer.

From source on any machine:

```bash
go run ./cmd/pitchboard
```

`--no-browser` skips opening a window. `--data` chooses the folder. `--port` chooses the port.

## What you can schedule

- Practices, camps, games, and any extra type you add
- Home and away games
- Indoor turf, indoor hardcourt, gym, outdoor grass, outdoor turf, plus surfaces you add
- Pitch size (3v3 through 11v11, gym floor, or your own)
- Weekly or daily repeats
- Closure dates for a location or the whole club

Drag a team from the left onto a time, or drag across the grid to create a booking. Drag a booking to move it, and drag its edge to change the length.

## Earlier hours for younger players

Age groups whose oldest players are **under the cutoff** (11 by default) are the early-priority teams. U10 qualifies. U12 does not. The early window defaults to 8:00–5:30, and both the age and the hours can be changed in Settings.

- The bright band on the calendar is that window.
- **Plan the week** places the youngest teams first, as early as a suitable pitch is free, then fills older teams afterward.
- A dashed outline marks a young team who starts after the window. Plan can move them, or swap them with an older team.
- Warn mode allows the late booking and marks it. Strict mode asks you to confirm.

A team can be forced into or out of that priority without changing its age.

## Export

Export the visible range, or any dates you choose:

- CSV (UTF-8 with a byte order mark so Excel on Windows opens it cleanly)
- Excel workbook (schedule, teams, and places)
- PDF agenda
- iCalendar (`.ics`) for Outlook or another calendar

Settings also downloads and restores a full JSON backup.

## Sample club

The first screen can load **Northfield Soccer Club**: a week of practices, a weekend skills camp, and games across a dome, a recreation centre (hardcourt and gym), a grass park, and an outdoor turf complex. One U8 practice is intentionally late so the early-hour warning is visible.
