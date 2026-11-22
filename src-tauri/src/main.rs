#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use std::io::{BufRead, BufReader};
use std::path::PathBuf;
use std::process::{Child, Command, Stdio};
use std::sync::{Arc, Mutex};
use tauri::{
    menu::{Menu, MenuItem, PredefinedMenuItem, Submenu},
    AppHandle, Manager, RunEvent, WebviewUrl, WebviewWindowBuilder, Wry,
};

struct BackendState {
    url: String,
    _child: Arc<Mutex<Option<Child>>>,
}

#[tauri::command]
fn get_backend_url(state: tauri::State<'_, BackendState>) -> String {
    state.url.clone()
}

fn detect_system_lang() -> String {
    for key in ["LC_ALL", "LC_MESSAGES", "LANG"] {
        if let Ok(val) = std::env::var(key) {
            let low = val.to_lowercase();
            if low.starts_with("en") {
                return "en".to_string();
            }
            if low.starts_with("es") {
                return "es".to_string();
            }
        }
    }
    "es".to_string()
}

/// Builds the native OS menu bar (macOS system menu bar & Linux GTK/KDE Global Menu via DBus).
#[cfg(any(target_os = "macos", target_os = "linux"))]
fn build_native_menu(app: &AppHandle, lang: &str) -> tauri::Result<Menu<Wry>> {
    let en = lang.eq_ignore_ascii_case("en");

    // File / Archivo
    let file_home = MenuItem::with_id(
        app,
        "file:home",
        if en { "Go to Home" } else { "Ir a Inicio" },
        true,
        Some("CmdOrCtrl+H"),
    )?;
    let file_new = MenuItem::with_id(
        app,
        "file:new",
        if en { "New Schedule..." } else { "Nuevo horario..." },
        true,
        Some("CmdOrCtrl+N"),
    )?;
    let file_open = MenuItem::with_id(
        app,
        "file:open",
        if en { "Open / Import (.omimo, .mfw, .mxt)..." } else { "Abrir / Importar (.omimo, .mfw, .mxt)..." },
        true,
        Some("CmdOrCtrl+O"),
    )?;
    let file_sample_design = MenuItem::with_id(
        app,
        "sample:design_school",
        if en { "Sample: Faculty of Arts & Media" } else { "Ejemplo: Facultad de Artes y Medios" },
        true,
        None::<&str>,
    )?;
    let file_sample_tiny = MenuItem::with_id(
        app,
        "sample:tiny_tutorial",
        if en { "Sample: Mini Tutorial" } else { "Ejemplo: Mini Tutorial" },
        true,
        None::<&str>,
    )?;
    let file_samples = Submenu::with_items(
        app,
        if en { "Sample Schedules" } else { "Horarios de prueba" },
        true,
        &[&file_sample_design, &file_sample_tiny],
    )?;
    let file_export = MenuItem::with_id(
        app,
        "file:export",
        if en { "Publish Web / Export..." } else { "Publicar Web / Exportar..." },
        true,
        Some("CmdOrCtrl+E"),
    )?;
    let file_close = MenuItem::with_id(
        app,
        "file:close_tab",
        if en { "Close Tab" } else { "Cerrar pestaña" },
        true,
        Some("CmdOrCtrl+W"),
    )?;
    let sep_f1 = PredefinedMenuItem::separator(app)?;
    let sep_f2 = PredefinedMenuItem::separator(app)?;
    let file_menu = Submenu::with_items(
        app,
        if en { "File" } else { "Archivo" },
        true,
        &[
            &file_home,
            &sep_f1,
            &file_new,
            &file_open,
            &file_samples,
            &file_export,
            &sep_f2,
            &file_close,
        ],
    )?;

    // Edit / Editar
    let edit_undo = MenuItem::with_id(
        app,
        "edit:undo",
        if en { "Undo" } else { "Deshacer" },
        true,
        Some("CmdOrCtrl+Z"),
    )?;
    let edit_redo = MenuItem::with_id(
        app,
        "edit:redo",
        if en { "Redo" } else { "Rehacer" },
        true,
        Some("Shift+CmdOrCtrl+Z"),
    )?;
    let sep_e1 = PredefinedMenuItem::separator(app)?;
    let edit_auto = MenuItem::with_id(
        app,
        "edit:auto_alloc",
        if en { "Auto-allocate Weeks" } else { "Auto-asignar semanas" },
        true,
        None::<&str>,
    )?;
    let sep_e2 = PredefinedMenuItem::separator(app)?;
    let edit_cut = PredefinedMenuItem::cut(app, None)?;
    let edit_copy = PredefinedMenuItem::copy(app, None)?;
    let edit_paste = PredefinedMenuItem::paste(app, None)?;
    let edit_sel = PredefinedMenuItem::select_all(app, None)?;
    let edit_menu = Submenu::with_items(
        app,
        if en { "Edit" } else { "Editar" },
        true,
        &[
            &edit_undo, &edit_redo, &sep_e1, &edit_auto, &sep_e2, &edit_cut, &edit_copy,
            &edit_paste, &edit_sel,
        ],
    )?;

    // View / Ver
    let view_home = MenuItem::with_id(
        app,
        "file:home",
        if en { "Home" } else { "Inicio" },
        true,
        None::<&str>,
    )?;
    let sep_v0 = PredefinedMenuItem::separator(app)?;
    let view_res = MenuItem::with_id(
        app,
        "view:resources",
        if en { "Resources & Teachers" } else { "Recursos y Profesores" },
        true,
        Some("CmdOrCtrl+1"),
    )?;
    let view_crs = MenuItem::with_id(
        app,
        "view:courses",
        if en { "Classes / Sessions" } else { "Sesiones / Asignaturas" },
        true,
        Some("CmdOrCtrl+2"),
    )?;
    let view_wks = MenuItem::with_id(
        app,
        "view:weeks",
        if en { "Weeks" } else { "Semanas" },
        true,
        Some("CmdOrCtrl+3"),
    )?;
    let view_tt = MenuItem::with_id(
        app,
        "view:timetables",
        if en { "Timetables" } else { "Horarios" },
        true,
        Some("CmdOrCtrl+4"),
    )?;
    let view_guides = MenuItem::with_id(
        app,
        "view:guides",
        if en { "Teaching Guides" } else { "Guías Docentes" },
        true,
        Some("CmdOrCtrl+5"),
    )?;
    let view_scripts = MenuItem::with_id(
        app,
        "view:scripts",
        if en { "Script Library" } else { "Librería de Scripts" },
        true,
        Some("CmdOrCtrl+6"),
    )?;
    let sep_v1 = PredefinedMenuItem::separator(app)?;
    let view_reload = MenuItem::with_id(
        app,
        "view:reload",
        if en { "Reload Interface" } else { "Recargar interfaz" },
        true,
        Some("CmdOrCtrl+R"),
    )?;
    let view_menu = Submenu::with_items(
        app,
        if en { "View" } else { "Ver" },
        true,
        &[&view_home, &sep_v0, &view_res, &view_crs, &view_wks, &view_tt, &view_guides, &view_scripts, &sep_v1, &view_reload],
    )?;

    // Tools / Herramientas
    let tools_opt = MenuItem::with_id(
        app,
        "tools:optimize",
        if en { "Optimise Timetables..." } else { "Optimizar horarios..." },
        true,
        Some("CmdOrCtrl+Shift+O"),
    )?;
    let tools_rooms = MenuItem::with_id(
        app,
        "tools:rooms",
        if en { "Select Rooms Manually..." } else { "Asignar salas manualmente..." },
        true,
        None::<&str>,
    )?;
    let tools_guides = MenuItem::with_id(
        app,
        "tools:guides",
        if en { "Teaching Guides & Astro Export..." } else { "Guías Docentes y exportar Astro..." },
        true,
        Some("CmdOrCtrl+G"),
    )?;
    let tools_collab = MenuItem::with_id(
        app,
        "tools:collab",
        if en { "Share / Join Real-Time (IPFS)..." } else { "Compartir / Unirse en tiempo real (IPFS)..." },
        true,
        None::<&str>,
    )?;
    let sep_t1 = PredefinedMenuItem::separator(app)?;
    let tools_stats = MenuItem::with_id(
        app,
        "tools:stats",
        if en { "Statistics & Health Check" } else { "Estadísticas y diagnóstico" },
        true,
        Some("CmdOrCtrl+I"),
    )?;
    let tools_mcp = MenuItem::with_id(
        app,
        "tools:mcp",
        if en { "MCP Server Configuration..." } else { "Configurar servidor MCP..." },
        true,
        None::<&str>,
    )?;
    let tools_menu = Submenu::with_items(
        app,
        if en { "Tools" } else { "Herramientas" },
        true,
        &[&tools_opt, &tools_rooms, &tools_guides, &tools_collab, &sep_t1, &tools_stats, &tools_mcp],
    )?;

    // Options / Configuración
    let opt_settings = MenuItem::with_id(
        app,
        "options:settings",
        if en { "Settings (Limits, Bells & Categories)..." } else { "Configuración (Límites, bloques y categorías)..." },
        true,
        Some("CmdOrCtrl+,"),
    )?;
    let opt_lang = MenuItem::with_id(
        app,
        "options:lang",
        if en { "Idioma: Cambiar a Español (ES)" } else { "Language: Switch to English (EN)" },
        true,
        None::<&str>,
    )?;
    let opt_menu = Submenu::with_items(
        app,
        if en { "Settings" } else { "Configuración" },
        true,
        &[&opt_settings, &opt_lang],
    )?;

    // Reports / Informes
    let rep_open = MenuItem::with_id(
        app,
        "reports:open",
        if en { "Publish Web Timetables / Print..." } else { "Publicar Horarios Web / Imprimir..." },
        true,
        Some("CmdOrCtrl+P"),
    )?;
    let rep_menu = Submenu::with_items(
        app,
        if en { "Reports" } else { "Informes" },
        true,
        &[&rep_open],
    )?;

    #[cfg(target_os = "macos")]
    {
        let app_about = PredefinedMenuItem::about(app, None, None)?;
        let app_updates = MenuItem::with_id(
            app,
            "app:check_updates",
            if en { "Check for Updates..." } else { "Buscar actualizaciones..." },
            true,
            None::<&str>,
        )?;
        let app_sep1 = PredefinedMenuItem::separator(app)?;
        let app_services = PredefinedMenuItem::services(app, None)?;
        let app_sep2 = PredefinedMenuItem::separator(app)?;
        let app_hide = PredefinedMenuItem::hide(app, None)?;
        let app_hide_others = PredefinedMenuItem::hide_others(app, None)?;
        let app_show_all = PredefinedMenuItem::show_all(app, None)?;
        let app_sep3 = PredefinedMenuItem::separator(app)?;
        let app_quit = PredefinedMenuItem::quit(app, None)?;
        let app_submenu = Submenu::with_items(
            app,
            "OpenMimo",
            true,
            &[
                &app_about,
                &app_updates,
                &app_sep1,
                &app_services,
                &app_sep2,
                &app_hide,
                &app_hide_others,
                &app_show_all,
                &app_sep3,
                &app_quit,
            ],
        )?;
        Menu::with_items(
            app,
            &[
                &app_submenu,
                &file_menu,
                &edit_menu,
                &view_menu,
                &tools_menu,
                &opt_menu,
                &rep_menu,
            ],
        )
    }

    #[cfg(target_os = "linux")]
    {
        Menu::with_items(
            app,
            &[&file_menu, &edit_menu, &view_menu, &tools_menu, &opt_menu, &rep_menu],
        )
    }
}

#[tauri::command]
fn set_app_language(app: AppHandle, lang: String) -> Result<(), String> {
    if let Some(win) = app.get_webview_window("main") {
        let _ = win.set_title("OpenMimo");
    }
    #[cfg(any(target_os = "macos", target_os = "linux"))]
    {
        if let Ok(menu) = build_native_menu(&app, &lang) {
            let _ = app.set_menu(menu);
        }
    }
    Ok(())
}

fn find_backend_binary() -> PathBuf {
    let bin_name = if cfg!(windows) {
        "openmimo-backend.exe"
    } else {
        "openmimo-backend"
    };

    let mut candidates = Vec::new();
    if let Ok(exe) = std::env::current_exe() {
        if let Some(dir) = exe.parent() {
            candidates.push(dir.join(bin_name));
            candidates.push(dir.join("backend").join(bin_name));
            candidates.push(dir.join("../Resources/backend").join(bin_name));
            candidates.push(dir.join("../lib/OpenMimo/backend").join(bin_name));
            candidates.push(dir.join("../lib/openmimo/backend").join(bin_name));
            candidates.push(dir.join("../backend").join(bin_name));
            candidates.push(dir.join("../../backend").join(bin_name));
            candidates.push(dir.join("../../../backend").join(bin_name));
        }
    }
    candidates.push(PathBuf::from("backend").join(bin_name));
    candidates.push(PathBuf::from("app/backend").join(bin_name));
    candidates.push(PathBuf::from("../backend").join(bin_name));

    for cand in &candidates {
        if cand.exists() {
            return cand.clone();
        }
    }
    PathBuf::from(bin_name)
}

fn spawn_go_backend() -> Result<(String, Child), String> {
    let bin = find_backend_binary();
    let mut child = Command::new(&bin)
        .arg("-port")
        .arg("0")
        .stdout(Stdio::piped())
        .stderr(Stdio::inherit())
        .spawn()
        .map_err(|e| format!("Failed to start Go backend {:?}: {}", bin, e))?;

    let stdout = child
        .stdout
        .take()
        .ok_or_else(|| "Failed to capture backend stdout".to_string())?;
    let mut reader = BufReader::new(stdout);
    let mut line = String::new();
    reader
        .read_line(&mut line)
        .map_err(|e| format!("Failed reading backend port: {}", e))?;

    let port = line
        .trim()
        .strip_prefix("OPENMIMO_PORT=")
        .ok_or_else(|| format!("Unexpected backend banner: {:?}", line))?
        .to_string();

    Ok((format!("http://127.0.0.1:{}", port), child))
}

fn parse_cli_arg(flag: &str) -> Option<String> {
    let args: Vec<String> = std::env::args().collect();
    let eq_prefix = format!("{}=", flag);
    let mut i = 1;
    while i < args.len() {
        if args[i] == flag && i + 1 < args.len() {
            return Some(args[i + 1].clone());
        }
        if let Some(p) = args[i].strip_prefix(&eq_prefix) {
            return Some(p.to_string());
        }
        i += 1;
    }
    None
}

fn has_cli_flag(flag: &str) -> bool {
    std::env::args().skip(1).any(|a| a == flag)
}

#[cfg(target_os = "macos")]
mod mac_native {
    use std::ffi::{c_double, c_int, c_void, CString};
    use std::process::Child;
    use std::sync::atomic::{AtomicI32, Ordering};
    use std::sync::{Arc, Mutex};

    extern "C" {
        fn openmimo_center_traffic_lights(ns_window_ptr: *mut c_void, header_height: c_double);
        fn openmimo_take_snapshot(
            wk_webview_ptr: *mut c_void,
            out_path: *const std::ffi::c_char,
            cb: extern "C" fn(c_int),
        );
    }

    static SNAP_STATUS: AtomicI32 = AtomicI32::new(-1);
    static BACKEND_CHILD: Mutex<Option<Arc<Mutex<Option<Child>>>>> = Mutex::new(None);

    extern "C" fn on_snapshot_done(status: c_int) {
        SNAP_STATUS.store(status, Ordering::SeqCst);
        if let Ok(guard) = BACKEND_CHILD.lock() {
            if let Some(child_arc) = guard.as_ref() {
                if let Ok(mut ch) = child_arc.lock() {
                    if let Some(mut c) = ch.take() {
                        let _ = c.kill();
                        let _ = c.wait();
                    }
                }
            }
        }
        std::process::exit(if status == 0 { 0 } else { 1 });
    }

    pub fn center_traffic_lights(window: &tauri::WebviewWindow, header_height: f64) {
        if let Ok(ptr) = window.ns_window() {
            unsafe {
                openmimo_center_traffic_lights(ptr as *mut c_void, header_height);
            }
        }
    }

    pub fn trigger_snapshot(
        window: tauri::WebviewWindow,
        out_path: String,
        child_arc: Arc<Mutex<Option<Child>>>,
    ) {
        if let Ok(mut g) = BACKEND_CHILD.lock() {
            *g = Some(child_arc);
        }
        let _ = window.with_webview(move |wv| {
            let c_path = CString::new(out_path).expect("valid screenshot path");
            unsafe {
                openmimo_take_snapshot(wv.inner() as *mut c_void, c_path.as_ptr(), on_snapshot_done);
            }
        });
    }
}

fn main() {
    if has_cli_flag("--mcp") {
        let bin = find_backend_binary();
        let status = Command::new(&bin)
            .arg("-mcp")
            .stdin(Stdio::inherit())
            .stdout(Stdio::inherit())
            .stderr(Stdio::inherit())
            .status()
            .unwrap_or_else(|e| {
                eprintln!("Failed to launch openmimo MCP backend {:?}: {}", bin, e);
                std::process::exit(1);
            });
        std::process::exit(status.code().unwrap_or(0));
    }

    let screenshot_path = parse_cli_arg("--screenshot");
    let init_view = parse_cli_arg("--view");
    let init_modal = parse_cli_arg("--modal");
    let init_lang = parse_cli_arg("--lang");
    let is_screenshot = screenshot_path.is_some();

    let (backend_url, child) = match spawn_go_backend() {
        Ok(pair) => pair,
        Err(e) => {
            eprintln!("openmimo backend error: {}", e);
            std::process::exit(1);
        }
    };
    let child_arc = Arc::new(Mutex::new(Some(child)));
    let child_for_exit = child_arc.clone();
    let child_for_snap = child_arc.clone();

    let use_native_menu = cfg!(any(target_os = "macos", target_os = "linux"));
    let is_mac_csd = cfg!(target_os = "macos");
    let platform = std::env::consts::OS;
    let start_lang = init_lang.clone().unwrap_or_else(detect_system_lang);

    let view_js = match &init_view {
        Some(v) => format!("{:?}", v),
        None => "null".to_string(),
    };
    let modal_js = match &init_modal {
        Some(m) => format!("{:?}", m),
        None => "null".to_string(),
    };
    let lang_js = match &init_lang {
        Some(l) => format!("{:?}", l),
        None => format!("{:?}", start_lang),
    };

    let init_script = format!(
        "window.__OPENMIMO_API__ = {:?}; window.__OPENMIMO_SCREENSHOT__ = {}; window.__OPENMIMO_INIT_VIEW__ = {}; window.__OPENMIMO_INIT_MODAL__ = {}; window.__OPENMIMO_INIT_LANG__ = {}; window.__OPENMIMO_NATIVE_MENU__ = {}; window.__OPENMIMO_MAC_CSD__ = {}; window.__OPENMIMO_PLATFORM__ = {:?};",
        backend_url, is_screenshot, view_js, modal_js, lang_js, use_native_menu, is_mac_csd, platform
    );

    let app = tauri::Builder::default()
        .manage(BackendState {
            url: backend_url,
            _child: child_arc,
        })
        .invoke_handler(tauri::generate_handler![get_backend_url, set_app_language])
        .setup(move |app| {
            #[cfg(any(target_os = "macos", target_os = "linux"))]
            {
                let menu = build_native_menu(app.handle(), &start_lang)?;
                app.set_menu(menu)?;
            }

            let mut builder =
                WebviewWindowBuilder::new(app, "main", WebviewUrl::App("index.html".into()))
                    .title("OpenMimo")
                    .inner_size(1340.0, 840.0)
                    .min_inner_size(820.0, 600.0)
                    .center()
                    .initialization_script(&init_script);

            #[cfg(target_os = "macos")]
            {
                builder = builder
                    .title_bar_style(tauri::TitleBarStyle::Overlay)
                    .hidden_title(true);
            }

            if is_screenshot {
                builder = builder.focused(false);
            }

            let window = builder.build()?;

            #[cfg(target_os = "macos")]
            {
                // Vertically center Close / Minimize / Zoom buttons inside our 48px CSD header bar
                mac_native::center_traffic_lights(&window, 48.0);
            }

            if let Some(out_path) = screenshot_path {
                let win_clone = window.clone();
                let child_clone = child_for_snap.clone();
                std::thread::spawn(move || {
                    std::thread::sleep(std::time::Duration::from_millis(2200));
                    #[cfg(target_os = "macos")]
                    {
                        let win_main = win_clone.clone();
                        let _ = win_clone.run_on_main_thread(move || {
                            mac_native::trigger_snapshot(win_main, out_path, child_clone);
                        });
                    }
                    #[cfg(not(target_os = "macos"))]
                    {
                        let _ = (win_clone, out_path, child_clone);
                        std::process::exit(0);
                    }
                });
            }

            Ok(())
        })
        .on_menu_event(|app, event| {
            let id = event.id().as_ref();
            if let Some(win) = app.get_webview_window("main") {
                let js = format!(
                    "window.__openmimoMenuAction && window.__openmimoMenuAction({:?});",
                    id
                );
                let _ = win.eval(&js);
            }
        })
        .build(tauri::generate_context!())
        .expect("error while building openmimo Tauri application");

    app.run(move |_app_handle, event| {
        if let RunEvent::Exit = event {
            if let Ok(mut ch) = child_for_exit.lock() {
                if let Some(mut c) = ch.take() {
                    let _ = c.kill();
                    let _ = c.wait();
                }
            }
        }
    });
}
