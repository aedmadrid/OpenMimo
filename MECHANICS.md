# OpenMimo — Reverse-Engineered Mechanics & Format Specification

Source target: `MFW-FREE.EXE` (32-bit x86 Delphi PE, Mimosa Scheduling Software Free Edition v5.x) and accompanying configuration/data files (`Default.mpf`, `english.cfw`, `spanish.cfw`, `Data/*.mfw`, `Data/*.mxt`).

## 1. Free Edition Artificial Limits (Removed in OpenMimo)

| Mechanic | Free Limit in `MFW-FREE.EXE` | Pro / File-Format Limit | OpenMimo Behavior | Source Evidence |
|---|---|---|---|---|
| Total Codes (Events + Resources) | `80` (`0x50`) hard cap; warning shown when `remaining < 16` (`remaining % 5 == 0 \|\| remaining < 6`) | `8191` (`0x1FFF`) in `.mfw` header (`offset 0x06`) | **Unlimited** (no 80-code cap; tested with 8191+ codes) | `CODE 0x55da74–0x55db6e` (`movl $0x50, %edi` at `0x55da9f`; warning check at `0x55daf0`) |
| File Saves | Limited saves in tentative/free builds ("cannot save your file on disk unlimited times") | Unlimited | **Unlimited** | `CODE 0x587950–0x587a16` (XREF `0x5879c3` to string `0x587d34`) |
| Max Schedule Records (`XT`) | Same binary array limit | `32763` (`0x7FFB`) in `.mfw` header (`offset 0x08`) | **Unlimited** | `.mfw` header `offset 0x08` |
| Max Collection Links (`XC`) | Same binary array limit | `32763` (`0x7FFB`) in `.mfw` header (`offset 0x0A`) | **Unlimited** | `.mfw` header `offset 0x0A` |
| Max Weeks / Terms | `255` | `255` | `1..255` configurable | `TTabNotebook` DFM (`0x2ae5c0`, `WeekLabel: MAX=255`), `.mfw` header `offset 0x48` |
| Max Days per Week | `7` | `7` | `1..7` configurable | `TTabNotebook` DFM (`DayLabel: MAX=7`), `.mfw` header `offset 0x4A` |
| Max Time Periods per Day | `30` | `30` | `1..30` configurable | `TTabNotebook` DFM (`HourLabel: MAX=30`), `.mfw` header `offset 0x4C` |

## 2. Core Data Model (Exact Mimosa Parity)

1. **Time Structure**:
   - `maxWeeks` (1..255), `maxDays` (1..7, default 5), `maxLectures` / `maxPeriods` (1..30, default 10), `firstWeek` (default 1).
   - Named days (`Monday..Sunday` / `Lunes..Domingo`), named weeks (`A1`, `[1]..[255]`, each with optional `length` multiplier in weeks, default 1), and bell times (`08:00-08:45`, etc.).
2. **Events (Courses) & Resources (Components)**:
   - Up to 7 Event categories (default ES: `c:Grupos chicos`, `g:Grupos grandes`, `v:Voluntario`, `o:Obligatorio`, `e:Estudios externos`, `t:Trabajo en terreno`; EN: `b:Basic`, `m:Middle`, `a:Advanced`, `f:Free choice`, `o:Meetings`, `e:Exams`, `s:Special events`).
   - Up to 7 Resource categories (default ES: `C:Clases`, `P:Profesores`, `S:Salas`, `M:Materias`, `A:Alumnos`, `E:Equipos`, `O:Otros`; EN: `G:Groups`, `T:Teachers`, `R:Rooms`, `S:Subjects`, `P:Students`, `E:Equipment`, `O:Other`).
   - Each Resource category has:
     - `noTimetables` (conflict-ignored flag: e.g., Subjects=true so multiple classes can study Math simultaneously; Classes/Teachers/Rooms=false so conflicts are enforced!).
     - `noGaps` (whether gaps are ignored during optimization: default `0;0;1;1;0;1;1` from `sample_intake1.mxt`, meaning Groups, Teachers, and Students minimize gaps, while Rooms/Subjects/Equipment don't!).
     - `roomsCategory` (default `3` = Rooms/Salas).
     - `weeklyLimit` (optional upper limit warning per resource category in Weeks view).
     - `gapWeight` (1..10 weight during gap optimization, default `1` from `Default.mpf` `[Optimize]`).
   - Each Event has: `code`, `name`, `category`, `planned` (MAX), `color`, `locked`, `memo`, and a **Collection** of assigned Resource IDs.
   - Derived counters (always live-computed):
     - `allocated` (`SUM`): sum of weekly allocations across all weeks (`weeklyLectures * weekLength`).
     - `scheduled` (`DONE`): total scheduled timetable slots across all weeks.
     - `left`: `allocated - scheduled` (or `planned - allocated` in Weeks view).
     - `links`: count of linked resources (for events) or linked events (for resources).
3. **Bookings (Unavailable / Special Time Slots)**:
   - Types (`english.cfw` / `spanish.cfw`): `/////` (Blocked/Unavailable), `Free`/`Libre` or `Holiday`, `Meeting`/`Reunión`, `Other`/`Otros`.
   - Applies to both Weeks (`Weekbookings`) and Timetables (`Tablebookings`) for any Event or Resource.
   - A booking blocks scheduling in that slot for that entity and any event using that resource.
4. **Manual Room Assignments**:
   - Individual scheduled slots `(course, week, day, period)` can override/assign a specific Room resource, checked against room availability at `(week, day, period)`.
   - Supports **Imitate** (propagating rooms from already-assigned slots of the same course or same time on other weeks).

## 3. Binary `.mfw` & Text `.mxt` File Formats (Verified by Byte-Level Analysis)

### `.mfw` Binary Format (Mimosa 5.x)
Verified against `sample_intake1.mfw` (vs `sample_intake1.mxt`), `sample_tiny_step_1..4.mfw`, `sample_secondary_school_spanish.mfw`, and `sample_veterinary_school.mfw`:
- **Header (`0x000..0x11D`, 286 bytes)**:
  - `0x004` (`uint16`): version tag (`5128` / `5129`)
  - `0x006` (`uint16`): max codes (`8191`)
  - `0x01A` (`uint16`): week lecture record count
  - `0x01E` (`uint16`): table lecture record count
  - `0x022` (`uint16`): total schedule record count (`week + table + bookings`)
  - `0x02E` (`uint16`): total codes (`numCourses + numResources`)
  - `0x036` (`uint16`): course count; `0x03A` (`uint16`): resource count
  - `0x066..0x069` (`4 x uint8`): `firstWeek`, `maxWeeks`, `maxDays`, `maxHours`
- **Fixed Pascal ShortString Tables (`0x11E..0xDE8`)**:
  - `0x11E`: Days string (`$`-delimited, 60 bytes)
  - `0x15C`: 30 Timeperiod strings (12 bytes each)
  - `0x2C8`: Course categories (256 bytes); `0x3C8`: Resource categories (256 bytes); `0x4C8`: Booking names (256 bytes)
  - `0x5D3`: Week names (8 bytes each)
- **Tagged Chunk Stream (`0xDE5..EOF`)**:
  - Chunk `10`: `numCodes` × 16-byte Pascal ShortString code identifiers
  - Chunk `11`: `numCodes` × 14-byte metadata (`sum`, `done`, `flags`, `ttIdx`, `collIdx`, `max`, `links`) where `flags & 0x4000` = isCourse, and bits `7..13` (`0x0080 << (cat-1)`) encode category 1..7
  - Chunk `21`: `(totalRecords + 1) × uint16` owner code (`0x4000 | codeIdx` for courses, `codeIdx` for resources/bookings)
  - Chunk `22`: `(totalRecords + 1) × uint16` start slot:
    - Week allocation (`day1 == 0`): `(week1 << 8)`
    - Timetable slot (`day1 > 0`): `(week1 << 8) | (day1 << 5) | hour1`
  - Chunk `23`: `(totalRecords + 1) × uint16` end slot:
    - Week allocation: `(week2 << 8) | weeklyPeriods`
    - Timetable slot: `(week2 << 8) | (day2 << 5) | hour2`
  - Chunk `24`: Collections flat array (`0x4000 | courseIdx` followed by resource indices)
  - Tail String Table: `<uint16 id><uint16 len+1><uint8 len><bytes>` where `id & 0x4000` maps full names to 1-based code indices, and other IDs store report titles / customer / department.

### `.mxt` Text Exchange Format
Verified from `sample_intake1.mxt` and `MFW-FREE.EXE` (`0x59ba88`):
INI-style sections: `[Header]`, `[Time]`, `[Timeperiods]`, `[Weeks]`, `[Blocks]`, `[Courses]`, `[Components]`, `[Coursedata]`, `[Componentdata]`, `[Collections]`, `[Weeklectures]`, `[Weekbookings]`, `[Tablelectures]`, `[Tablebookings]`, `[Manualrooms]`, `[Coursedates]`, `[Tablecomments]`.

## 4. Disassembled Functions, Constants & Scheduling Rules (`MFW-FREE.EXE`)

All addresses below are verified virtual addresses in `MFW-FREE.EXE` (`ImageBase = 0x400000`, `CODE` section `0x401000..0x5A4A98`), disassembled via `objdump -d`:

- **`0x55da74` — Free-Edition Capacity Check**:
  - Loads total code count (`uint16` via `[0x5a9ecc]`), computes `remaining = 0x50 - totalCodes` (`80 - totalCodes` at `0x55da9f`).
  - If `remaining < 1` (`0x55daad`): blocks adding new events/resources ("has reached its maximum capacity").
  - If `remaining < 0x10` (`16`, `0x55daf0`) and (`remaining % 5 == 0 || remaining < 6`): displays capacity warning.
  - **OpenMimo**: Artificial 80-code cap removed (supports unlimited codes; tested beyond 100+ codes).
- **`0x5877e8` — Tentative / Free File-Save Limit**:
  - Checks edition flags `[0x5a9eb8]` & `[0x5a9d2c]` and blocks unlimited disk saves in tentative/free editions (XREF `0x5879c3` to string `0x587d34`).
  - **OpenMimo**: Unlimited saves & exports.
- **`0x4e8348` — Manual Room Assignment Capacity (`TRoomsDlg`)**:
  - Sums `assignedRooms` (`[0x5aa228]`, uint16) + `emptySlots` (`[0x5a9f60]`, uint16) and enforces `<= 0xFFFF` (`65535` slots, `0x4e8399`).
  - **OpenMimo**: Unlimited manual room slots with real-time free-room availability and automatic room imitation.
- **`0x54a900` (`MaxWeeks`), `0x54aa70` (`MaxDays`), `0x54abbc` (`MaxPeriods`) — Non-Destructive Time Bounds Validation**:
  - Calls helper `0x49d1f8` to find the highest week/day/period currently containing allocations or scheduled slots, refusing to shrink limits below used indices ("cannot be smaller than %d").
  - **OpenMimo**: Enforced in Go backend `/api/settings` so shrinking weeks/days/periods never silently destroys scheduled data.
- **`0x57fe80..0x580060` — Gap Optimization & Delta Reporting**:
  - Tracks up to `0x2710` (`10000` moves, `0x57fe9f`) in the optimization change log.
  - Computes gap delta `gapsNow - gapsInitially` (`0x57ff9c..0x57ffa9`) and formats summary (`0x57ff48` -> `0x5808a8`).
- **`0x59b304` & `0x59bb4c` — `.mxt` Exchange Format Parser/Writer**:
  - Parses `[Weeklectures]`, `[Weekbookings]`, `[Tablelectures]`, `[Tablebookings]`, `[Manualrooms]` (`0x59ba88`) and `NoTimetables` (`0x59dead`).
- **`0x4bb4c4` — Impossible Combination / Conflict Check**:
  - Checks impossible resource overlaps from the active week forward (`0x4bb559` -> `0x4bb650`).

### Hard Constraints & Gap Formula
- **Hard Constraints**:
  1. No two events sharing a conflict-enforced resource (`noTimetables == false`) may overlap in `(week, day, period)`.
  2. No event may be scheduled in a slot where the event or any of its linked resources has a Booking.
  3. An event's scheduled slots in a week cannot exceed its allocated periods for that week (when `maxWeeks == 1`, all `planned` periods auto-allocate to week 1).
  4. Locked events are skipped unless `includeLocked` is checked.
  5. Respects configured `[minDay..maxDay]`, `[minPeriod..maxPeriod]`, `[fromWeek..toWeek]`, and block length `[minSeq..maxSeq]` (default `1..10` from `Default.mpf`).
- **Gap Calculation & Optimization** (`Default.mpf` `[Optimize]`, `0x57fe80`, `TOptimisationDlg` DFM):
  - For each resource with `noGaps == false` and each `(week, day)`: let `first` and `last` be the earliest and latest occupied periods on that day (optionally counting bookings if `bookingsWithGaps` is true). Daily gaps = number of empty slots strictly between `first` and `last`.
  - Weighted gap score = `sum(resource.dailyGaps * gapWeight[resource.category])`.
  - Optimizer runs iterative local search minimizing weighted gaps (deterministic `randSeed` default `38746` from `Default.mpf`, XREF `0x5644b2`).

## 5. Semantic Palette (Modernized from `Default.mpf`)

Original `Default.mpf` Delphi `TColor` (`0xBBGGRR`) values converted to RGB:
- `clCourse` = `#ffcf2f`, `fcCourse` = `#000080` (Amber/Gold for Events)
- `clComp` = `#6699cc`, `fcComp` = `#ffffff` (Steel Blue for Resources)
- `clWeeks` = `#ffffcc` (Warm tint for Weeks grid)
- `clTTGrid` = `#ccffcc` (Mint tint for Timetables grid)
- `fcOk` = `#339933` (Green for OK/Available/Scheduled), `fcNotOk` = `#ff0000` (Red for Conflict/Over-limit)
- `clOldWeeks` = `#cc6600` (Orange for Protected/Locked weeks)
