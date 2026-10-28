# Pitchboard

Pitchboard is a portable soccer-club scheduler for one Windows computer. It is set up first for [Aurora Soccer Club](https://aurorasoccer.ca/), and the colours, fonts, age groups, and exports can be changed for any other club. It keeps practices, camps, and games on an Outlook-style calendar, with drag and drop, indoor and outdoor pitches, and earlier hours reserved first for players under 11.

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

## Look

The default theme matches Aurora’s site: black header `#171717`, green `#29683b`, Oswald for headings, and Merriweather for text. Settings → Look changes every colour and the two fonts. **Use Aurora colours** puts the original palette back.

## Age groups

Each age group can hold as many teams as you need. Groups are fully editable: name, ages, program (Development, City Rec, Alliance, Academy, or any label you type), practice and game length, sessions a week, and the pitch size.

A group can also set its own hours. Leave the hours blank to use the club early window. Aurora Advanced Academy in the sample uses 4:30–8:30 PM, so the planner offers those teams the evening instead of the morning. Early hours can be forced on or off for a whole group, separate from the under-11 cutoff.

## Import a timetable

Import reads a CSV or Excel file. Use it for a master timetable from Saskatoon Youth Soccer: times that are free for practice or games, this club’s own rows, and matches held by the other zones (SUSC, Hollandia, Lakewood, and Eastside) that a team might swap into.

The first row names the columns. Date, start, end, field, type, zone, home, and away are recognized. Each column has a dropdown if a header needs to be corrected. A weekday such as Monday uses the week you pick in the import window. Rows for this club become bookings when the place matches a pitch. Other zones become dashed swap blocks. A row with no team becomes an open time. Click one, choose the team and place, and book it.

Our zone defaults to Aurora. Settings → Club can switch it to SUSC, Hollandia, Lakewood, or Eastside.

## Booking

A new booking asks for four choices: team, what it is, place, and when. The place dropdown is the location and pitch together, and the surface and size come from that pitch. Title, status, notes, all day, and repeat sit under More. The calendar filters are three dropdowns: place, surface, and kind.

## Exports to other systems

Besides CSV, Excel, PDF, and iCalendar, Export can build a file for:

- **RAMP InterActive** practices: `Team_ID`, `Event_Type` (always `Practice`), `Start_Date` (`YYYY-MM-DD`), `Start_Time` and `End_Time` (`07:15 PM`), `Location_ID`, `Notes`. Team and location ids are the numeric ids from RAMP. Enter them on each team and each place. A pitch id overrides the place. Rows missing an id are left out, and Pitchboard says how many.
- **TeamLinkt**: date, time, type, division, home team, away team, location, duration, notes. Use the names already stored in TeamLinkt.
- **TeamSnap**: the team schedule template, with dates as `MM/DD/YYYY`. Practices omit the opponent.
- **GameChanger**: `date`, `time`, `home`, `away`, `location`, `duration` in minutes.

Settings can save a named export that keeps one of those layouts and limits it to practices, camps, games, or other types.

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

The first screen can load a sample week for **Aurora Soccer Club**: several teams in the same age groups, an Academy group with evening hours, and bookings across the SaskTel Sports Centre, Henk Ruys Soccer Centre, and Forest Park. One U8 practice is intentionally late so the early-hour warning is visible. The sample RAMP ids are placeholders to replace with the real numbers from RAMP.
