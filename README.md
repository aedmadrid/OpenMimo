# OpenMimo

[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)

**OpenMimo** es una aplicación libre de planificación de horarios académicos y guías docentes construida con **Tauri + React + Go**, compatible con archivos heredados de Mimosa (`.mfw` / `.mxt`) y con formato abierto propio `.omimo`.

## Características principales

- **Horarios sin límites**: 6 espacios de trabajo integrados (`Recursos`, `Sesiones`, `Semanas`, `Horarios`, `Guías` y `Scripts`).
- **Motor de Automejora y bloques consecutivos**: optimización multi-pasada con reglas configurables de mínimo y máximo de horas seguidas por curso.
- **Guías Docentes colaborativas + exportación Astro**: agrupa sesiones bajo asignaturas globales y publica un sitio web estático en Astro con un clic.
- **Librería de Scripts con IA**: escribe, ejecuta con soporte Deshacer, genera con IA y comparte scripts `.omimo-script.json`.
- **Servidor MCP integrado para Agentes**: controla horarios, recursos, restricciones y scripts desde cualquier agente compatible con MCP (stdio y HTTP).
- **Colaboración en tiempo real e IPFS**: comparte horarios con código `OMIPFS-...` editables mientras el equipo anfitrión mantiene el proyecto abierto.
- **Actualizaciones automáticas**: consulta nuevas versiones en `https://github.com/aedmadrid/openmimo/releases`.

## Estructura del proyecto

- `src/` — Interfaz React + TypeScript (Vite / Bun)
- `src-tauri/` — Host nativo Tauri v2 (macOS y Windows)
- `backend/` — Motor de horarios, importador `.mfw`/`.mxt`, MCP e IPFS en Go
- `landing/` — Página web oficial de presentación construida con Astro

## Licencia

Distribuido bajo licencia **GNU General Public License v3.0 (GPL-3.0-or-later)**. Consulta el archivo [LICENSE](LICENSE).
